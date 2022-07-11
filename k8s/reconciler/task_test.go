package reconciler_test

import (
	"context"
	"fmt"
	"time"

	"code.cloudfoundry.org/eirini-controller/k8s/k8sfakes"
	"code.cloudfoundry.org/eirini-controller/k8s/reconciler"
	"code.cloudfoundry.org/eirini-controller/k8s/reconciler/reconcilerfakes"
	eiriniv1 "code.cloudfoundry.org/eirini-controller/pkg/apis/eirini/v1"
	"code.cloudfoundry.org/eirini-controller/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Task", func() {
	var (
		taskReconciler  *reconciler.Task
		reconcileResult reconcile.Result
		reconcileErr    error
		getTaskErr      error
		getJobErr       error
		namespacedName  types.NamespacedName
		task            *eiriniv1.Task
		ttlSeconds      int
		k8sClient       *k8sfakes.FakeClient
		statusWriter    *k8sfakes.FakeStatusWriter
		desirer         *reconcilerfakes.FakeTaskDesirer
		statusGetter    *reconcilerfakes.FakeTaskStatusGetter
	)

	BeforeEach(func() {
		k8sClient = new(k8sfakes.FakeClient)
		statusWriter = new(k8sfakes.FakeStatusWriter)
		k8sClient.StatusReturns(statusWriter)

		getTaskErr = nil
		getJobErr = k8serrors.NewNotFound(schema.GroupResource{}, "not found")
		k8sClient.GetStub = func(_ context.Context, _ types.NamespacedName, o client.Object) error {
			taskPtr, ok := o.(*eiriniv1.Task)
			if ok {
				if getTaskErr != nil {
					return getTaskErr
				}
				task.DeepCopyInto(taskPtr)

				return nil
			}

			jobPtr, ok := o.(*batchv1.Job)
			if ok {
				if getJobErr != nil {
					return getJobErr
				}
				(&batchv1.Job{}).DeepCopyInto(jobPtr)

				return nil
			}

			Fail(fmt.Sprintf("Unsupported object: %v", o))

			return nil
		}
		desirer = new(reconcilerfakes.FakeTaskDesirer)
		statusGetter = new(reconcilerfakes.FakeTaskStatusGetter)

		namespacedName = types.NamespacedName{
			Namespace: "my-namespace",
			Name:      "my-name",
		}

		logger := tests.NewTestLogger("task-reconciler")
		ttlSeconds = 30
		taskReconciler = reconciler.NewTask(logger, k8sClient, desirer, statusGetter, ttlSeconds)
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
		statusGetter.GetStatusConditionsReturns([]metav1.Condition{
			{
				Type:   eiriniv1.TaskInitializedConditionType,
				Status: metav1.ConditionTrue,
			},
		}, nil)
	})

	JustBeforeEach(func() {
		reconcileResult, reconcileErr = taskReconciler.Reconcile(context.Background(), reconcile.Request{NamespacedName: namespacedName})
	})

	It("creates the job in the CR's namespace", func() {
		Expect(reconcileErr).NotTo(HaveOccurred())

		By("invoking the task desirer", func() {
			Expect(desirer.DesireCallCount()).To(Equal(1))
			_, actualTask := desirer.DesireArgsForCall(0)
			Expect(actualTask.Spec).To(Equal(task.Spec))
		})
	})

	It("loads the task using names from request", func() {
		Expect(k8sClient.GetCallCount()).To(Equal(2))
		_, namespacedName, _ := k8sClient.GetArgsForCall(0)
		Expect(namespacedName.Namespace).To(Equal("my-namespace"))
		Expect(namespacedName.Name).To(Equal("my-name"))
	})

	When("getting status condition fails", func() {
		BeforeEach(func() {
			statusGetter.GetStatusConditionsReturns(nil, errors.New("boom"))
			getJobErr = nil
		})

		It("returns an error", func() {
			Expect(reconcileErr).To(MatchError(ContainSubstring("boom")))
		})
	})

	When("the task cannot be found", func() {
		BeforeEach(func() {
			getTaskErr = k8serrors.NewNotFound(schema.GroupResource{}, "foo")
		})

		It("neither requeues nor returns an error", func() {
			Expect(reconcileResult.Requeue).To(BeFalse())
			Expect(reconcileErr).ToNot(HaveOccurred())
		})
	})

	When("getting the task returns another error", func() {
		BeforeEach(func() {
			getTaskErr = errors.New("some problem")
		})

		It("returns an error", func() {
			Expect(reconcileErr).To(MatchError(ContainSubstring("some problem")))
		})
	})

	When("the job cannot be found", func() {
		BeforeEach(func() {
			getJobErr = k8serrors.NewNotFound(schema.GroupResource{}, "foo")
		})

		It("desires the job", func() {
			Expect(desirer.DesireCallCount()).To(Equal(1))
			_, actualTask := desirer.DesireArgsForCall(0)
			Expect(actualTask.Name).To(Equal(task.Name))
			Expect(actualTask.Namespace).To(Equal(task.Namespace))
			Expect(actualTask.Spec).To(Equal(task.Spec))
		})

		It("updates the task execution status (to initialized)", func() {
			Expect(statusGetter.GetStatusConditionsCallCount()).To(Equal(1))

			Expect(statusWriter.PatchCallCount()).To(Equal(1))
			_, obj, _, _ := statusWriter.PatchArgsForCall(0)
			Expect(obj).To(BeAssignableToTypeOf(&eiriniv1.Task{}))
		})

		When("updating the task status returns an error", func() {
			BeforeEach(func() {
				statusWriter.PatchReturns(errors.New("crumpets"))
			})

			It("returns an error", func() {
				Expect(reconcileErr).To(MatchError(ContainSubstring("crumpets")))
			})
		})

		When("desiring the job fails", func() {
			BeforeEach(func() {
				desirer.DesireReturns(nil, errors.New("boom"))
			})

			It("doesn't update the status", func() {
				Expect(statusWriter.PatchCallCount()).To(Equal(0))
			})

			It("returns an error", func() {
				Expect(reconcileErr).To(MatchError(ContainSubstring("boom")))
			})
		})
	})

	When("getting the job returns another error", func() {
		BeforeEach(func() {
			getJobErr = errors.New("some problem")
		})

		It("returns an error", func() {
			Expect(reconcileErr).To(MatchError(ContainSubstring("some problem")))
		})

		It("doesn't update the status", func() {
			Expect(statusWriter.PatchCallCount()).To(Equal(0))
		})
	})

	When("the task has previously completed successfully", func() {
		BeforeEach(func() {
			task = &eiriniv1.Task{
				Status: eiriniv1.TaskStatus{
					Conditions: []metav1.Condition{{
						Type:               eiriniv1.TaskSucceededConditionType,
						Status:             metav1.ConditionTrue,
						LastTransitionTime: metav1.Now(),
					}},
				},
			}
		})

		It("does not desire the task again", func() {
			Expect(desirer.DesireCallCount()).To(Equal(0))
		})

		When("the task has exceeded the ttl", func() {
			BeforeEach(func() {
				task = &eiriniv1.Task{
					Status: eiriniv1.TaskStatus{
						Conditions: []metav1.Condition{{
							Type:               eiriniv1.TaskSucceededConditionType,
							Status:             metav1.ConditionTrue,
							LastTransitionTime: metav1.NewTime(time.Now().Add(-time.Minute)),
						}},
					},
				}
			})

			It("deletes the task", func() {
				Expect(k8sClient.DeleteAllOfCallCount()).To(Equal(1))
			})

			When("deleting the task fails", func() {
				BeforeEach(func() {
					k8sClient.DeleteAllOfReturns(errors.New("boom"))
				})

				It("returns an error", func() {
					Expect(reconcileErr).To(MatchError(ContainSubstring("boom")))
				})
			})
		})
	})

	When("the task has previously failed", func() {
		BeforeEach(func() {
			task = &eiriniv1.Task{
				Status: eiriniv1.TaskStatus{
					Conditions: []metav1.Condition{{
						Type:               eiriniv1.TaskFailedConditionType,
						Status:             metav1.ConditionTrue,
						LastTransitionTime: metav1.Now(),
					}},
				},
			}
		})

		It("does not desire the task again", func() {
			Expect(desirer.DesireCallCount()).To(Equal(0))
		})

		When("the task has exceeded the ttl", func() {
			BeforeEach(func() {
				task = &eiriniv1.Task{
					Status: eiriniv1.TaskStatus{
						Conditions: []metav1.Condition{{
							Type:               eiriniv1.TaskFailedConditionType,
							Status:             metav1.ConditionTrue,
							LastTransitionTime: metav1.NewTime(time.Now().Add(-time.Minute)),
						}},
					},
				}
			})

			It("deletes the task", func() {
				Expect(k8sClient.DeleteAllOfCallCount()).To(Equal(1))
			})
		})
	})

	When("there is a private registry set", func() {
		BeforeEach(func() {
			getJobErr = k8serrors.NewNotFound(schema.GroupResource{}, "not found")
			task.Spec.PrivateRegistry = &eiriniv1.PrivateRegistry{
				Username: "admin",
				Password: "p4ssw0rd",
			}
		})

		It("passes the private registry details to the desirer", func() {
			Expect(desirer.DesireCallCount()).To(Equal(1))
			_, actualTask := desirer.DesireArgsForCall(0)
			Expect(actualTask.Spec.PrivateRegistry).ToNot(BeNil())
			Expect(actualTask.Spec.PrivateRegistry.Username).To(Equal("admin"))
			Expect(actualTask.Spec.PrivateRegistry.Password).To(Equal("p4ssw0rd"))
		})
	})

	When("the job has already been desired", func() {
		BeforeEach(func() {
			getJobErr = nil
		})

		It("updates the task execution status", func() {
			Expect(statusGetter.GetStatusConditionsCallCount()).To(Equal(1))

			Expect(statusWriter.PatchCallCount()).To(Equal(1))
			_, obj, _, _ := statusWriter.PatchArgsForCall(0)
			Expect(obj).To(BeAssignableToTypeOf(&eiriniv1.Task{}))
		})

		When("the task has failed", func() {
			BeforeEach(func() {
				statusGetter.GetStatusConditionsReturns([]metav1.Condition{
					{
						Type:   eiriniv1.TaskFailedConditionType,
						Status: metav1.ConditionTrue,
					},
				}, nil)
			})

			It("requeues the event after the ttl", func() {
				Expect(reconcileResult.RequeueAfter).To(Equal(time.Duration(ttlSeconds) * time.Second))
			})
		})

		When("the task has completed successfully", func() {
			BeforeEach(func() {
				statusGetter.GetStatusConditionsReturns([]metav1.Condition{
					{
						Type:   eiriniv1.TaskSucceededConditionType,
						Status: metav1.ConditionTrue,
					},
				}, nil)
			})

			It("requeues the event after the ttl", func() {
				Expect(reconcileResult.RequeueAfter).To(Equal(time.Duration(ttlSeconds) * time.Second))
			})
		})

		When("updating the task status returns an error", func() {
			BeforeEach(func() {
				statusWriter.PatchReturns(errors.New("crumpets"))
			})

			It("returns an error", func() {
				Expect(reconcileErr).To(MatchError(ContainSubstring("crumpets")))
			})
		})
	})
})
