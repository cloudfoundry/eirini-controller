package eirini_controller_test

import (
	"context"

	eiriniv1 "code.cloudfoundry.org/eirini-controller/pkg/apis/eirini/v1"
	"code.cloudfoundry.org/eirini-controller/tests"
	"code.cloudfoundry.org/eirini-controller/tests/integration"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Tasks", func() {
	var (
		taskName    string
		taskGUID    string
		task        *eiriniv1.Task
		serviceName string
	)

	BeforeEach(func() {
		taskName = "the-task"
		taskGUID = tests.GenerateGUID()

		task = &eiriniv1.Task{
			ObjectMeta: metav1.ObjectMeta{
				Name: taskName,
			},
			Spec: eiriniv1.TaskSpec{
				Name:      taskName,
				GUID:      taskGUID,
				AppGUID:   "the-app-guid",
				AppName:   "wavey",
				SpaceName: "the-space",
				OrgName:   "the-org",
				Env: map[string]string{
					"FOO": "BAR",
				},
				Image:   "eirini/dorini",
				Command: []string{"/notdora"},
			},
		}
	})

	JustBeforeEach(func() {
		_, err := fixture.EiriniClientset.
			EiriniV1().
			Tasks(fixture.Namespace).
			Create(context.Background(), task, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		Eventually(integration.EnsureStatusConditionTrue(fixture.EiriniClientset, fixture.Namespace, taskName, eiriniv1.TaskStartedConditionType)).Should(Succeed())
	})

	Describe("task creation", func() {
		JustBeforeEach(func() {
			serviceName = tests.ExposeAsService(fixture.Clientset, fixture.Namespace, taskGUID, 8080, "/")
		})

		It("runs the task", func() {
			Expect(tests.RequestServiceFn(fixture.Namespace, serviceName, 8080, "/")()).To(ContainSubstring("not Dora"))
		})
	})

	Describe("task time to live", func() {
		BeforeEach(func() {
			task.Spec.Image = "eirini/busybox"
			task.Spec.Command = []string{"/bin/sh", "-c", "sleep 1"}
		})

		It("deletes the job after the ttl has expired", func() {
			Eventually(integration.EnsureStatusConditionTrue(fixture.EiriniClientset, fixture.Namespace, taskName, eiriniv1.TaskSucceededConditionType)).Should(Succeed())

			Eventually(integration.ListJobs(fixture.Clientset,
				fixture.Namespace,
				taskGUID,
			)).Should(BeEmpty())
			Consistently(integration.ListJobs(fixture.Clientset,
				fixture.Namespace,
				taskGUID,
			)).Should(BeEmpty())
		})
	})

	Describe("task deletion", func() {
		JustBeforeEach(func() {
			serviceName = tests.ExposeAsService(fixture.Clientset, fixture.Namespace, taskGUID, 8080, "/")
			err := fixture.EiriniClientset.
				EiriniV1().
				Tasks(fixture.Namespace).
				Delete(context.Background(), taskName, metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred())
		})

		It("stops the task", func() {
			Eventually(func() error {
				_, err := tests.RequestServiceFn(fixture.Namespace, serviceName, 8080, "/")()

				return err
			}).Should(MatchError(ContainSubstring("context deadline exceeded")))
		})
	})
})
