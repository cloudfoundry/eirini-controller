package jobs

import (
	"context"

	"code.cloudfoundry.org/eirini-controller/k8s/utils/dockerutils"
	eiriniv1 "code.cloudfoundry.org/eirini-controller/pkg/apis/eirini/v1"
	"code.cloudfoundry.org/eirini-controller/util"
	"code.cloudfoundry.org/lager"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	batch "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

//counterfeiter:generate . TaskToJobConverter
//counterfeiter:generate . JobCreator
//counterfeiter:generate . SecretsClient

type TaskToJobConverter interface {
	Convert(*eiriniv1.Task, *corev1.Secret) *batch.Job
}

type JobCreator interface {
	Create(ctx context.Context, namespace string, job *batch.Job) (*batch.Job, error)
}

type SecretsClient interface {
	Create(ctx context.Context, namespace string, secret *corev1.Secret) (*corev1.Secret, error)
	SetOwner(ctx context.Context, secret *corev1.Secret, owner metav1.Object) (*corev1.Secret, error)
	Delete(ctx context.Context, namespace string, name string) error
}

type Desirer struct {
	logger             lager.Logger
	taskToJobConverter TaskToJobConverter
	jobCreator         JobCreator
	secrets            SecretsClient
	scheme             *runtime.Scheme
}

func NewDesirer(
	logger lager.Logger,
	taskToJobConverter TaskToJobConverter,
	jobCreator JobCreator,
	secretCreator SecretsClient,
	scheme *runtime.Scheme,
) Desirer {
	return Desirer{
		logger:             logger,
		taskToJobConverter: taskToJobConverter,
		jobCreator:         jobCreator,
		secrets:            secretCreator,
		scheme:             scheme,
	}
}

func (d *Desirer) Desire(ctx context.Context, task *eiriniv1.Task) error {
	logger := d.logger.Session("desire-task", lager.Data{"guid": task.Spec.GUID, "name": task.Name, "namespace": task.Namespace})

	var (
		err                   error
		privateRegistrySecret *corev1.Secret
	)

	if imageInPrivateRegistry(task) {
		privateRegistrySecret, err = d.createPrivateRegistrySecret(ctx, task.Namespace, task)
		if err != nil {
			return errors.Wrap(err, "failed to create task secret")
		}
	}

	job := d.taskToJobConverter.Convert(task, privateRegistrySecret)

	job.Namespace = task.Namespace

	if err = ctrl.SetControllerReference(task, job, d.scheme); err != nil {
		return errors.Wrap(err, "failed to set controller reference")
	}

	job, err = d.jobCreator.Create(ctx, task.Namespace, job)
	if err != nil {
		logger.Error("failed-to-create-job", err)

		return d.cleanupAndError(ctx, err, privateRegistrySecret)
	}

	if privateRegistrySecret != nil {
		_, err = d.secrets.SetOwner(ctx, privateRegistrySecret, job)
		if err != nil {
			return errors.Wrap(err, "failed-to-set-secret-ownership")
		}
	}

	return nil
}

func imageInPrivateRegistry(task *eiriniv1.Task) bool {
	return task.Spec.PrivateRegistry != nil && task.Spec.PrivateRegistry.Username != "" && task.Spec.PrivateRegistry.Password != ""
}

func (d *Desirer) createPrivateRegistrySecret(ctx context.Context, namespace string, task *eiriniv1.Task) (*corev1.Secret, error) {
	secret := &corev1.Secret{}

	secret.GenerateName = PrivateRegistrySecretGenerateName
	secret.Type = corev1.SecretTypeDockerConfigJson

	dockerConfig := dockerutils.NewDockerConfig(
		util.ParseImageRegistryHost(task.Spec.Image),
		task.Spec.PrivateRegistry.Username,
		task.Spec.PrivateRegistry.Password,
	)

	dockerConfigJSON, err := dockerConfig.JSON()
	if err != nil {
		return nil, errors.Wrap(err, "failed-to-get-docker-config")
	}

	secret.StringData = map[string]string{
		dockerutils.DockerConfigKey: dockerConfigJSON,
	}

	return d.secrets.Create(ctx, namespace, secret)
}

func (d *Desirer) cleanupAndError(ctx context.Context, jobCreationError error, privateRegistrySecret *corev1.Secret) error {
	resultError := multierror.Append(nil, jobCreationError)

	if privateRegistrySecret != nil {
		err := d.secrets.Delete(ctx, privateRegistrySecret.Namespace, privateRegistrySecret.Name)
		if err != nil {
			resultError = multierror.Append(resultError, errors.Wrap(err, "failed to cleanup registry secret"))
		}
	}

	return resultError
}
