package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Component describes a Rusi component type.
type Component struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +optional
	Spec ComponentSpec `json:"spec,omitempty"`
	// +optional
	Auth `json:"auth,omitempty"`
	// +optional
	Scopes []string `json:"scopes,omitempty"`
}

// ComponentSpec is the spec for a component.
type ComponentSpec struct {
	Type    string `json:"type"`
	Version string `json:"version"`
	// +optional
	IgnoreErrors bool           `json:"ignoreErrors"`
	Metadata     []MetadataItem `json:"metadata"`
	// +optional
	InitTimeout string `json:"initTimeout"`
}

// MetadataItem is a name/value pair for a metadata.
type MetadataItem struct {
	Name string `json:"name"`
	// +optional
	Value apiextensionsv1.JSON `json:"value,omitempty"`
	// +optional
	SecretKeyRef SecretKeyRef `json:"secretKeyRef,omitempty"`
}

// SecretKeyRef is a reference to a secret holding the value for the metadata item. Name is the secret name, and key is the field in the secret.
type SecretKeyRef struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

// Auth represents authentication details for the component.
type Auth struct {
	SecretStore string `json:"secretStore"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ComponentList is a list of Rusi components.
type ComponentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Component `json:"items"`
}
