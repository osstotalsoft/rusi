package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Subscription describes an pub/sub event subscription.
type Subscription struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              SubscriptionSpec `json:"spec,omitempty"`
	// +optional
	Scopes []string `json:"scopes,omitempty"`
}

// SubscriptionSpec is the spec for an event subscription.
type SubscriptionSpec struct {
	Topic      string `json:"topic"`
	Pubsubname string `json:"pubsubname"`
	// +optional
	Metadata map[string]string `json:"metadata,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SubscriptionList is a list of Rusi event sources.
type SubscriptionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Subscription `json:"items"`
}
