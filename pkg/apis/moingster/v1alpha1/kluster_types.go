package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KlusterSpec defines the desired state of Kluster
type KlusterSpec struct {
	Kubernetes Kubernetes `json:"kubernetes"`
	Nodes      Nodes      `json:"nodes"`
}

type Kubernetes struct {
	Version         string `json:"version"`
	NetworkProvider string `json:"networkProvider"`
}

type Nodes struct {
	Count    int      `json:"count"`
	NodeSpec NodeSpec `json:"nodeSpec"`
}

type NodeSpec struct {
	Cores    int `json:"cores"`
	MemoryMb int `json:"memoryMb"`
}

// KlusterStatus defines the observed state of Kluster
type KlusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Kluster is the Schema for the klusters API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=klusters,scope=Namespaced
type Kluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KlusterSpec   `json:"spec,omitempty"`
	Status KlusterStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KlusterList contains a list of Kluster
type KlusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Kluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Kluster{}, &KlusterList{})
}
