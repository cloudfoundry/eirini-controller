package eats_test

import (
	"context"

	eiriniv1 "code.cloudfoundry.org/eirini-controller/pkg/apis/eirini/v1"
	"code.cloudfoundry.org/eirini-controller/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Tasks CRD [needs-logs-for: eirini-controller]", func() {
	var (
		task            *eiriniv1.Task
		taskName        string
		taskGUID        string
		taskDeleteOpts  metav1.DeleteOptions
		taskServiceName string
		port            int32
		ctx             context.Context
		envSecret       *corev1.Secret
	)

	BeforeEach(func() {
		port = 8080

		taskName = tests.GenerateGUID()
		taskGUID = tests.GenerateGUID()
		var err error
		envSecret, err = fixture.Clientset.CoreV1().Secrets(fixture.Namespace).
			Create(context.Background(), &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: tests.GenerateGUID()},
				StringData: map[string]string{"password": "my-password"},
			}, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		falsePointer := false
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
				Environment: []corev1.EnvVar{
					{
						Name: "PASSWORD",
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: envSecret.Name,
								},
								Key:      "password",
								Optional: &falsePointer,
							},
						},
					},
				},
				Image:   "eirini/dorini",
				Command: []string{"/notdora"},
			},
		}

		ctx = context.Background()
	})

	getTaskStatus := func() (eiriniv1.TaskStatus, error) {
		runningTask, err := fixture.EiriniClientset.
			EiriniV1().
			Tasks(fixture.Namespace).
			Get(ctx, taskName, metav1.GetOptions{})
		if err != nil {
			return eiriniv1.TaskStatus{}, err
		}

		return runningTask.Status, nil
	}

	Describe("Creating a Task CRD", func() {
		JustBeforeEach(func() {
			_, err := fixture.EiriniClientset.
				EiriniV1().
				Tasks(fixture.Namespace).
				Create(ctx, task, metav1.CreateOptions{})

			Expect(err).NotTo(HaveOccurred())

			taskServiceName = tests.ExposeAsService(fixture.Clientset, fixture.Namespace, taskGUID, port)
		})

		It("initializes the task", func() {
			Eventually(func(g Gomega) {
				status, err := getTaskStatus()
				g.Expect(err).NotTo(HaveOccurred())

				g.Expect(meta.IsStatusConditionTrue(status.Conditions, eiriniv1.TaskInitializedConditionType)).Should(BeTrue())
			}).Should(Succeed())

			jobs, err := fixture.Clientset.BatchV1().Jobs(fixture.Namespace).List(ctx, metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(jobs.Items).To(HaveLen(1))
		})

		It("runs the task", func() {
			Eventually(func(g Gomega) {
				status, err := getTaskStatus()
				g.Expect(err).NotTo(HaveOccurred())

				g.Expect(meta.IsStatusConditionTrue(status.Conditions, eiriniv1.TaskStartedConditionType)).Should(BeTrue())
				g.Expect(meta.FindStatusCondition(status.Conditions, eiriniv1.TaskStartedConditionType).LastTransitionTime).NotTo(BeZero())
			}).Should(Succeed())

			Eventually(tests.RequestServiceFn(fixture.Namespace, taskServiceName, port, "/")).Should(ContainSubstring("Dora!"))
		})

		It("gets the env var via the secret", func() {
			Eventually(tests.RequestServiceFn(fixture.Namespace, taskServiceName, port, "/env")).Should(ContainSubstring("PASSWORD=my-password"))
		})

		When("the task image lives in a private registry", func() {
			BeforeEach(func() {
				task.Spec.Image = "eiriniuser/notdora:latest"
				task.Spec.PrivateRegistry = &eiriniv1.PrivateRegistry{
					Username: "eiriniuser",
					Password: tests.GetEiriniDockerHubPassword(),
				}
				port = 8888
			})

			It("runs the task", func() {
				Eventually(tests.RequestServiceFn(fixture.Namespace, taskServiceName, port, "/")).Should(ContainSubstring("Dora!"))
			})
		})

		When("the task completes successfully", func() {
			BeforeEach(func() {
				task.Spec.Image = "eirini/busybox"
				task.Spec.Command = []string{"echo", "hello"}
			})

			It("sets the succeeded condition", func() {
				Eventually(func(g Gomega) {
					status, err := getTaskStatus()
					g.Expect(err).NotTo(HaveOccurred())

					g.Expect(meta.IsStatusConditionTrue(status.Conditions, eiriniv1.TaskSucceededConditionType)).Should(BeTrue())
					g.Expect(meta.FindStatusCondition(status.Conditions, eiriniv1.TaskSucceededConditionType).LastTransitionTime).NotTo(BeZero())
				}).Should(Succeed())
			})
		})

		When("the task fails", func() {
			BeforeEach(func() {
				task.Spec.Image = "eirini/busybox"
				task.Spec.Command = []string{"false"}
			})

			It("sets the failed condition", func() {
				Eventually(func(g Gomega) {
					status, err := getTaskStatus()
					g.Expect(err).NotTo(HaveOccurred())

					g.Expect(meta.IsStatusConditionTrue(status.Conditions, eiriniv1.TaskFailedConditionType)).Should(BeTrue())
					cond := meta.FindStatusCondition(status.Conditions, eiriniv1.TaskFailedConditionType)
					g.Expect(cond.LastTransitionTime).NotTo(BeZero())
					g.Expect(cond.Reason).To(Equal("Error"))
					g.Expect(cond.Message).To(Equal("Failed with exit code: 1"))
				}).Should(Succeed())
			})
		})
	})

	Describe("Deleting the Task CRD", func() {
		BeforeEach(func() {
			_, err := fixture.EiriniClientset.
				EiriniV1().
				Tasks(fixture.Namespace).
				Create(context.Background(), task, metav1.CreateOptions{})

			Expect(err).NotTo(HaveOccurred())
			taskServiceName = tests.ExposeAsService(fixture.Clientset, fixture.Namespace, taskGUID, port)
			Eventually(tests.RequestServiceFn(fixture.Namespace, taskServiceName, port, "/")).Should(ContainSubstring("Dora!"))
		})

		JustBeforeEach(func() {
			err := fixture.EiriniClientset.
				EiriniV1().
				Tasks(fixture.Namespace).
				Delete(context.Background(), taskName, taskDeleteOpts)
			Expect(err).NotTo(HaveOccurred())
		})

		It("kills the task", func() {
			// better to check Task status here, once that is available
			Eventually(func() error {
				_, err := tests.RequestServiceFn(fixture.Namespace, taskServiceName, port, "/")()

				return err
			}, "20s").Should(HaveOccurred())
		})
	})
})
