package reconciler

import (
	"context"

	"code.cloudfoundry.org/eirini-controller/k8s/utils"
	eiriniv1 "code.cloudfoundry.org/eirini-controller/pkg/apis/eirini/v1"
	"code.cloudfoundry.org/lager"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

//counterfeiter:generate . LRPWorkloadCLient
//counterfeiter:generate -o reconcilerfakes/fake_controller_runtime_client.go sigs.k8s.io/controller-runtime/pkg/client.Client
//counterfeiter:generate -o reconcilerfakes/fake_status_writer.go sigs.k8s.io/controller-runtime/pkg/client.StatusWriter
//counterfeiter:generate . LRPsCrClient

type LRPWorkloadCLient interface {
	Desire(ctx context.Context, lrp *eiriniv1.LRP) error
	Update(ctx context.Context, lrp *eiriniv1.LRP) error
	GetStatus(ctx context.Context, lrp *eiriniv1.LRP) (eiriniv1.LRPStatus, error)
}

type LRPsCrClient interface {
	UpdateLRPStatus(context.Context, *eiriniv1.LRP, eiriniv1.LRPStatus) error
	GetLRP(context.Context, string, string) (*eiriniv1.LRP, error)
}

func NewLRP(logger lager.Logger, lrpsCrClient LRPsCrClient, workloadClient LRPWorkloadCLient, statefulsetGetter StatefulSetGetter) *LRP {
	return &LRP{
		logger:            logger,
		lrpsCrClient:      lrpsCrClient,
		workloadClient:    workloadClient,
		statefulsetGetter: statefulsetGetter,
	}
}

type LRP struct {
	logger            lager.Logger
	lrpsCrClient      LRPsCrClient
	workloadClient    LRPWorkloadCLient
	statefulsetGetter StatefulSetGetter
}

func (r *LRP) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	logger := r.logger.Session("reconcile-lrp",
		lager.Data{
			"name":      request.NamespacedName.Name,
			"namespace": request.NamespacedName.Namespace,
		})

	lrp, err := r.lrpsCrClient.GetLRP(ctx, request.Namespace, request.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Error("lrp-not-found", err)

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

func (r *LRP) do(ctx context.Context, lrp *eiriniv1.LRP) error {
	stSetName, err := utils.GetStatefulsetName(lrp)
	if err != nil {
		return errors.Wrapf(err, "failed to determine statefulset name for lrp {%s}%s", lrp.Namespace, lrp.Name)
	}

	_, err = r.statefulsetGetter.Get(ctx, lrp.Namespace, stSetName)
	if apierrors.IsNotFound(err) {
		return errors.Wrap(r.workloadClient.Desire(ctx, lrp), "failed to desire lrp")
	}

	if err != nil {
		return errors.Wrap(err, "failed to get statefulSet")
	}

	var errs *multierror.Error

	err = r.updateStatus(ctx, lrp)
	errs = multierror.Append(errs, errors.Wrap(err, "failed to update lrp status"))

	err = r.workloadClient.Update(ctx, lrp)
	errs = multierror.Append(errs, errors.Wrap(err, "failed to update app"))

	return errs.ErrorOrNil()
}

func (r *LRP) updateStatus(ctx context.Context, lrp *eiriniv1.LRP) error {
	lrpStatus, err := r.workloadClient.GetStatus(ctx, lrp)
	if err != nil {
		return err
	}

	return r.lrpsCrClient.UpdateLRPStatus(ctx, lrp, lrpStatus)
}
