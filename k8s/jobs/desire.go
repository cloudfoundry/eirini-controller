package jobs

import (
	"context"

	eiriniv1 "code.cloudfoundry.org/eirini-controller/pkg/apis/eirini/v1"
	"code.cloudfoundry.org/lager"
	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//counterfeiter:generate . TaskToJobConverter
//counterfeiter:generate . JobCreator
//counterfeiter:generate . SecretsClient

type TaskToJobConverter interface {
	Convert(*eiriniv1.Task) *batchv1.Job
}

type JobCreator interface {
	Create(ctx context.Context, namespace string, job *batchv1.Job) (*batchv1.Job, error)
}

type SecretsClient interface {
	Create(ctx context.Context, namespace string, secret *corev1.Secret) (*corev1.Secret, error)
	SetOwner(ctx context.Context, secret *corev1.Secret, owner metav1.Object) (*corev1.Secret, error)
	Delete(ctx context.Context, namespace string, name string) error
}

type Desirer struct {
	logger             lager.Logger
	taskToJobConverter TaskToJobConverter
	client             client.Client
	scheme             *runtime.Scheme
}

func NewDesirer(
	logger lager.Logger,
	taskToJobConverter TaskToJobConverter,
	client client.Client,
	scheme *runtime.Scheme,
) *Desirer {
	return &Desirer{
		logger:             logger,
		taskToJobConverter: taskToJobConverter,
		client:             client,
		scheme:             scheme,
	}
}

func (d *Desirer) Desire(ctx context.Context, task *eiriniv1.Task) (*batchv1.Job, error) {
	logger := d.logger.Session("desire-task", lager.Data{"guid": task.Spec.GUID, "name": task.Name, "namespace": task.Namespace})

	job := d.taskToJobConverter.Convert(task)

	job.Namespace = task.Namespace

	if err := ctrl.SetControllerReference(task, job, d.scheme); err != nil {
		return nil, errors.Wrap(err, "failed to set controller reference")
	}

	if err := d.client.Create(ctx, job); err != nil {
		logger.Error("failed-to-create-job", err)

		return nil, err
	}

	return job, nil
}
