package jobs_test

import (
	"context"
	"time"

	"code.cloudfoundry.org/eirini-controller/k8s/jobs"
	eiriniv1 "code.cloudfoundry.org/eirini-controller/pkg/apis/eirini/v1"
	"code.cloudfoundry.org/eirini-controller/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("StatusGetter", func() {
	var (
		statusGetter *jobs.StatusGetter
		job          *batchv1.Job
		conditions   []metav1.Condition
	)

	BeforeEach(func() {
		job = &batchv1.Job{
			Status: batchv1.JobStatus{},
		}

		statusGetter = jobs.NewStatusGetter(tests.NewTestLogger("status_getter_test"))
	})

	JustBeforeEach(func() {
		conditions = statusGetter.GetStatusConditions(context.Background(), job)
	})

	It("returns an initialized condition", func() {
		Expect(meta.IsStatusConditionTrue(conditions, eiriniv1.TaskInitializedConditionType)).To(BeTrue())
	})

	When("the job is running", func() {
		var now metav1.Time

		BeforeEach(func() {
			now = metav1.Now()
			job = &batchv1.Job{
				Status: batchv1.JobStatus{
					StartTime: &now,
				},
			}
		})

		It("contains a started condition with a matching timestamp", func() {
			Expect(meta.IsStatusConditionTrue(conditions, eiriniv1.TaskStartedConditionType)).To(BeTrue())
			Expect(meta.FindStatusCondition(conditions, eiriniv1.TaskStartedConditionType).LastTransitionTime).To(Equal(now))
		})
	})

	When("the job has succeeded", func() {
		var (
			now   metav1.Time
			later metav1.Time
		)

		BeforeEach(func() {
			now = metav1.Now()
			later = metav1.NewTime(now.Add(time.Hour))
			job = &batchv1.Job{
				Status: batchv1.JobStatus{
					StartTime:      &now,
					Succeeded:      1,
					CompletionTime: &later,
				},
			}
		})

		It("contains a succeeded condition", func() {
			Expect(meta.IsStatusConditionTrue(conditions, eiriniv1.TaskSucceededConditionType)).To(BeTrue())
			Expect(meta.FindStatusCondition(conditions, eiriniv1.TaskSucceededConditionType).LastTransitionTime).To(Equal(later))
		})
	})

	When("the job has failed", func() {
		var (
			now   metav1.Time
			later metav1.Time
		)

		BeforeEach(func() {
			now = metav1.Now()
			later = metav1.NewTime(now.Add(time.Hour))
			job = &batchv1.Job{
				Status: batchv1.JobStatus{
					Conditions: []batchv1.JobCondition{
						{
							Type:               batchv1.JobComplete,
							LastTransitionTime: metav1.Now(),
						},
						{
							Type:               batchv1.JobFailed,
							LastTransitionTime: metav1.Now(),
						},
						{
							Type:               batchv1.JobFailed,
							LastTransitionTime: later,
						},
					},
					StartTime: &now,
					Failed:    1,
				},
			}
		})

		It("returns a failed status", func() {
			Expect(meta.IsStatusConditionTrue(conditions, eiriniv1.TaskFailedConditionType)).To(BeTrue())
			Expect(meta.FindStatusCondition(conditions, eiriniv1.TaskFailedConditionType).LastTransitionTime).To(Equal(later))
		})
	})
})
