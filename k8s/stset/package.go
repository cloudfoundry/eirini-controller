package stset

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

const (
	AppSourceType = "APP"

	AnnotationAppName              = "workloads.cloudfoundry.org/application-name"
	AnnotationVersion              = "workloads.cloudfoundry.org/version"
	AnnotationAppID                = "workloads.cloudfoundry.org/application-id"
	AnnotationSpaceName            = "workloads.cloudfoundry.org/space-name"
	AnnotationOrgName              = "workloads.cloudfoundry.org/org-name"
	AnnotationOrgGUID              = "workloads.cloudfoundry.org/org-guid"
	AnnotationSpaceGUID            = "workloads.cloudfoundry.org/space-guid"
	AnnotationProcessGUID          = "workloads.cloudfoundry.org/process-guid"
	AnnotationLastReportedAppCrash = "workloads.cloudfoundry.org/last-reported-app-crash"
	AnnotationLastReportedLRPCrash = "workloads.cloudfoundry.org/last-reported-lrp-crash"

	LabelGUID        = "workloads.cloudfoundry.org/guid"
	LabelOrgGUID     = AnnotationOrgGUID
	LabelOrgName     = AnnotationOrgName
	LabelSpaceGUID   = AnnotationSpaceGUID
	LabelSpaceName   = AnnotationSpaceName
	LabelVersion     = "workloads.cloudfoundry.org/version"
	LabelAppGUID     = "workloads.cloudfoundry.org/app-guid"
	LabelProcessType = "workloads.cloudfoundry.org/process-type"
	LabelSourceType  = "workloads.cloudfoundry.org/source-type"

	ApplicationContainerName = "opi"

	PdbMinAvailableInstances          = 1
	PrivateRegistrySecretGenerateName = "private-registry-"
)
