package reconciler

import (
	"context"
	"fmt"
	"time"

	eiriniv1 "code.cloudfoundry.org/eirini-controller/pkg/apis/eirini/v1"
	"code.cloudfoundry.org/lager"
	exterrors "github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type Task struct {
	taskCrClient   TasksCrClient
	workloadClient TaskWorkloadClient
	logger         lager.Logger
	ttlSeconds     int
}

//counterfeiter:generate . TasksCrClient

type TasksCrClient interface {
	UpdateTaskStatus(context.Context, *eiriniv1.Task, eiriniv1.TaskStatus) error
	GetTask(context.Context, string, string) (*eiriniv1.Task, error)
}

//counterfeiter:generate . TaskWorkloadClient

type TaskWorkloadClient interface {
	Desire(ctx context.Context, task *eiriniv1.Task) error
	GetStatus(ctx context.Context, taskGUID string) (eiriniv1.TaskStatus, error)
	Delete(ctx context.Context, guid string) error
}

func NewTask(logger lager.Logger, taskCrClient TasksCrClient, workloadClient TaskWorkloadClient, ttlSeconds int) *Task {
	return &Task{
		taskCrClient:   taskCrClient,
		workloadClient: workloadClient,
		logger:         logger,
		ttlSeconds:     ttlSeconds,
	}
}

func (t *Task) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	logger := t.logger.Session("reconcile-task", lager.Data{"request": request})
	logger.Debug("start")

	task, err := t.taskCrClient.GetTask(ctx, request.NamespacedName.Namespace, request.NamespacedName.Name)
	if errors.IsNotFound(err) {
		logger.Debug("task-not-found")

		return reconcile.Result{}, nil
	}

	if err != nil {
		logger.Error("task-get-failed", err)

		return reconcile.Result{}, fmt.Errorf("could not fetch task: %w", err)
	}

	if taskHasCompleted(task.Status) {
		return t.handleExpiredTask(ctx, logger, task)
	}

	err = t.workloadClient.Desire(ctx, task)
	if err != nil && !errors.IsAlreadyExists(err) {
		logger.Error("desire-task-failed", err)

		return reconcile.Result{}, exterrors.Wrap(err, "failed to desire task")
	}

	status, err := t.workloadClient.GetStatus(ctx, task.Spec.GUID)
	if err != nil {
		logger.Error("failed-to-get-task-status", err)

		return reconcile.Result{}, exterrors.Wrap(err, "failed to get task status")
	}

	if err := t.taskCrClient.UpdateTaskStatus(ctx, task, status); err != nil {
		return reconcile.Result{}, exterrors.Wrap(err, "failed to update task status")
	}

	if taskHasCompleted(status) {
		return reconcile.Result{RequeueAfter: time.Duration(t.ttlSeconds) * time.Second}, nil
	}

	return reconcile.Result{}, nil
}

func (t *Task) handleExpiredTask(ctx context.Context, logger lager.Logger, task *eiriniv1.Task) (reconcile.Result, error) {
	if t.taskHasExpired(task) {
		logger.Debug("deleting-expired-task")

		err := t.workloadClient.Delete(ctx, task.Spec.GUID)
		if err != nil {
			return reconcile.Result{}, exterrors.Wrap(err, "failed to delete task")
		}
	}

	logger.Debug("task-already-completed")

	return reconcile.Result{}, nil
}

func taskHasCompleted(status eiriniv1.TaskStatus) bool {
	return status.EndTime != nil &&
		(status.ExecutionStatus == eiriniv1.TaskFailed ||
			status.ExecutionStatus == eiriniv1.TaskSucceeded)
}

func (t *Task) taskHasExpired(task *eiriniv1.Task) bool {
	ttlExpire := metav1.NewTime(time.Now().Add(-time.Duration(t.ttlSeconds) * time.Second))

	return task.Status.EndTime.Before(&ttlExpire)
}
