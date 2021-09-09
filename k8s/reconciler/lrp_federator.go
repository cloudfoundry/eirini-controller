package reconciler

import (
	"context"

	eiriniv1 "code.cloudfoundry.org/eirini-controller/pkg/apis/eirini/v1"
	"code.cloudfoundry.org/lager"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	rspv1 "sigs.k8s.io/kubefed/pkg/apis/scheduling/v1alpha1"
	"sigs.k8s.io/kubefed/pkg/kubefedctl/federate"
)

func NewLRPFederator(logger lager.Logger, client client.Client, dynamicClient dynamic.Interface) *LRPFederator {
	return &LRPFederator{
		logger:        logger,
		client:        client,
		dynamicClient: dynamicClient,
	}
}

type LRPFederator struct {
	logger        lager.Logger
	client        client.Client
	dynamicClient dynamic.Interface
}

func (r *LRPFederator) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	logger := r.logger.Session("reconcile-lrp-federator",
		lager.Data{
			"name":      request.Name,
			"namespace": request.Namespace,
		})

	lrp := &eiriniv1.LRP{}

	err := r.client.Get(ctx, request.NamespacedName, lrp)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Debug("lrp-not-found")

			return reconcile.Result{}, nil
		}

		logger.Error("failed-to-get-lrp", err)

		return reconcile.Result{}, errors.Wrap(err, "failed to get lrp")
	}

	err = r.do(ctx, lrp)
	if err != nil {
		logger.Error("failed-to-reconcile", err)
	}

	return reconcile.Result{}, err
}

func (r *LRPFederator) do(ctx context.Context, lrp *eiriniv1.LRP) error {
	updatedLRP := lrp.DeepCopy()
	if updatedLRP.Labels == nil {
		updatedLRP.Labels = make(map[string]string)
	}
	updatedLRP.Labels["kubefed.io/managed"] = "true"

	if err := r.client.Patch(ctx, updatedLRP, client.MergeFrom(lrp)); err != nil {
		r.logger.Error("failed-to-patch-lrp", err, lager.Data{"namespace": lrp.Namespace})

		return errors.Wrap(err, "failed to patch lrp")
	}

	rsp := &rspv1.ReplicaSchedulingPreference{
		ObjectMeta: v1.ObjectMeta{
			Name:      lrp.Name,
			Namespace: lrp.Namespace,
		},
		Spec: rspv1.ReplicaSchedulingPreferenceSpec{
			TargetKind:                   "FederatedLRP",
			TotalReplicas:                int32(lrp.Spec.Replicas),
			Rebalance:                    true,
			IntersectWithClusterSelector: true,
		},
	}
	if err := r.client.Create(ctx, rsp); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			r.logger.Error("failed-to-create-rsp", err, lager.Data{"namespace": lrp.Namespace})

			return errors.Wrap(err, "failed to create RSP")
		}
	}

	lrpRes := schema.GroupVersionResource{Group: "eirini.cloudfoundry.org", Version: "v1", Resource: "lrps"}
	unstructuredLRP, err := r.dynamicClient.Resource(lrpRes).Namespace(lrp.Namespace).Get(ctx, lrp.Name, metav1.GetOptions{})
	if err != nil {
		r.logger.Error("failed-to-get-unstructured-lrp", err, lager.Data{"namespace": lrp.Namespace})

		return errors.Wrap(err, "failed to get unstructured lrp")

	}

	unstructuredFederatedLRP, err := federate.FederateResources([]*unstructured.Unstructured{unstructuredLRP})
	// TODO: set placement on the federated lrp to respect isolation segment
	if err != nil {
		r.logger.Error("failed-to-federate-lrp", err, lager.Data{"namespace": lrp.Namespace, "unstructuredfederatedlrp": unstructuredFederatedLRP})

		return errors.Wrap(err, "failed to federate lrp")
	}

	if lrp.Labels != nil && lrp.Labels["isolationSegment"] != "" {
		unstructured.SetNestedField(unstructuredFederatedLRP[0].Object, lrp.Labels["isolationSegment"], "spec", "placement", "clusterSelector", "matchLabels", "isolationSegment")
	}

	federatedLrpRes := schema.GroupVersionResource{Group: "types.kubefed.io", Version: "v1beta1", Resource: "federatedlrps"}
	_, err = r.dynamicClient.Resource(federatedLrpRes).Namespace(lrp.Namespace).Create(ctx, unstructuredFederatedLRP[0], metav1.CreateOptions{})
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			r.logger.Error("failed-to-create-federatedlrp", err, lager.Data{"namespace": lrp.Namespace})

			return errors.Wrap(err, "failed to create federatedlrp")
		}
	}

	return nil
}
