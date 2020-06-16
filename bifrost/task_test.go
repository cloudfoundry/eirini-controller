package bifrost_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"code.cloudfoundry.org/eirini/bifrost"
	"code.cloudfoundry.org/eirini/bifrost/bifrostfakes"
	"code.cloudfoundry.org/eirini/models/cf"
	"code.cloudfoundry.org/eirini/opi"
)

var _ = Describe("Buildpack task", func() {
	var (
		err           error
		taskBifrost   *bifrost.Task
		taskConverter *bifrostfakes.FakeTaskConverter
		taskDesirer   *bifrostfakes.FakeTaskDesirer
		jsonClient    *bifrostfakes.FakeJSONClient
		taskGUID      string
		task          opi.Task
	)

	BeforeEach(func() {
		taskConverter = new(bifrostfakes.FakeTaskConverter)
		taskDesirer = new(bifrostfakes.FakeTaskDesirer)
		jsonClient = new(bifrostfakes.FakeJSONClient)
		taskGUID = "task-guid"
		task = opi.Task{GUID: "my-guid"}
		taskConverter.ConvertTaskReturns(task, nil)
		taskBifrost = &bifrost.Task{
			Converter:   taskConverter,
			TaskDesirer: taskDesirer,
			JSONClient:  jsonClient,
		}
	})

	Describe("Transfer Task", func() {
		var (
			taskRequest cf.TaskRequest
		)

		BeforeEach(func() {
			taskRequest = cf.TaskRequest{
				Name:               "cake",
				AppGUID:            "app-guid",
				AppName:            "foo",
				OrgName:            "my-org",
				OrgGUID:            "asdf123",
				SpaceName:          "my-space",
				SpaceGUID:          "fdsa4321",
				CompletionCallback: "my-callback",
				Environment:        nil,
				Lifecycle: cf.Lifecycle{
					BuildpackLifecycle: &cf.BuildpackLifecycle{
						DropletHash:  "h123jhh",
						DropletGUID:  "fds1234",
						StartCommand: "run",
					},
				},
			}
			task := opi.Task{GUID: "my-guid"}
			taskConverter.ConvertTaskReturns(task, nil)

		})

		JustBeforeEach(func() {
			err = taskBifrost.TransferTask(context.Background(), taskGUID, taskRequest)
		})

		It("transfers the task", func() {
			Expect(err).NotTo(HaveOccurred())

			Expect(taskConverter.ConvertTaskCallCount()).To(Equal(1))
			actualTaskGUID, actualTaskRequest := taskConverter.ConvertTaskArgsForCall(0)
			Expect(actualTaskGUID).To(Equal(taskGUID))
			Expect(actualTaskRequest).To(Equal(taskRequest))

			Expect(taskDesirer.DesireCallCount()).To(Equal(1))
			desiredTask := taskDesirer.DesireArgsForCall(0)
			Expect(desiredTask.GUID).To(Equal("my-guid"))
		})

		When("converting the task fails", func() {
			BeforeEach(func() {
				taskConverter.ConvertTaskReturns(opi.Task{}, errors.New("task-conv-err"))
			})

			It("returns the error", func() {
				Expect(err).To(MatchError(ContainSubstring("task-conv-err")))
			})

			It("does not desire the task", func() {
				Expect(taskDesirer.DesireCallCount()).To(Equal(0))
			})
		})

		When("desiring the task fails", func() {
			BeforeEach(func() {
				taskDesirer.DesireReturns(errors.New("desire-task-err"))
			})

			It("returns the error", func() {
				Expect(err).To(MatchError(ContainSubstring("desire-task-err")))
			})
		})
	})

	Describe("Cancel Task", func() {
		BeforeEach(func() {
			taskDesirer.DeleteReturns("the/callback/url", nil)
		})

		JustBeforeEach(func() {
			err = taskBifrost.CancelTask(taskGUID)
		})

		It("succeeds", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("deletes the task", func() {
			Expect(taskDesirer.DeleteCallCount()).To(Equal(1))
			Expect(taskDesirer.DeleteArgsForCall(0)).To(Equal(taskGUID))
		})

		When("deleting the task fails", func() {
			BeforeEach(func() {
				taskDesirer.DeleteReturns("", errors.New("delete-task-err"))
			})

			It("returns the error", func() {
				Expect(err).To(MatchError(ContainSubstring("delete-task-err")))
			})
		})

		It("notifies the cloud controller", func() {
			Eventually(jsonClient.PostCallCount).Should(Equal(1))

			url, data := jsonClient.PostArgsForCall(0)
			Expect(url).To(Equal("the/callback/url"))
			Expect(data).To(Equal(cf.TaskCompletedRequest{
				TaskGUID:      taskGUID,
				Failed:        true,
				FailureReason: "task was cancelled",
			}))
		})

		When("notifying the cloud controller fails", func() {
			BeforeEach(func() {
				jsonClient.PostReturns(errors.New("cc-error"))
			})

			It("still succeeds", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		When("the callback URL is empty", func() {
			BeforeEach(func() {
				taskDesirer.DeleteReturns("", nil)
			})

			It("still succeeds", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("does not notify the cloud controller", func() {
				Consistently(jsonClient.PostCallCount).Should(BeZero())
			})
		})

		When("cloud controller notification takes forever", func() {
			It("still succeeds", func(done Done) {
				jsonClient.PostStub = func(string, interface{}) error {
					<-make(chan interface{}) // block forever
					return nil
				}

				err = taskBifrost.CancelTask(taskGUID)

				Expect(err).NotTo(HaveOccurred())

				close(done)
			})
		})
	})
})