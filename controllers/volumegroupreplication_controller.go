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
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cachev1alpha1 "github.com/rakeshgm/volgroup-shim-operator/api/v1alpha1"
)

// VolumeGroupReplicationReconciler reconciles a VolumeGroupReplication object
type VolumeGroupReplicationReconciler struct {
	client.Client
	APIReader client.Reader
	Scheme    *runtime.Scheme
}

//+kubebuilder:rbac:groups=cache.storage.ramendr.io,resources=volumegroupreplications,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cache.storage.ramendr.io,resources=volumegroupreplications/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cache.storage.ramendr.io,resources=volumegroupreplications/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the VolumeGroupReplication object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *VolumeGroupReplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	log := ctrl.Log.WithName("VolumeGroupReplicationReconciler")
	log.Info("Entering reconcile loop", "namespacedName", req.NamespacedName)

	defer log.Info("Exiting reconcile loop")

	// TODO(user): your logic here
	v := VGRInstance{
		reconciler:     r,
		ctx:            ctx,
		log:            log,
		instance:       &cachev1alpha1.VolumeGroupReplication{},
		namespacedName: req.NamespacedName.String(),
	}

	// Fetch the VolumeGroupReplication instance
	if err := r.APIReader.Get(ctx, req.NamespacedName, v.instance); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Resource not found")

			return ctrl.Result{}, nil
		}

		log.Error(err, "Failed to get resource")

		return ctrl.Result{}, fmt.Errorf("failed to reconcile VolumeGroupReplication (%v), %s",
			req.NamespacedName, err.Error())
	}

	// Get VolumeGroupReplicationClass
	_, err := v.getVolumeGroupReplicationClass(log, types.NamespacedName{
		Name:      v.instance.Spec.VolumeGroupReplicationClass,
		Namespace: req.Namespace,
	})
	if err != nil {
		setFailureCondition(v.instance)

		uErr := r.updateGroupReplicationStatus(v.instance, log, getCurrentReplicationState(v.instance), err.Error())
		if uErr != nil {
			log.Error(uErr, "failed to update volumeGroupReplication status", "VRName", v.instance.Name)
		}

		return ctrl.Result{}, nil
	}

	err = v.validatePVClabels()
	if err != nil {
		setFailureCondition(v.instance)

		uErr := r.updateGroupReplicationStatus(v.instance, log, getCurrentReplicationState(v.instance), err.Error())
		if uErr != nil {
			log.Error(uErr, "failed to update validatePVClabels status", "VRName", v.instance.Name)
		}

		return ctrl.Result{}, nil
	}

	v.instance.Status.LastCompletionTime = getCurrentTime()
	volState := getGroupReplicationState(v.instance)

	msg := fmt.Sprintf("volume is marked %s", string(volState))
	log.Info(msg)

	err = r.updateGroupReplicationStatus(v.instance, log, volState, msg)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VolumeGroupReplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1alpha1.VolumeGroupReplication{}).
		Complete(r)
}

func (r *VolumeGroupReplicationReconciler) updateGroupReplicationStatus(
	instance *cachev1alpha1.VolumeGroupReplication,
	logger logr.Logger,
	state cachev1alpha1.State,
	message string) error {
	instance.Status.State = state
	instance.Status.Message = message
	instance.Status.ObservedGeneration = instance.Generation
	if err := r.Client.Status().Update(context.TODO(), instance); err != nil {
		logger.Error(err, "failed to update status")

		return err
	}

	return nil
}

func setFailureCondition(instance *cachev1alpha1.VolumeGroupReplication) {
	switch instance.Spec.ReplicationState {
	case cachev1alpha1.Primary:
		setFailedPromotionCondition(&instance.Status.Conditions, instance.Generation)
	case cachev1alpha1.Secondary:
		setFailedDemotionCondition(&instance.Status.Conditions, instance.Generation)
	}
}

func getGroupReplicationState(instance *cachev1alpha1.VolumeGroupReplication) cachev1alpha1.State {
	switch instance.Spec.ReplicationState {
	case cachev1alpha1.Primary:
		return cachev1alpha1.PrimaryState
	case cachev1alpha1.Secondary:
		return cachev1alpha1.SecondaryState
	}

	return cachev1alpha1.UnknownState
}

func getCurrentReplicationState(instance *cachev1alpha1.VolumeGroupReplication) cachev1alpha1.State {
	if instance.Status.State == "" {
		return cachev1alpha1.UnknownState
	}

	return instance.Status.State
}

func getCurrentTime() *metav1.Time {
	metav1NowTime := metav1.NewTime(time.Now())

	return &metav1NowTime
}
