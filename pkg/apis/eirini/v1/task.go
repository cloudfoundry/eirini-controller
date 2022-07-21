package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:shortName=task
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:JSONPath=.status.execution_status,type=string,name=State
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Task describes a short-lived job running alongside an LRP
type Task struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TaskSpec   `json:"spec"`
	Status TaskStatus `json:"status"`
}

type TaskSpec struct {
	// +kubebuilder:validation:Required
	GUID string `json:"GUID"`
	Name string `json:"name"`
	// +kubebuilder:validation:Required
	Image           string           `json:"image"`
	PrivateRegistry *PrivateRegistry `json:"privateRegistry,omitempty"`
	// deprecated: Env is deprecated. Use Environment instead
	Env         map[string]string `json:"env,omitempty"`
	Environment []corev1.EnvVar   `json:"environment,omitempty"`
	// +kubebuilder:validation:Required
	Command   []string `json:"command,omitempty"`
	AppName   string   `json:"appName"`
	AppGUID   string   `json:"appGUID"`
	OrgName   string   `json:"orgName"`
	OrgGUID   string   `json:"orgGUID"`
	SpaceName string   `json:"spaceName"`
	SpaceGUID string   `json:"spaceGUID"`
	MemoryMB  int64    `json:"memoryMB"`
	DiskMB    int64    `json:"diskMB"`
	CPUMillis int64    `json:"cpuMillis"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type TaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Task `json:"items"`
}

const (
	TaskInitializedConditionType = "Initialized"
	TaskStartedConditionType     = "Started"
	TaskSucceededConditionType   = "Succeeded"
	TaskFailedConditionType      = "Failed"
)

type TaskStatus struct {
	Conditions []metav1.Condition `json:"conditions"`
}
