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

	cachev1alpha1 "github.com/rakeshgm/volgroup-shim-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/go-logr/logr"
)

// getVolumeGroupReplicationClass get volume replication class object from the subjected namespace and return the same.
func (r *VolumeGroupReplicationReconciler) getVolumeGroupReplicationClass(logger logr.Logger,
	req types.NamespacedName) (*cachev1alpha1.VolumeGroupReplicationClass, error) {
	vrcObj := &cachev1alpha1.VolumeGroupReplicationClass{}

	err := r.Client.Get(context.TODO(), req, vrcObj)
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
