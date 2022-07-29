package jobs_test

import (
	"code.cloudfoundry.org/eirini-controller/k8s/jobs"
	"code.cloudfoundry.org/eirini-controller/k8s/jobs/jobsfakes"
	"code.cloudfoundry.org/eirini-controller/k8s/k8sfakes"
	eiriniv1 "code.cloudfoundry.org/eirini-controller/pkg/apis/eirini/v1"
	eirinischeme "code.cloudfoundry.org/eirini-controller/pkg/generated/clientset/versioned/scheme"
	"code.cloudfoundry.org/eirini-controller/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Desire", func() {
	const (
		image    = "gcr.io/foo/bar"
		taskGUID = "task-123"
	)

	var (
		taskToJobConverter *jobsfakes.FakeTaskToJobConverter
		client             *k8sfakes.FakeClient

		job        *batchv1.Job
		createdJob *batchv1.Job
		task       *eiriniv1.Task
		desireErr  error

		desirer *jobs.Desirer
	)

	BeforeEach(func() {
		job = &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name: "the-job-name",
				UID:  "the-job-uid",
			},
		}

		client = new(k8sfakes.FakeClient)
		taskToJobConverter = new(jobsfakes.FakeTaskToJobConverter)
		taskToJobConverter.ConvertReturns(job)

		task = &eiriniv1.Task{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "app-namespace",
			},
			Spec: eiriniv1.TaskSpec{
				Image: image,
				ImagePullSecrets: []corev1.LocalObjectReference{{
					Name: "my-registry-secret",
				}},
				Command:   []string{"/lifecycle/launch"},
				AppName:   "my-app",
				Name:      "task-name",
				AppGUID:   "my-app-guid",
				OrgName:   "my-org",
				SpaceName: "my-space",
				SpaceGUID: "space-id",
				OrgGUID:   "org-id",
				GUID:      taskGUID,
				MemoryMB:  1,
				CPUMillis: 2,
				DiskMB:    3,
			},
		}

		desirer = jobs.NewDesirer(
			tests.NewTestLogger("desiretask"),
			taskToJobConverter,
			client,
			eirinischeme.Scheme,
		)
	})

	JustBeforeEach(func() {
		createdJob, desireErr = desirer.Desire(ctx, task)
	})

	It("succeeds", func() {
		Expect(createdJob).To(Equal(job))
		Expect(desireErr).NotTo(HaveOccurred())
	})

	It("creates a job", func() {
		Expect(client.CreateCallCount()).To(Equal(1))
		_, actualJob, _ := client.CreateArgsForCall(0)
		Expect(actualJob).To(Equal(job))
	})

	When("creating the job fails", func() {
		BeforeEach(func() {
			client.CreateReturns(errors.New("create-failed"))
		})

		It("returns an error", func() {
			Expect(desireErr).To(MatchError(ContainSubstring("create-failed")))
		})
	})

	It("converts the task to job", func() {
		Expect(taskToJobConverter.ConvertCallCount()).To(Equal(1))
		Expect(taskToJobConverter.ConvertArgsForCall(0)).To(Equal(task))
	})

	It("sets the job namespace", func() {
		Expect(job.Namespace).To(Equal("app-namespace"))
	})
})
