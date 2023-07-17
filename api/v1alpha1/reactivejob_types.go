/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and reactivejob-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ReactiveJobSpec defines the desired state of ReactiveJob.
type ReactiveJobSpec struct {
	// Specifies the job that will be created when executing.
	JobTemplate batchv1.JobTemplateSpec `json:"jobTemplate"`
}

// ReactiveJobStatus defines the observed state of ReactiveJob.
type ReactiveJobStatus struct {
	// Observed generation.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// List of status conditions to indicate the status of a ReactiveJob.
	// +optional
	Conditions []ReactiveJobCondition `json:"conditions,omitempty"`

	// Readable form of the state.
	// +optional
	State ReactiveJobState `json:"state,omitempty"`

	// Name of current job; that is, the name of the last job that was created for this ReactiveJob.
	// +optional
	CurrentJobName string `json:"currentJobName,omitempty"`
}

// ReactiveJobCondition contains condition information for a ReactiveJob.
type ReactiveJobCondition struct {
	// Type of the condition.
	Type ReactiveJobConditionType `json:"type"`

	// Status of the condition, one of ('True', 'False', 'Unknown').
	Status corev1.ConditionStatus `json:"status"`

	// LastUpdateTime is the timestamp corresponding to the last status
	// update of this condition.
	// +optional
	LastUpdateTime *metav1.Time `json:"lastUpdateTime,omitempty"`

	// LastTransitionTime is the timestamp corresponding to the last status
	// change of this condition.
	// +optional
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty"`

	// Reason is a brief machine readable explanation for the condition's last
	// transition.
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message is a human readable description of the details of the last
	// transition, complementing reason.
	// +optional
	Message string `json:"message,omitempty"`
}

// ReactiveJobConditionType represents a ReactiveJob condition type.
type ReactiveJobConditionType string

const (
	// ReactiveJobConditionReady represents the fact that a given ReactiveJob is ready.
	ReactiveJobConditionTypeReady ReactiveJobConditionType = "Ready"
)

// ReactiveJobState represents a condition state in a readable form.
// +kubebuilder:validation:Enum=New;Processing;DeletionBlocked;Deleting;Ready;Error
type ReactiveJobState string

// These are valid condition states.
const (
	// Represents the fact that the ReactiveJob was first seen.
	ReactiveJobStateNew ReactiveJobState = "New"

	// Represents the fact that the ReactiveJob is reconciling.
	ReactiveJobStateProcessing ReactiveJobState = "Processing"

	// Represents the fact that the ReactiveJob should be deleted, but deletion is blocked.
	ReactiveJobStateDeletionBlocked ReactiveJobState = "DeletionBlocked"

	// Represents the fact that the ReactiveJob is being deleted.
	ReactiveJobStateDeleting ReactiveJobState = "Deleting"

	// Represents the fact that the ReactiveJob is ready.
	ReactiveJobStateReady ReactiveJobState = "Ready"

	// Represents the fact that the ReactiveJob has an error.
	ReactiveJobStateError ReactiveJobState = "Error"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +genclient

// ReactiveJob is the Schema for the reactivejobs API.
type ReactiveJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ReactiveJobSpec `json:"spec,omitempty"`
	// +kubebuilder:default={"observedGeneration":-1}
	Status ReactiveJobStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ReactiveJobList contains a list of ReactiveJob.
type ReactiveJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ReactiveJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ReactiveJob{}, &ReactiveJobList{})
}
