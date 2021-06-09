package reconciler_test

import (
	"context"
	"fmt"
	"time"

	"code.cloudfoundry.org/eirini-controller/k8s/reconciler"
	"code.cloudfoundry.org/eirini-controller/k8s/reconciler/reconcilerfakes"
	eiriniv1 "code.cloudfoundry.org/eirini-controller/pkg/apis/eirini/v1"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Task", func() {
	var (
		taskReconciler  *reconciler.Task
		reconcileResult reconcile.Result
		reconcileErr    error
		tasksCrClient   *reconcilerfakes.FakeTasksCrClient
		namespacedName  types.NamespacedName
		workloadClient  *reconcilerfakes.FakeTaskWorkloadClient
		task            *eiriniv1.Task
		ttlSeconds      int
	)

	BeforeEach(func() {
		tasksCrClient = new(reconcilerfakes.FakeTasksCrClient)
		namespacedName = types.NamespacedName{
			Namespace: "my-namespace",
			Name:      "my-name",
		}
		workloadClient = new(reconcilerfakes.FakeTaskWorkloadClient)

		logger := lagertest.NewTestLogger("task-reconciler")
		ttlSeconds = 30
		taskReconciler = reconciler.NewTask(logger, tasksCrClient, workloadClient, ttlSeconds)
		task = &eiriniv1.Task{
			ObjectMeta: metav1.ObjectMeta{
				Name:      namespacedName.Name,
				Namespace: namespacedName.Namespace,
			},
			Spec: eiriniv1.TaskSpec{
				GUID:      "guid",
				Name:      "my-name",
				Image:     "my/image",
				Env:       map[string]string{"foo": "bar"},
				Command:   []string{"foo", "baz"},
				AppName:   "jim",
				AppGUID:   "app-guid",
				OrgName:   "organ",
				OrgGUID:   "orgid",
				SpaceName: "spacan",
				SpaceGUID: "spacid",
				MemoryMB:  768,
				DiskMB:    512,
				CPUWeight: 13,
			},
		}
		tasksCrClient.GetTaskReturns(task, nil)
		workloadClient.GetStatusReturns(eiriniv1.TaskStatus{
			ExecutionStatus: eiriniv1.TaskStarting,
		}, nil)
	})

	JustBeforeEach(func() {
		reconcileResult, reconcileErr = taskReconciler.Reconcile(context.Background(), reconcile.Request{NamespacedName: namespacedName})
	})

	It("creates the job in the CR's namespace", func() {
		Expect(reconcileErr).NotTo(HaveOccurred())

		By("invoking the task desirer", func() {
			Expect(workloadClient.DesireCallCount()).To(Equal(1))
			_, actualTask := workloadClient.DesireArgsForCall(0)
			Expect(actualTask).To(Equal(task))
		})

		By("updating the task execution status", func() {
			Expect(tasksCrClient.UpdateTaskStatusCallCount()).To(Equal(1))
			_, actualTask, status := tasksCrClient.UpdateTaskStatusArgsForCall(0)
			Expect(actualTask).To(Equal(task))
			Expect(status.ExecutionStatus).To(Equal(eiriniv1.TaskStarting))
		})
	})

	It("loads the task using names from request", func() {
		Expect(tasksCrClient.GetTaskCallCount()).To(Equal(1))
		_, namespace, name := tasksCrClient.GetTaskArgsForCall(0)
		Expect(namespace).To(Equal("my-namespace"))
		Expect(name).To(Equal("my-name"))
	})

	When("the task cannot be found", func() {
		BeforeEach(func() {
			tasksCrClient.GetTaskReturns(nil, errors.NewNotFound(schema.GroupResource{}, "foo"))
		})

		It("neither requeues nor returns an error", func() {
			Expect(reconcileResult.Requeue).To(BeFalse())
			Expect(reconcileErr).ToNot(HaveOccurred())
		})
	})

	When("getting the task returns another error", func() {
		BeforeEach(func() {
			tasksCrClient.GetTaskReturns(nil, fmt.Errorf("some problem"))
		})

		It("returns an error", func() {
			Expect(reconcileErr).To(MatchError(ContainSubstring("some problem")))
		})
	})

	It("gets the new task status", func() {
		Expect(workloadClient.GetStatusCallCount()).To(Equal(1))
		_, guid := workloadClient.GetStatusArgsForCall(0)
		Expect(guid).To(Equal("guid"))
	})

	It("updates the task with the new status", func() {
		Expect(tasksCrClient.UpdateTaskStatusCallCount()).To(Equal(1))
		_, _, newStatus := tasksCrClient.UpdateTaskStatusArgsForCall(0)
		Expect(newStatus.ExecutionStatus).To(Equal(eiriniv1.TaskStarting))
	})

	When("the task has previously completed successfully", func() {
		BeforeEach(func() {
			now := metav1.Now()
			task = &eiriniv1.Task{
				Status: eiriniv1.TaskStatus{
					ExecutionStatus: eiriniv1.TaskSucceeded,
					EndTime:         &now,
				},
			}
			tasksCrClient.GetTaskReturns(task, nil)
		})

		It("does not desire the task again", func() {
			Expect(workloadClient.DesireCallCount()).To(Equal(0))
		})

		When("the task has exceeded the ttl", func() {
			BeforeEach(func() {
				earlier := metav1.NewTime(time.Now().Add(-time.Minute))
				task = &eiriniv1.Task{
					Status: eiriniv1.TaskStatus{
						ExecutionStatus: eiriniv1.TaskSucceeded,
						EndTime:         &earlier,
					},
				}
				tasksCrClient.GetTaskReturns(task, nil)
			})

			It("deletes the task", func() {
				Expect(workloadClient.DeleteCallCount()).To(Equal(1))
			})

			When("deleting the task fails", func() {
				BeforeEach(func() {
					workloadClient.DeleteReturns(fmt.Errorf("boom"))
				})

				It("returns an error", func() {
					Expect(reconcileErr).To(MatchError(ContainSubstring("boom")))
				})
			})
		})
	})

	When("the task has previously failed", func() {
		var now metav1.Time

		BeforeEach(func() {
			now = metav1.Now()
			task = &eiriniv1.Task{
				Status: eiriniv1.TaskStatus{
					ExecutionStatus: eiriniv1.TaskFailed,
					EndTime:         &now,
				},
			}
			tasksCrClient.GetTaskReturns(task, nil)
		})

		It("does not desire the task again", func() {
			Expect(workloadClient.DesireCallCount()).To(Equal(0))
		})

		When("the task has exceeded the ttl", func() {
			BeforeEach(func() {
				earlier := metav1.NewTime(time.Now().Add(-time.Minute))
				task = &eiriniv1.Task{
					Status: eiriniv1.TaskStatus{
						ExecutionStatus: eiriniv1.TaskFailed,
						EndTime:         &earlier,
					},
				}
				tasksCrClient.GetTaskReturns(task, nil)
			})

			It("deletes the task", func() {
				Expect(workloadClient.DeleteCallCount()).To(Equal(1))
			})
		})
	})

	When("gettin the task status returns an error", func() {
		BeforeEach(func() {
			workloadClient.GetStatusReturns(eiriniv1.TaskStatus{}, fmt.Errorf("potato"))
		})

		It("returns an error", func() {
			Expect(reconcileErr).To(MatchError(ContainSubstring("potato")))
		})
	})

	When("updating the task status returns an error", func() {
		BeforeEach(func() {
			tasksCrClient.UpdateTaskStatusReturns(fmt.Errorf("crumpets"))
		})

		It("returns an error", func() {
			Expect(reconcileErr).To(MatchError(ContainSubstring("crumpets")))
		})
	})

	When("the task has completed successfully", func() {
		BeforeEach(func() {
			now := metav1.Now()
			workloadClient.GetStatusReturns(eiriniv1.TaskStatus{
				ExecutionStatus: eiriniv1.TaskSucceeded,
				EndTime:         &now,
			}, nil)
		})

		It("requeues the event after the ttl", func() {
			Expect(reconcileResult.RequeueAfter).To(Equal(time.Duration(ttlSeconds) * time.Second))
		})
	})

	When("the task has failed", func() {
		BeforeEach(func() {
			now := metav1.Now()
			workloadClient.GetStatusReturns(eiriniv1.TaskStatus{
				ExecutionStatus: eiriniv1.TaskFailed,
				EndTime:         &now,
			}, nil)
		})

		It("requeues the event after the ttl", func() {
			Expect(reconcileResult.RequeueAfter).To(Equal(time.Duration(ttlSeconds) * time.Second))
		})
	})

	When("there is a private registry set", func() {
		BeforeEach(func() {
			task.Spec.PrivateRegistry = &eiriniv1.PrivateRegistry{
				Username: "admin",
				Password: "p4ssw0rd",
			}
		})

		It("passes the private registry details to the desirer", func() {
			Expect(workloadClient.DesireCallCount()).To(Equal(1))
			_, actualTask := workloadClient.DesireArgsForCall(0)
			Expect(actualTask.Spec.PrivateRegistry).ToNot(BeNil())
			Expect(actualTask.Spec.PrivateRegistry.Username).To(Equal("admin"))
			Expect(actualTask.Spec.PrivateRegistry.Password).To(Equal("p4ssw0rd"))
		})
	})

	When("desiring the task returns an error", func() {
		BeforeEach(func() {
			workloadClient.DesireReturns(fmt.Errorf("some error"))
		})

		It("returns an error", func() {
			Expect(reconcileErr).To(MatchError(ContainSubstring("some error")))
			Expect(tasksCrClient.UpdateTaskStatusCallCount()).To(Equal(0))
		})
	})
})
