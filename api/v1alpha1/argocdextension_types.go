package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ArgoCDExtensionSpec defines the desired state of ArgoCDExtension
type ArgoCDExtensionSpec struct {
	// Sources specifies where the extension should come from
	Sources []ExtensionSource `json:"sources"`
}

type ArgoCDExtensionConditionType string

const (
	ConditionReady ArgoCDExtensionConditionType = "Ready"
)

type ArgoCDExtensionCondition struct {
	// Type is an ArgoCDExtension condition type
	Type ArgoCDExtensionConditionType `json:"type"`
	// Boolean status describing if the condition is currently true
	Status metav1.ConditionStatus `json:"status,string"`
	// Message contains human-readable message indicating details about condition
	Message string `json:"message"`
}

// ArgoCDExtensionStatus defines the observed state of ArgoCDExtension
type ArgoCDExtensionStatus struct {
	Conditions []ArgoCDExtensionCondition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true

// ArgoCDExtension is the Schema for the argocdextensions API
type ArgoCDExtension struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ArgoCDExtensionSpec   `json:"spec,omitempty"`
	Status ArgoCDExtensionStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ArgoCDExtensionList contains a list of ArgoCDExtension
type ArgoCDExtensionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ArgoCDExtension `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ArgoCDExtension{}, &ArgoCDExtensionList{})
}

// ExtensionSource specifies where the extension should be sourced from
type ExtensionSource struct {
	// Git is specified if the extension should be sourced from a git repository
	Git *GitSource `json:"git,omitempty"`
	// Web is specified if the extension should be sourced from a web file
	Web *WebSource `json:"web,omitempty"`
}

// GitSource specifies a repo that holds an extension
type GitSource struct {
	// URL specifies the Git repository URL to fetch
	Url string `json:"url,omitempty"`
	// Revision specifies the revision of the Repository to fetch
	Revision string `json:"revision,omitempty"`
}

// WebSource specifies a repo that holds an extension
type WebSource struct {
	// URK specifies the remote file URL
	Url string `json:"url,omitempty"`
}
