package jobs

import (
	eiriniv1 "code.cloudfoundry.org/eirini-controller/pkg/apis/eirini/v1"
	batch "k8s.io/api/batch/v1"
)

func toTask(job batch.Job) *eiriniv1.Task {
	return &eiriniv1.Task{
		Spec: eiriniv1.TaskSpec{
			GUID: job.Labels[LabelGUID],
		},
	}
}
