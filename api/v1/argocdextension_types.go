/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ArgoCDExtensionSpec defines the desired state of ArgoCDExtension
type ArgoCDExtensionSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Extends specifies what part of Argo CD should be extended
	Extends string `json:"extends,omitempty"`

	// Source specifies where the extension should come from
	Source ExtensionSource `json:"source"`

	// Target specifies which K8S resource the extension should target
	Target ExtensionTarget `json:"target,omitempty"`
}

// ArgoCDExtensionStatus defines the observed state of ArgoCDExtension
type ArgoCDExtensionStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

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
	// Repository is specified if the extension should be sourced from a git repository
	Repository *RepositorySource `json:"repository,omitempty"`
	// File is specified if the extension should be sourced from a URL
	File *string `json:"file,omitempty"`
}

// RepositorySource specifies a repo that holds an extension
type RepositorySource struct {
	// Name specifies the name of the Repository to fetch
	Url *string `json:"name,omitempty"`
	// Revision specifies the revision of the Repository to fetch
	Revision *string `json:"revision,omitempty"`
}

// ExtensionTarget specifies what the extension should target
type ExtensionTarget struct {
	// Resource specifies a K8S resource to target
	Resource ResourceTarget `json:"resource,omitempty"`
}

type ResourceTarget struct {
	Group string `json:"group,omitempty"`
	Kind  string `json:"kind,omitempty"`
}
