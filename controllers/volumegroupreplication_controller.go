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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	volrep "github.com/csi-addons/kubernetes-csi-addons/apis/replication.storage/v1alpha1"
	cachev1alpha1 "github.com/rakeshgm/volgroup-shim-operator/api/v1alpha1"
)

const storageProvisioner = "rook-ceph.rbd.csi.ceph.com"

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
	volGrouReplClass, err := r.getVolumeGroupReplicationClass(log, types.NamespacedName{
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
	v.volGroupReplicationClass = volGrouReplClass

	err = r.validatePVClabels(v.instance, log)
	if err != nil {
		setFailureCondition(v.instance)

		uErr := r.updateGroupReplicationStatus(v.instance, log, getCurrentReplicationState(v.instance), err.Error())
		if uErr != nil {
			log.Error(uErr, "failed to update validatePVClabels status", "VRName", v.instance.Name)
		}

		return ctrl.Result{}, nil
	}

	volRepClass, err := v.selectVolumeReplicationClass()

	if err != nil {
		setFailureCondition(v.instance)

		uErr := r.updateGroupReplicationStatus(v.instance, log, getCurrentReplicationState(v.instance), err.Error())
		if uErr != nil {
			log.Error(uErr, "failed to update volumeGroupReplication status", "VRName", v.instance.Name)
		}

		return ctrl.Result{}, nil
	}

	log.Info("found", "volRepClass", volRepClass.Name)

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

type VGRInstance struct {
	reconciler               *VolumeGroupReplicationReconciler
	ctx                      context.Context
	log                      logr.Logger
	instance                 *cachev1alpha1.VolumeGroupReplication
	savedInstanceStatus      cachev1alpha1.VolumeGroupReplicationStatus
	volGroupReplicationClass *cachev1alpha1.VolumeGroupReplicationClass
	namespacedName           string
	result                   ctrl.Result
}

// VolumeGroupReplicationReconciler get volume replication class object from the subjected namespace and return the same.
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

func (r *VolumeGroupReplicationReconciler) validatePVClabels(
	instance *cachev1alpha1.VolumeGroupReplication,
	logger logr.Logger,
) error {
	pvcLabelSelector := instance.Spec.Selector

	pvcSelector, err := metav1.LabelSelectorAsSelector(pvcLabelSelector)
	if err != nil {
		logger.Error(err, "error with PVC label selector", "pvcSelector", pvcLabelSelector)

		return fmt.Errorf("error with PVC label selector, %w", err)
	}
	logger.Info("Fetching PersistentVolumeClaims", "pvcSelector", pvcLabelSelector)

	listOptions := []client.ListOption{
		client.MatchingLabelsSelector{
			Selector: pvcSelector,
		},
	}

	pvcList := &corev1.PersistentVolumeClaimList{}
	if err := r.Client.List(context.TODO(), pvcList, listOptions...); err != nil {
		logger.Error(err, "failed to list PersistentVolumeClaims", "pvcSelector", pvcLabelSelector)

		return fmt.Errorf("failed to list PersistentVolumeClaims, %w", err)
	}

	logger.Info(fmt.Sprintf("Found %d PVCs using label selector %v", len(pvcList.Items), pvcLabelSelector))

	for idx := range pvcList.Items {
		logger.Info("PVC", "name", pvcList.Items[idx].Name, "namespace", pvcList.Items[idx].Namespace)
	}

	return nil
}

func (v *VGRInstance) listVolRepClasses() (*volrep.VolumeReplicationClassList, error) {

	replClassList := &volrep.VolumeReplicationClassList{}

	if err := v.reconciler.List(context.TODO(), replClassList); err != nil {
		v.log.Error(err, "Failed to list Replication Classes")

		return nil, fmt.Errorf("failed to list Replication Classes (%w)", err)
	}

	return replClassList, nil

}

func (v *VGRInstance) selectVolumeReplicationClass() (*volrep.VolumeReplicationClass, error) {

	schedulingIntervalFromVGRClass := v.volGroupReplicationClass.Spec.Parameters["schedulingInterval"]

	replClassList, err := v.listVolRepClasses()
	if err != nil {
		v.log.Info("unable to list volRepClasses")

		return nil, err
	}

	for index := range replClassList.Items {
		replicationClass := &replClassList.Items[index]

		if storageProvisioner != replicationClass.Spec.Provisioner {
			continue
		}

		schedulingInterval, found := replicationClass.Spec.Parameters["schedulingInterval"]
		if !found {
			// schedule not present in parameters of this replicationClass.
			continue
		}

		// ReplicationClass that matches both schedulingInterval from VGRClass and provisioner
		if schedulingInterval == schedulingIntervalFromVGRClass {
			v.log.Info(fmt.Sprintf("Found VolumeReplicationClass that matches provisioner and schedule %s/%s",
				storageProvisioner, schedulingIntervalFromVGRClass))

			return replicationClass, nil
		}
	}

	v.log.Info(fmt.Sprintf("No VolumeReplicationClass found to match provisioner and schedule %s/%s",
		storageProvisioner, schedulingIntervalFromVGRClass))

	return nil, fmt.Errorf("no VolumeReplicationClass found to match provisioner and schedule")
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
