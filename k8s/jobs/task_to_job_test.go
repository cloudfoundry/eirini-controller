package jobs_test

import (
	"fmt"

	eirinictrl "code.cloudfoundry.org/eirini-controller"
	"code.cloudfoundry.org/eirini-controller/k8s/jobs"
	eiriniv1 "code.cloudfoundry.org/eirini-controller/pkg/apis/eirini/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	batch "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var _ = Describe("TaskToJob", func() {
	const (
		image          = "docker.png"
		taskGUID       = "task-123"
		serviceAccount = "service-account"
		registrySecret = "registry-secret"
	)

	var (
		job                               *batch.Job
		task                              *eiriniv1.Task
		allowAutomountServiceAccountToken bool
	)

	assertGeneralSpec := func(job *batch.Job) {
		automountServiceAccountToken := false
		ExpectWithOffset(1, job.Spec.Template.Spec.RestartPolicy).To(Equal(corev1.RestartPolicyNever))
		ExpectWithOffset(1, job.Spec.Template.Spec.AutomountServiceAccountToken).To(Equal(&automountServiceAccountToken))
	}

	assertContainer := func(container corev1.Container, name string) {
		Expect(container.Name).To(Equal(name))
		Expect(container.Image).To(Equal(image))
		Expect(container.ImagePullPolicy).To(Equal(corev1.PullAlways))

		Expect(container.Env).To(ContainElements(
			corev1.EnvVar{Name: "my-env-var", Value: "env"},
			corev1.EnvVar{Name: eirinictrl.EnvCFInstanceGUID, ValueFrom: expectedValFrom("metadata.uid")},
			corev1.EnvVar{Name: eirinictrl.EnvCFInstanceInternalIP, ValueFrom: expectedValFrom("status.podIP")},
			corev1.EnvVar{Name: eirinictrl.EnvCFInstanceIP, ValueFrom: expectedValFrom("status.hostIP")},
			corev1.EnvVar{Name: eirinictrl.EnvPodName, ValueFrom: expectedValFrom("metadata.name")},
			corev1.EnvVar{Name: eirinictrl.EnvCFInstanceAddr, Value: ""},
			corev1.EnvVar{Name: eirinictrl.EnvCFInstancePort, Value: ""},
			corev1.EnvVar{Name: eirinictrl.EnvCFInstancePorts, Value: "[]"},
		))
	}

	BeforeEach(func() {
		allowAutomountServiceAccountToken = false

		task = &eiriniv1.Task{
			Spec: eiriniv1.TaskSpec{
				Image:     image,
				Command:   []string{"/lifecycle/launch"},
				AppName:   "my-app",
				Name:      "task-name",
				AppGUID:   "my-app-guid",
				OrgName:   "my-org",
				SpaceName: "my-space",
				SpaceGUID: "space-id",
				OrgGUID:   "org-id",
				GUID:      taskGUID,
				Env: map[string]string{
					"my-env-var": "env",
				},
				MemoryMB:  1,
				CPUMillis: 2,
				DiskMB:    3,
			},
		}
	})

	JustBeforeEach(func() {
		job = jobs.NewTaskToJobConverter(serviceAccount, registrySecret, allowAutomountServiceAccountToken).Convert(task)
	})

	It("returns a job for the task with the correct attributes", func() {
		assertGeneralSpec(job)

		Expect(job.Name).To(Equal("my-app-my-space-task-name"))
		Expect(job.Spec.Template.Spec.ServiceAccountName).To(Equal(serviceAccount))
		Expect(job.Spec.Template.Spec.ImagePullSecrets).To(ConsistOf(corev1.LocalObjectReference{Name: registrySecret}))

		containers := job.Spec.Template.Spec.Containers
		Expect(containers).To(HaveLen(1))
		assertContainer(containers[0], "opi-task")
		Expect(containers[0].Command).To(ConsistOf("/lifecycle/launch"))

		By("setting the expected annotations on the job", func() {
			Expect(job.Annotations).To(SatisfyAll(
				HaveKeyWithValue(jobs.AnnotationAppName, "my-app"),
				HaveKeyWithValue(jobs.AnnotationAppID, "my-app-guid"),
				HaveKeyWithValue(jobs.AnnotationOrgName, "my-org"),
				HaveKeyWithValue(jobs.AnnotationOrgGUID, "org-id"),
				HaveKeyWithValue(jobs.AnnotationSpaceName, "my-space"),
				HaveKeyWithValue(jobs.AnnotationSpaceGUID, "space-id"),
			))
		})

		By("setting the expected labels on the job", func() {
			Expect(job.Labels).To(SatisfyAll(
				HaveKeyWithValue(jobs.LabelAppGUID, "my-app-guid"),
				HaveKeyWithValue(jobs.LabelGUID, "task-123"),
				HaveKeyWithValue(jobs.LabelSourceType, "TASK"),
				HaveKeyWithValue(jobs.LabelName, "task-name"),
			))
		})

		By("setting the expected annotations on the associated pod", func() {
			Expect(job.Spec.Template.Annotations).To(SatisfyAll(
				HaveKeyWithValue(jobs.AnnotationAppName, "my-app"),
				HaveKeyWithValue(jobs.AnnotationAppID, "my-app-guid"),
				HaveKeyWithValue(jobs.AnnotationOrgName, "my-org"),
				HaveKeyWithValue(jobs.AnnotationOrgGUID, "org-id"),
				HaveKeyWithValue(jobs.AnnotationSpaceName, "my-space"),
				HaveKeyWithValue(jobs.AnnotationSpaceGUID, "space-id"),
				HaveKeyWithValue(jobs.AnnotationTaskContainerName, "opi-task"),
				HaveKeyWithValue(jobs.AnnotationGUID, "task-123"),
			))
		})

		By("setting the expected labels on the associated pod", func() {
			Expect(job.Spec.Template.Labels).To(SatisfyAll(
				HaveKeyWithValue(jobs.LabelAppGUID, "my-app-guid"),
				HaveKeyWithValue(jobs.LabelGUID, "task-123"),
				HaveKeyWithValue(jobs.LabelSourceType, "TASK"),
			))
		})

		By("creating a secret reference with the registry credentials", func() {
			Expect(job.Spec.Template.Spec.ImagePullSecrets).To(ConsistOf(
				corev1.LocalObjectReference{Name: "registry-secret"},
			))
		})

		By("setting limits and request", func() {
			resources := job.Spec.Template.Spec.Containers[0].Resources
			Expect(resources.Limits.Memory().ScaledValue(resource.Mega)).To(BeEquivalentTo(1))
			Expect(resources.Requests.Memory().ScaledValue(resource.Mega)).To(BeEquivalentTo(1))
			Expect(resources.Limits.StorageEphemeral().ScaledValue(resource.Mega)).To(BeEquivalentTo(3))
			Expect(resources.Requests.StorageEphemeral().ScaledValue(resource.Mega)).To(BeEquivalentTo(3))
			Expect(resources.Requests.Cpu().ScaledValue(resource.Milli)).To(BeEquivalentTo(2))
		})

		By("configuring pod security context", func() {
			securityContext := containers[0].SecurityContext
			Expect(securityContext).NotTo(BeNil())

			Expect(securityContext.AllowPrivilegeEscalation).To(PointTo(BeFalse()))
			Expect(securityContext.RunAsNonRoot).To(PointTo(BeTrue()))

			Expect(securityContext.Capabilities).NotTo(BeNil())
			Expect(securityContext.Capabilities.Drop).To(ConsistOf(corev1.Capability("ALL")))
			Expect(securityContext.Capabilities.Add).To(BeEmpty())

			Expect(securityContext.SeccompProfile).NotTo(BeNil())
			Expect(securityContext.SeccompProfile.Type).To(Equal(corev1.SeccompProfileTypeRuntimeDefault))
		})
	})

	When("the task has environment set", func() {
		BeforeEach(func() {
			task.Spec.Environment = []corev1.EnvVar{
				{
					Name: "bobs",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "your",
							},
							Key: "uncle",
						},
					},
				},
			}
		})

		It("is included in the stateful set env vars", func() {
			Expect(job.Spec.Template.Spec.Containers).To(HaveLen(1))
			container := job.Spec.Template.Spec.Containers[0]
			Expect(container.Env).To(ContainElement(
				corev1.EnvVar{Name: "bobs", ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "your"},
						Key:                  "uncle",
					},
				}},
			))
		})
	})

	When("allowAutomountServiceAccountToken is true", func() {
		BeforeEach(func() {
			allowAutomountServiceAccountToken = true
		})

		It("does not set automountServiceAccountToken on the pod spec", func() {
			Expect(job.Spec.Template.Spec.AutomountServiceAccountToken).To(BeNil())
		})
	})

	When("the app name and space name are too long", func() {
		BeforeEach(func() {
			task.Spec.AppName = "app-with-very-long-name"
			task.Spec.SpaceName = "space-with-a-very-very-very-very-very-very-long-name"
		})

		It("should truncate the app and space name", func() {
			Expect(job.Name).To(Equal("app-with-very-long-name-space-with-a-ver-task-name"))
		})
	})

	When("the prefix would be invalid", func() {
		BeforeEach(func() {
			task.Spec.AppName = ""
			task.Spec.SpaceName = ""
		})

		It("should use the guid as the prefix instead", func() {
			Expect(job.Name).To(Equal(fmt.Sprintf("%s-%s", taskGUID, task.Spec.Name)))
		})
	})

	When("the task supplies an addition registry secret", func() {
		BeforeEach(func() {
			task.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: "my-registry-secret"}}
		})

		It("appends the extra image pull secret on the pod spec", func() {
			Expect(job.Spec.Template.Spec.ImagePullSecrets).To(ConsistOf(
				corev1.LocalObjectReference{Name: "registry-secret"},
				corev1.LocalObjectReference{Name: "my-registry-secret"},
			))
		})
	})
})
