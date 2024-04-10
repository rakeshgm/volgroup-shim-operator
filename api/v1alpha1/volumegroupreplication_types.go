/*
Copyright 2024.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ReplicationState represents the replication operations to be performed on the volume
type ReplicationState string

const (
	// Promote the protected PVCs to primary
	Primary ReplicationState = "primary"

	// Demote the proteced PVCs to secondary
	Secondary ReplicationState = "secondary"
)

type State string

const (
	PrimaryState   State = "Primary"
	SecondaryState State = "Secondary"
	UnknownState   State = "Unknown"
)

// VolumeGroupReplicationSpec defines the desired state of VolumeGroupReplication
type VolumeGroupReplicationSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// Desired state of all volumes [primary or secondary] in this replication group;
	// this value is propagated to children VolumeReplication CRs
	// +kubebuilder:validation:Enum=primary;secondary
	ReplicationState ReplicationState `json:"replicationState"`

	// VolumeGroupReplicationClass may be left nil to indicate that
	// the default class will be used.
	VolumeGroupReplicationClass string `json:"volumeGroupReplicationClass"`

	// A label query over persistent volume claims to be grouped together
	// for replication. This labelSelector will be used to match
	// the label added to a PVC.
	Selector *metav1.LabelSelector `json:"selector"`
}

// VolumeGroupReplicationStatus defines the observed state of VolumeGroupReplication
type VolumeGroupReplicationStatus struct {
	// Important: Run "make" to regenerate code after modifying this file

	State   State  `json:"state,omitempty"`
	Message string `json:"message,omitempty"`
	// Conditions are the list of conditions and their status.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// observedGeneration is the last generation change the operator has dealt with
	// +optional
	ObservedGeneration int64        `json:"observedGeneration,omitempty"`
	LastStartTime      *metav1.Time `json:"lastStartTime,omitempty"`

	LastCompletionTime *metav1.Time `json:"lastCompletionTime,omitempty"`
	// lastGroupSyncTime is the time of the most recent successful synchronization of all PVCs
	//+optional
	LastGroupSyncTime *metav1.Time `json:"lastGroupSyncTime,omitempty"`

	// lastGroupSyncDuration is the max time from all the successful synced PVCs
	//+optional
	LastGroupSyncDuration *metav1.Duration `json:"lastGroupSyncDuration,omitempty"`

	// lastGroupSyncBytes is the total bytes transferred from the most recent
	// successful synchronization of all PVCs
	//+optional
	LastGroupSyncBytes *int64 `json:"lastGroupSyncBytes,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// VolumeGroupReplication is the Schema for the volumegroupreplications API
type VolumeGroupReplication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VolumeGroupReplicationSpec   `json:"spec,omitempty"`
	Status VolumeGroupReplicationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VolumeGroupReplicationList contains a list of VolumeGroupReplication
type VolumeGroupReplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VolumeGroupReplication `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VolumeGroupReplication{}, &VolumeGroupReplicationList{})
}
