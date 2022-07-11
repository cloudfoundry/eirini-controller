package jobs

import (
	"context"
	"fmt"

	eiriniv1 "code.cloudfoundry.org/eirini-controller/pkg/apis/eirini/v1"
	"code.cloudfoundry.org/lager"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StatusGetter struct {
	logger    lager.Logger
	k8sClient client.Client
}

func NewStatusGetter(logger lager.Logger, k8sClient client.Client) *StatusGetter {
	return &StatusGetter{
		logger:    logger,
		k8sClient: k8sClient,
	}
}

func (s *StatusGetter) GetStatusConditions(ctx context.Context, job *batchv1.Job) ([]metav1.Condition, error) {
	logger := s.logger.Session("get status condtions", lager.Data{"name": job.Name, "namespace": job.Namespace})
	conditions := []metav1.Condition{
		{
			Type:    eiriniv1.TaskInitializedConditionType,
			Status:  metav1.ConditionTrue,
			Reason:  "job_created",
			Message: "Job created",
		},
	}

	if job.Status.StartTime == nil {
		return conditions, nil
	}

	conditions = append(conditions, metav1.Condition{
		Type:               eiriniv1.TaskStartedConditionType,
		Status:             metav1.ConditionTrue,
		LastTransitionTime: *job.Status.StartTime,
		Reason:             "job_started",
		Message:            "Job started",
	})

	if job.Status.Succeeded > 0 && job.Status.CompletionTime != nil {
		conditions = append(conditions, metav1.Condition{
			Type:               eiriniv1.TaskSucceededConditionType,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: *job.Status.CompletionTime,
			Reason:             "job_succeeded",
			Message:            "Job succeeded",
		})
	}

	lastFailureTimestamp := getLastFailureTimestamp(job.Status)
	if job.Status.Failed > 0 && lastFailureTimestamp != nil {
		terminationState, err := s.getFailedContainerStatus(ctx, job)
		if err != nil {
			logger.Error("failed to get container status", err)

			return nil, fmt.Errorf("failed to get container status: %w", err)
		}

		conditions = append(conditions, metav1.Condition{
			Type:               eiriniv1.TaskFailedConditionType,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: *lastFailureTimestamp,
			Reason:             terminationState.Reason,
			Message:            fmt.Sprintf("Failed with exit code: %d", terminationState.ExitCode),
		})
	}

	return conditions, nil
}

func (s *StatusGetter) getFailedContainerStatus(ctx context.Context, job *batchv1.Job) (*corev1.ContainerStateTerminated, error) {
	var jobPods corev1.PodList
	if err := s.k8sClient.List(ctx, &jobPods, client.InNamespace(job.Namespace), client.MatchingLabels{"job-name": job.Name}); err != nil {
		return nil, err
	}

	if len(jobPods.Items) > 1 {
		return nil, fmt.Errorf("found more than one pod for job %s:%s", job.Namespace, job.Name)
	}

	if len(jobPods.Items) == 0 {
		return nil, fmt.Errorf("no pods found for job %s:%s", job.Namespace, job.Name)
	}

	jobPod := jobPods.Items[0]

	for _, containerStatus := range jobPod.Status.ContainerStatuses {
		if containerStatus.Name != jobPod.Annotations[AnnotationTaskContainerName] {
			continue
		}

		if containerStatus.State.Terminated == nil {
			return nil, fmt.Errorf("no terminated state found for job %s:%s", job.Namespace, job.Name)
		}

		return containerStatus.State.Terminated, nil
	}

	return nil, fmt.Errorf("no task container found for job %s:%s", job.Namespace, job.Name)
}

func getLastFailureTimestamp(jobStatus batchv1.JobStatus) *metav1.Time {
	var lastFailure *metav1.Time

	for _, condition := range jobStatus.Conditions {
		condition := condition
		if condition.Type != batchv1.JobFailed {
			continue
		}

		if lastFailure == nil || condition.LastTransitionTime.After(lastFailure.Time) {
			lastFailure = &condition.LastTransitionTime
		}
	}

	return lastFailure
}
