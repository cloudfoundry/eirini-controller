package jobs

import (
	"context"

	eiriniv1 "code.cloudfoundry.org/eirini-controller/pkg/apis/eirini/v1"
	"code.cloudfoundry.org/lager"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type StatusGetter struct {
	logger lager.Logger
}

func NewStatusGetter(logger lager.Logger) *StatusGetter {
	return &StatusGetter{
		logger: logger,
	}
}

func (s *StatusGetter) GetStatusConditions(ctx context.Context, job *batchv1.Job) []metav1.Condition {
	conditions := []metav1.Condition{
		{
			Type:    eiriniv1.TaskInitializedConditionType,
			Status:  metav1.ConditionTrue,
			Reason:  "job_created",
			Message: "Job created",
		},
	}

	if job.Status.StartTime == nil {
		return conditions
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
		conditions = append(conditions, metav1.Condition{
			Type:               eiriniv1.TaskFailedConditionType,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: *lastFailureTimestamp,
			Reason:             "job_failed",
			Message:            "Job failed",
		})
	}

	return conditions
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
