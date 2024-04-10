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

package controllers

import (
	"time"

	volGroupRep "github.com/rakeshgm/volgroup-shim-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ConditionCompleted = "Completed"
	ConditionDegraded  = "Degraded"
	ConditionResyncing = "Resyncing"
)

const (
	Success = "Success"
	// used when replication state is primary
	Promoted        = "Promoted"
	FailedToPromote = "FailedToPromote"
	// used when replication state is secondary
	Demoted        = "Demoted"
	FailedToDemote = "FailedToDemote"

	Error          = "Error"
	VolumeDegraded = "VolumeDegraded"
	Healthy        = "Healthy"

	ResyncTriggered = "ResyncTriggered"
	FailedToResync  = "FailedToResync"
	NotResyncing    = "NotResyncing"
	Unknown         = "Unknown"
)

func setStatusCondition(existingConditions *[]metav1.Condition, newCondition *metav1.Condition) {
	if existingConditions == nil {
		existingConditions = &[]metav1.Condition{}
	}

	existingCondition := findCondition(*existingConditions, newCondition.Type)
	if existingCondition == nil {
		newCondition.LastTransitionTime = metav1.NewTime(time.Now())
		*existingConditions = append(*existingConditions, *newCondition)

		return
	}

	if existingCondition.Status != newCondition.Status {
		existingCondition.Status = newCondition.Status
		existingCondition.LastTransitionTime = metav1.NewTime(time.Now())
	}

	existingCondition.Reason = newCondition.Reason
	existingCondition.ObservedGeneration = newCondition.ObservedGeneration
}

func findCondition(existingConditions []metav1.Condition, conditionType string) *metav1.Condition {
	for i := range existingConditions {
		if existingConditions[i].Type == conditionType {
			return &existingConditions[i]
		}
	}

	return nil
}

func setStatusConditionCompleted(conditions *[]metav1.Condition, reason string, observedGeneration int64,
	cStatus metav1.ConditionStatus,
) {
	setStatusCondition(conditions, &metav1.Condition{
		Type:               ConditionCompleted,
		Reason:             reason,
		ObservedGeneration: observedGeneration,
		Status:             cStatus,
	})
}

func setStatusConditionDegrated(conditions *[]metav1.Condition, reason string, observedGeneration int64,
	cStatus metav1.ConditionStatus,
) {
	setStatusCondition(conditions, &metav1.Condition{
		Type:               ConditionDegraded,
		Reason:             reason,
		ObservedGeneration: observedGeneration,
		Status:             cStatus,
	})
}

func setStatusConditionResyncing(conditions *[]metav1.Condition, reason string, observedGeneration int64,
	cStatus metav1.ConditionStatus,
) {
	setStatusCondition(conditions, &metav1.Condition{
		Type:               ConditionResyncing,
		Reason:             reason,
		ObservedGeneration: observedGeneration,
		Status:             cStatus,
	})
}

func setConditionsToUnknown(conditions *[]metav1.Condition, observedGeneration int64) {
	setStatusConditionCompleted(conditions, Unknown, observedGeneration, metav1.ConditionUnknown)
	setStatusConditionDegrated(conditions, Unknown, observedGeneration, metav1.ConditionUnknown)
	setStatusConditionResyncing(conditions, Unknown, observedGeneration, metav1.ConditionUnknown)
}

// sets conditions when volume promotion was failed.
func setFailedPromotionCondition(conditions *[]metav1.Condition, observedGeneration int64) {
	setStatusConditionCompleted(conditions, FailedToPromote, observedGeneration, metav1.ConditionFalse)
	setStatusConditionDegrated(conditions, Error, observedGeneration, metav1.ConditionTrue)
	setStatusConditionResyncing(conditions, Error, observedGeneration, metav1.ConditionFalse)
}

// sets conditions when volume demotion was failed.
func setFailedDemotionCondition(conditions *[]metav1.Condition, observedGeneration int64) {
	setStatusConditionCompleted(conditions, FailedToDemote, observedGeneration, metav1.ConditionFalse)
	setStatusConditionDegrated(conditions, Error, observedGeneration, metav1.ConditionTrue)
	setStatusConditionResyncing(conditions, Error, observedGeneration, metav1.ConditionFalse)
}

func updateStatusConditonCompleted(instance *volGroupRep.VolumeGroupReplication, newCondition *metav1.Condition) {
	switch instance.Spec.ReplicationState {
	case volGroupRep.Primary:
		updateStatusConditionCompletedForPrimaryState(&instance.Status.Conditions, newCondition, instance.Generation)
	case volGroupRep.Secondary:
		updateStatusConditionCompletedForSecondaryState(&instance.Status.Conditions, newCondition, instance.Generation)
	}
}

func updateStatusConditionCompletedForPrimaryState(existingConditions *[]metav1.Condition,
	newCondition *metav1.Condition, observedGeneration int64,
) {
	if newCondition.Status == metav1.ConditionFalse {
		setStatusConditionCompleted(existingConditions, FailedToPromote, observedGeneration, metav1.ConditionFalse)
	} else {
		setStatusConditionCompleted(existingConditions, Promoted, observedGeneration, metav1.ConditionTrue)
	}
}

func updateStatusConditionCompletedForSecondaryState(existingConditions *[]metav1.Condition,
	newCondition *metav1.Condition, observedGeneration int64,
) {
	if newCondition.Status == metav1.ConditionFalse {
		setStatusConditionCompleted(existingConditions, FailedToDemote, observedGeneration, metav1.ConditionFalse)
	} else {
		setStatusConditionCompleted(existingConditions, Demoted, observedGeneration, metav1.ConditionTrue)
	}
}

func updateStatusConditionDegraded(existingConditions *[]metav1.Condition,
	newCondition *metav1.Condition, observedGeneration int64,
) {
	if newCondition.Status == metav1.ConditionTrue {
		setStatusConditionDegrated(existingConditions, VolumeDegraded, observedGeneration, metav1.ConditionTrue)
	} else {
		setStatusConditionDegrated(existingConditions, Healthy, observedGeneration, metav1.ConditionFalse)
	}
}

func updateStatusConditionResyncing(existingConditions *[]metav1.Condition,
	newCondition *metav1.Condition, observedGeneration int64,
) {
	if newCondition.Status == metav1.ConditionFalse {
		setStatusConditionResyncing(existingConditions, NotResyncing, observedGeneration, metav1.ConditionFalse)
	} else {
		setStatusConditionResyncing(existingConditions, ResyncTriggered, observedGeneration, metav1.ConditionTrue)
	}
}
