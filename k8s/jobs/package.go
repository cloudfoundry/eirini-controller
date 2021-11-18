package jobs

import (
	"code.cloudfoundry.org/eirini-controller/k8s/stset"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

const (
	TaskSourceType = "TASK"

	AnnotationGUID                        = "workloads.cloudfoundry.org/guid"
	AnnotationAppName                     = stset.AnnotationAppName
	AnnotationAppID                       = stset.AnnotationAppID
	AnnotationOrgName                     = stset.AnnotationOrgName
	AnnotationOrgGUID                     = stset.AnnotationOrgGUID
	AnnotationSpaceName                   = stset.AnnotationSpaceName
	AnnotationSpaceGUID                   = stset.AnnotationSpaceGUID
	AnnotationTaskContainerName           = "workloads.cloudfoundry.org/opi-task-container-name"
	AnnotationTaskCompletionReportCounter = "workloads.cloudfoundry.org/task_completion_report_counter"
	AnnotationCCAckedTaskCompletion       = "workloads.cloudfoundry.org/cc_acked_task_completion"

	LabelGUID          = stset.LabelGUID
	LabelName          = "workloads.cloudfoundry.org/name"
	LabelAppGUID       = stset.LabelAppGUID
	LabelSourceType    = stset.LabelSourceType
	LabelTaskCompleted = "workloads.cloudfoundry.org/task_completed"

	TaskCompletedTrue                 = "true"
	PrivateRegistrySecretGenerateName = stset.PrivateRegistrySecretGenerateName
)
