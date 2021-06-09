package stset

import (
	"context"

	"code.cloudfoundry.org/eirini-controller/k8s/utils"
	"code.cloudfoundry.org/eirini-controller/k8s/utils/dockerutils"
	eiriniv1 "code.cloudfoundry.org/eirini-controller/pkg/apis/eirini/v1"
	"code.cloudfoundry.org/eirini-controller/util"
	"code.cloudfoundry.org/lager"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

//counterfeiter:generate . SecretsClient
//counterfeiter:generate . StatefulSetCreator
//counterfeiter:generate . LRPToStatefulSetConverter
//counterfeiter:generate . PodDisruptionBudgetUpdater

type LRPToStatefulSetConverter interface {
	Convert(statefulSetName string, lrp *eiriniv1.LRP, privateRegistrySecret *corev1.Secret) (*appsv1.StatefulSet, error)
}

type SecretsClient interface {
	Create(ctx context.Context, namespace string, secret *corev1.Secret) (*corev1.Secret, error)
	SetOwner(ctx context.Context, secret *corev1.Secret, owner metav1.Object) (*corev1.Secret, error)
	Delete(ctx context.Context, namespace string, name string) error
}

type StatefulSetCreator interface {
	Create(ctx context.Context, namespace string, statefulSet *appsv1.StatefulSet) (*appsv1.StatefulSet, error)
}

type PodDisruptionBudgetUpdater interface {
	Update(ctx context.Context, stset *appsv1.StatefulSet, lrp *eiriniv1.LRP) error
}

type Desirer struct {
	logger                     lager.Logger
	secrets                    SecretsClient
	statefulSets               StatefulSetCreator
	lrpToStatefulSetConverter  LRPToStatefulSetConverter
	podDisruptionBudgetCreator PodDisruptionBudgetUpdater
	scheme                     *runtime.Scheme
}

func NewDesirer(
	logger lager.Logger,
	secrets SecretsClient,
	statefulSets StatefulSetCreator,
	lrpToStatefulSetConverter LRPToStatefulSetConverter,
	podDisruptionBudgetCreator PodDisruptionBudgetUpdater,
	scheme *runtime.Scheme,
) Desirer {
	return Desirer{
		logger:                     logger,
		secrets:                    secrets,
		statefulSets:               statefulSets,
		lrpToStatefulSetConverter:  lrpToStatefulSetConverter,
		podDisruptionBudgetCreator: podDisruptionBudgetCreator,
		scheme:                     scheme,
	}
}

func (d *Desirer) Desire(ctx context.Context, lrp *eiriniv1.LRP) error {
	logger := d.logger.Session("desire", lager.Data{"guid": lrp.Spec.GUID, "version": lrp.Spec.Version, "namespace": lrp.Namespace})

	statefulSetName, err := utils.GetStatefulsetName(lrp)
	if err != nil {
		return err
	}

	privateRegistrySecret, err := d.createRegistryCredsSecretIfRequired(ctx, lrp)
	if err != nil {
		return err
	}

	st, err := d.lrpToStatefulSetConverter.Convert(statefulSetName, lrp, privateRegistrySecret)
	if err != nil {
		return err
	}

	st.Namespace = lrp.Namespace

	if err = ctrl.SetControllerReference(lrp, st, d.scheme); err != nil {
		return errors.Wrap(err, "failed to set controller reference")
	}

	stSet, err := d.statefulSets.Create(ctx, lrp.Namespace, st)
	if err != nil {
		var statusErr *k8serrors.StatusError
		if errors.As(err, &statusErr) && statusErr.Status().Reason == metav1.StatusReasonAlreadyExists {
			logger.Debug("statefulset-already-exists", lager.Data{"error": err.Error()})

			return nil
		}

		return d.cleanupAndError(ctx, errors.Wrap(err, "failed to create statefulset"), privateRegistrySecret)
	}

	if err := d.setSecretOwner(ctx, privateRegistrySecret, stSet); err != nil {
		logger.Error("failed-to-set-owner-to-the-registry-secret", err)

		return errors.Wrap(err, "failed to set owner to the registry secret")
	}

	if err := d.podDisruptionBudgetCreator.Update(ctx, stSet, lrp); err != nil {
		logger.Error("failed-to-create-pod-disruption-budget", err)

		return errors.Wrap(err, "failed to create pod disruption budget")
	}

	return nil
}

func (d *Desirer) setSecretOwner(ctx context.Context, privateRegistrySecret *corev1.Secret, stSet *appsv1.StatefulSet) error {
	if privateRegistrySecret == nil {
		return nil
	}

	_, err := d.secrets.SetOwner(ctx, privateRegistrySecret, stSet)

	return err
}

func (d *Desirer) createRegistryCredsSecretIfRequired(ctx context.Context, lrp *eiriniv1.LRP) (*corev1.Secret, error) {
	if lrp.Spec.PrivateRegistry == nil {
		return nil, nil
	}

	secret, err := generateRegistryCredsSecret(lrp)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate private registry secret for statefulset")
	}

	secret, err = d.secrets.Create(ctx, lrp.Namespace, secret)

	return secret, errors.Wrap(err, "failed to create private registry secret for statefulset")
}

func (d *Desirer) cleanupAndError(ctx context.Context, stsetCreationError error, privateRegistrySecret *corev1.Secret) error {
	resultError := multierror.Append(nil, stsetCreationError)

	if privateRegistrySecret != nil {
		err := d.secrets.Delete(ctx, privateRegistrySecret.Namespace, privateRegistrySecret.Name)
		if err != nil {
			resultError = multierror.Append(resultError, errors.Wrap(err, "failed to cleanup registry secret"))
		}
	}

	return resultError
}

func generateRegistryCredsSecret(lrp *eiriniv1.LRP) (*corev1.Secret, error) {
	dockerConfig := dockerutils.NewDockerConfig(
		util.ParseImageRegistryHost(lrp.Spec.Image),
		lrp.Spec.PrivateRegistry.Username,
		lrp.Spec.PrivateRegistry.Password,
	)

	dockerConfigJSON, err := dockerConfig.JSON()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate privete registry config")
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: PrivateRegistrySecretGenerateName,
		},
		Type: corev1.SecretTypeDockerConfigJson,
		StringData: map[string]string{
			dockerutils.DockerConfigKey: dockerConfigJSON,
		},
	}, nil
}
