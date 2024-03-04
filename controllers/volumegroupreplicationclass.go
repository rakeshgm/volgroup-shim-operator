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
	"context"
	"fmt"

	cachev1alpha1 "github.com/rakeshgm/volgroup-shim-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
)

type VGRInstance struct {
	reconciler          *VolumeGroupReplicationReconciler
	ctx                 context.Context
	log                 logr.Logger
	instance            *cachev1alpha1.VolumeGroupReplication
	savedInstanceStatus cachev1alpha1.VolumeGroupReplicationStatus
	namespacedName      string
	result              ctrl.Result
}

// VGRInstance get volume replication class object from the subjected namespace and return the same.
func (v *VGRInstance) getVolumeGroupReplicationClass(logger logr.Logger,
	req types.NamespacedName) (*cachev1alpha1.VolumeGroupReplicationClass, error) {
	vrcObj := &cachev1alpha1.VolumeGroupReplicationClass{}

	err := v.reconciler.Client.Get(context.TODO(), req, vrcObj)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Error(err, "VolumeGroupReplicationClass not found", "VolumeGroupReplicationClass", req.Name)
		} else {
			logger.Error(err, "Got an unexpected error while fetching VolumeGroupReplicationClass", "VolumeGroupReplicationClass", req.Name)
		}

		return nil, err
	}

	return vrcObj, nil
}

func (v *VGRInstance) validatePVClabels() error {
	pvcLabelSelector := v.instance.Spec.Selector

	pvcSelector, err := metav1.LabelSelectorAsSelector(pvcLabelSelector)
	if err != nil {
		v.log.Error(err, "error with PVC label selector", "pvcSelector", pvcLabelSelector)

		return fmt.Errorf("error with PVC label selector, %w", err)
	}
	v.log.Info("Fetching PersistentVolumeClaims", "pvcSelector", pvcLabelSelector)

	listOptions := []client.ListOption{
		client.MatchingLabelsSelector{
			Selector: pvcSelector,
		},
	}

	pvcList := &corev1.PersistentVolumeClaimList{}
	if err := v.reconciler.Client.List(context.TODO(), pvcList, listOptions...); err != nil {
		v.log.Error(err, "failed to list PersistentVolumeClaims", "pvcSelector", pvcLabelSelector)

		return fmt.Errorf("failed to list PersistentVolumeClaims, %w", err)
	}

	v.log.Info(fmt.Sprintf("Found %d PVCs using label selector %v", len(pvcList.Items), pvcLabelSelector))

	for idx := range pvcList.Items {
		v.log.Info("PVC", "name", pvcList.Items[idx].Name, "namespace", pvcList.Items[idx].Namespace)
	}

	return nil
}
