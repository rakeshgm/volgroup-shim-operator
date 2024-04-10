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
	// "github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	volRep "github.com/csi-addons/kubernetes-csi-addons/apis/replication.storage/v1alpha1"
	volGroupRep "github.com/rakeshgm/volgroup-shim-operator/api/v1alpha1"
)

const storageProvisioner = "rook-ceph.rbd.csi.ceph.com"

// VolumeGroupReplicationReconciler reconciles a VolumeGroupReplication object
type VolumeGroupReplicationReconciler struct {
	client.Client
	APIReader client.Reader
	Scheme    *runtime.Scheme
	Log       logr.Logger
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

type VGRInstance struct {
	reconciler               *VolumeGroupReplicationReconciler
	ctx                      context.Context
	log                      logr.Logger
	instance                 *volGroupRep.VolumeGroupReplication
	savedInstanceStatus      volGroupRep.VolumeGroupReplicationStatus
	volGroupReplicationClass *volGroupRep.VolumeGroupReplicationClass
	volRepClass              *volRep.VolumeReplicationClass
	volRepPVCs               []corev1.PersistentVolumeClaim
	volReps                  []volRep.VolumeReplication
	namespacedName           string
}

func (r *VolumeGroupReplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// add uuid to log if needed, removed now to make log more readable
	log := r.Log.WithValues("VGR",
		req.NamespacedName,
		// "rid", uuid.New(),
	)
	log.Info("Entering reconcile loop", "namespacedName", req.NamespacedName)

	defer log.Info("Exiting reconcile loop")

	// TODO(user): your logic here
	v := VGRInstance{
		reconciler:     r,
		ctx:            ctx,
		log:            log,
		instance:       &volGroupRep.VolumeGroupReplication{},
		namespacedName: req.NamespacedName.Namespace,
	}

	// Fetch the VolumeGroupReplication instance
	if err := r.APIReader.Get(ctx, req.NamespacedName, v.instance); err != nil {
		if k8serrors.IsNotFound(err) {
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

		uErr := r.updateVGRStatus(v.instance, log, getCurrentReplicationState(v.instance), err.Error())
		if uErr != nil {
			log.Error(uErr, "failed to update volumeGroupReplication status", "VRName", v.instance.Name)
		}

		return ctrl.Result{}, nil
	}
	v.volGroupReplicationClass = volGrouReplClass

	// set initial conditions as unknown
	if len(v.instance.Status.Conditions) == 0 {
		setConditionsToUnknown(&v.instance.Status.Conditions, v.instance.Generation)
		message := "initial conditions set"
		log.Info(message)
		uErr := r.updateVGRStatus(v.instance, log, getCurrentReplicationState(v.instance), message)
		if uErr != nil {
			log.Error(uErr, "failed to set volumeGroupReplication initial status conditions", "VRName", v.instance.Name)
			return ctrl.Result{}, uErr
		}
	}

	// list PVCs
	pvcList, err := r.listPVCs(v.instance, log)
	if err != nil {
		log.Error(err, "failed in listing PVCs")
		setFailureCondition(v.instance)

		uErr := r.updateVGRStatus(v.instance, log, getCurrentReplicationState(v.instance), err.Error())
		if uErr != nil {
			log.Error(uErr, "failed to update volumeGroupReplication status", "VRName", v.instance.Name)
		}

		return ctrl.Result{}, nil
	}
	v.volRepPVCs = pvcList.Items

	// get appropriate VolRepClass
	volRepClass, err := v.selectVolumeReplicationClass()
	if err != nil {
		log.Error(err, "error in seleting VolumeReplicationClass")
		setFailureCondition(v.instance)

		uErr := r.updateVGRStatus(v.instance, log, getCurrentReplicationState(v.instance), err.Error())
		if uErr != nil {
			log.Error(uErr, "failed to update volumeGroupReplication status", "VRName", v.instance.Name)
		}

		return ctrl.Result{}, nil
	}
	v.volRepClass = volRepClass

	if err := v.processVolRep(); err != nil {
		log.Error(err, "error in reconciling VolumeReplication for PVCs")

		setFailureCondition(v.instance)

		uErr := r.updateVGRStatus(v.instance, log, getCurrentReplicationState(v.instance), err.Error())
		if uErr != nil {
			log.Error(uErr, "failed to update volumeGroupReplication status", "VRName", v.instance.Name)
		}

		return ctrl.Result{}, nil
	}

	volRepList, err := v.listVolReps()
	if err != nil {
		log.Error(err, "error in listing VolumeReplication Resources")

		setFailureCondition(v.instance)

		uErr := r.updateVGRStatus(v.instance, log, getCurrentReplicationState(v.instance), err.Error())
		if uErr != nil {
			log.Error(uErr, "failed to update volumeGroupReplication status", "VRName", v.instance.Name)
		}

		return ctrl.Result{}, nil
	}
	v.volReps = *volRepList

	v.UpdateVGRConditions()
	v.updateVGRLastGroupSyncTime()
	v.updateVGRLastGroupSyncDuration()
	v.updateVGRLastGroupSyncBytes()

	v.instance.Status.LastCompletionTime = getCurrentTime()
	volState := getGroupReplicationState(v.instance)

	msg := fmt.Sprintf("volume is marked %s", string(volState))
	log.Info(msg)

	err = r.updateVGRStatus(v.instance, log, volState, msg)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VolumeGroupReplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&volGroupRep.VolumeGroupReplication{}).
		Owns(&volRep.VolumeReplication{}).
		Complete(r)
}

func (r *VolumeGroupReplicationReconciler) updateVGRStatus(
	instance *volGroupRep.VolumeGroupReplication,
	logger logr.Logger,
	state volGroupRep.State,
	message string,
) error {
	instance.Status.State = state
	instance.Status.Message = message
	instance.Status.ObservedGeneration = instance.Generation
	if err := r.Client.Status().Update(context.TODO(), instance); err != nil {
		logger.Error(err, "failed to update status")

		return err
	}

	return nil
}

// VolumeGroupReplicationReconciler get volume replication class object from the subjected namespace and return the same.
func (r *VolumeGroupReplicationReconciler) getVolumeGroupReplicationClass(logger logr.Logger,
	req types.NamespacedName,
) (*volGroupRep.VolumeGroupReplicationClass, error) {
	vrcObj := &volGroupRep.VolumeGroupReplicationClass{}

	err := r.Client.Get(context.TODO(), req, vrcObj)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			logger.Error(err, "VolumeGroupReplicationClass not found", "VolumeGroupReplicationClass", req.Name)
		} else {
			logger.Error(err, "Got an unexpected error while fetching VolumeGroupReplicationClass", "VolumeGroupReplicationClass", req.Name)
		}

		return nil, err
	}

	return vrcObj, nil
}

func (r *VolumeGroupReplicationReconciler) listPVCs(
	instance *volGroupRep.VolumeGroupReplication,
	logger logr.Logger,
) (*corev1.PersistentVolumeClaimList, error) {
	pvcLabelSelector := instance.Spec.Selector

	pvcSelector, err := metav1.LabelSelectorAsSelector(pvcLabelSelector)
	if err != nil {
		logger.Error(err, "error with PVC label selector", "pvcSelector", pvcLabelSelector)

		return nil, fmt.Errorf("error with PVC label selector, %w", err)
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

		return nil, fmt.Errorf("failed to list PersistentVolumeClaims, %w", err)
	}

	logger.Info(fmt.Sprintf("Found %d PVCs using label selector %v", len(pvcList.Items), pvcLabelSelector))
	for idx := range pvcList.Items {
		logger.Info("PVC", "name", pvcList.Items[idx].Name, "namespace", pvcList.Items[idx].Namespace)
	}

	return pvcList, nil
}

func (v *VGRInstance) listVolRepClasses() (*volRep.VolumeReplicationClassList, error) {
	replClassList := &volRep.VolumeReplicationClassList{}
	if err := v.reconciler.List(context.TODO(), replClassList); err != nil {
		v.log.Error(err, "Failed to list Replication Classes")
		return nil, fmt.Errorf("failed to list Replication Classes (%w)", err)
	}

	return replClassList, nil
}

func (v *VGRInstance) selectVolumeReplicationClass() (*volRep.VolumeReplicationClass, error) {
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
			v.log.Info(fmt.Sprintf("Found VolumeReplicationClass: (%s) that matches provisioner and schedule %s/%s",
				replicationClass.Name, storageProvisioner, schedulingIntervalFromVGRClass))

			return replicationClass, nil
		}
	}

	v.log.Info(fmt.Sprintf("No VolumeReplicationClass found to match provisioner and schedule %s/%s",
		storageProvisioner, schedulingIntervalFromVGRClass))

	return nil, fmt.Errorf("no VolumeReplicationClass found to match provisioner and schedule")
}

func (v *VGRInstance) setVROwner(volRep *volRep.VolumeReplication, owner client.Object) (bool, error) {
	const updated = true

	for _, ownerReference := range volRep.GetOwnerReferences() {
		if ownerReference.Name == owner.GetName() {
			return !updated, nil
		}
	}

	err := ctrl.SetControllerReference(owner, volRep, v.reconciler.Client.Scheme())
	if err != nil {
		return !updated, fmt.Errorf("failed to set VolRep owner %w", err)
	}

	err = v.updateVR(volRep)
	if err != nil {
		return !updated, fmt.Errorf("failed to update VolRep %s (%w)", volRep.GetName(), err)
	}

	v.log.Info(fmt.Sprintf("Object %s owns VolRep %s", owner.GetName(), volRep.GetName()))
	return updated, nil
}

func (v *VGRInstance) updateVR(volRep *volRep.VolumeReplication) error {
	v.log.Info("update VolRep")

	if err := v.reconciler.Update(v.ctx, volRep); err != nil {
		v.log.Error(err, "Failed to update VolumeReplication resource",
			"name", volRep.Name, "namespace", volRep.Namespace)
		return err
	}

	v.log.Info(fmt.Sprintf("Updated VolumeReplication resource (%s/%s)",
		volRep.Name, volRep.Namespace))

	return nil
}

func (v *VGRInstance) setOrUpdateVROwner(volRep *volRep.VolumeReplication) error {
	updated, err := v.setVROwner(volRep, v.instance)
	if err != nil {
		return fmt.Errorf("failed to setOnwer to VolRep: %w", err)
	}

	if !updated {
		v.log.Info("owner has already been set")
	}

	return nil
}

func (v *VGRInstance) createVR(vrNamespacedName types.NamespacedName, state volRep.ReplicationState) error {
	v.log.Info("createVR")

	volRep := &volRep.VolumeReplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vrNamespacedName.Name,
			Namespace: vrNamespacedName.Namespace,
		},
		Spec: volRep.VolumeReplicationSpec{
			DataSource: corev1.TypedLocalObjectReference{
				Kind:     "PersistentVolumeClaim",
				Name:     vrNamespacedName.Name,
				APIGroup: new(string),
			},
			ReplicationState:       state,
			VolumeReplicationClass: v.volRepClass.GetName(),
			AutoResync:             v.autoResync(state),
		},
	}

	v.log.Info("Creating VolumeReplication resource", "resource", volRep)
	if err := v.reconciler.Create(v.ctx, volRep); err != nil {
		return fmt.Errorf("failed to create VolumeReplication resource (%s), %w", vrNamespacedName, err)
	}

	return v.setOrUpdateVROwner(volRep)
}

func (v *VGRInstance) createOrUpdateVR(vrNamespacedName types.NamespacedName,
) error {
	v.log.Info("createOrUpdate VR")

	state := volRep.ReplicationState(v.instance.Spec.ReplicationState)
	volRepObj := &volRep.VolumeReplication{}

	err := v.reconciler.Get(v.ctx, vrNamespacedName, volRepObj)
	if err != nil {

		// if resource is not found, proceed to createVR.
		// exit if any error other than notFound occurs
		if !k8serrors.IsNotFound(err) {
			v.log.Error(err, "Failed to get VolumeReplication resource", "resource", vrNamespacedName)

			return fmt.Errorf("failed to get VolumeReplication resource"+" (%s/%s), %w",
				vrNamespacedName.Namespace, vrNamespacedName.Name, err)
		}

		// create VR for PVC
		if err = v.createVR(vrNamespacedName, state); err != nil {
			v.log.Error(err, "Failed to create VolumeReplication resource", "resource", vrNamespacedName)

			return fmt.Errorf("failed to create VolumeReplication resource"+" (%s/%s), %w",
				vrNamespacedName.Namespace, vrNamespacedName.Name, err)
		}
		return nil
	}

	if volRepObj.Spec.ReplicationState == state {
		v.log.Info("volumeReplication and volumeGroupReplication state match, no need to update",
			volRepObj.Name, volRepObj.Namespace)
		return nil
	}

	return v.updateVolRepState(volRepObj, state)
}

func (v *VGRInstance) updateVolRepState(volRep *volRep.VolumeReplication, state volRep.ReplicationState) error {
	v.log.Info("updating volRep:", volRep.Name, volRep.Namespace)
	volRep.Spec.ReplicationState = state
	volRep.Spec.AutoResync = v.autoResync(state)
	err := v.updateVR(volRep)
	if err != nil {
		return fmt.Errorf("failed to update VolRep %s (%w)", volRep.GetName(), err)
	}
	v.log.Info("volumeReplication Updated")

	return v.setOrUpdateVROwner(volRep)
}

func (v *VGRInstance) autoResync(state volRep.ReplicationState) bool {
	return state == volRep.Secondary
}

func (v *VGRInstance) processVolRep() error {
	v.log.Info("processing VolRep")
	for i := range v.volRepPVCs {
		pvc := &v.volRepPVCs[i]
		pvcNamespacedName := types.NamespacedName{Name: pvc.Name, Namespace: pvc.Namespace}

		err := v.createOrUpdateVR(pvcNamespacedName)
		if err != nil {
			v.log.Info("Failure in creating or updating VolumeReplication resource for PVC",
				"errorValue", err)
			return err
		}
	}

	return nil
}

func (v *VGRInstance) listVolReps() (*[]volRep.VolumeReplication, error) {
	v.log.Info("list volReps")

	volRepList := &volRep.VolumeReplicationList{}
	listOptions := &client.ListOptions{
		Namespace: v.namespacedName,
	}

	if err := v.reconciler.List(context.TODO(), volRepList, listOptions); err != nil {
		v.log.Error(err, "failed to list VolReps", "in namespace", v.namespacedName)
		return nil, fmt.Errorf("failed to list VolRepList, %w", err)
	}

	var processedVolRepList []volRep.VolumeReplication

	for indx := range volRepList.Items {
		vR := volRepList.Items[indx]
		for _, ownerReference := range vR.GetOwnerReferences() {
			if ownerReference.UID == v.instance.GetUID() {
				processedVolRepList = append(processedVolRepList, vR)
			}
		}
	}

	v.log.Info("volReps", "found:", len(processedVolRepList))

	return &processedVolRepList, nil
}

func (v *VGRInstance) UpdateVGRConditions() {
	v.log.Info("updating VGR conditions")

	for indx := range v.volReps {
		volRep := v.volReps[indx]
		v.log.Info("checking condition of volRep", "name", volRep.Name)

		for i := range volRep.Status.Conditions {
			condition := volRep.Status.Conditions[i]
			v.log.Info("got condition type and status", condition.Type, condition.Status)
			switch condition.Type {
			case ConditionCompleted:
				updateStatusConditonCompleted(v.instance, &condition)
			case ConditionDegraded:
				updateStatusConditionDegraded(&v.instance.Status.Conditions, &condition, v.instance.Generation)
			case ConditionResyncing:
				updateStatusConditionResyncing(&v.instance.Status.Conditions, &condition, v.instance.Generation)
			}
		}
	}
}

func (v *VGRInstance) updateVGRLastGroupSyncTime() {
	v.log.Info("updating lastGroupSyncTime")
	var leastLastSyncTime *metav1.Time

	for indx := range v.volReps {
		volRep := v.volReps[indx].Status
		// If any protected PVC reports nil, report that back (no sync time available)
		if volRep.LastSyncTime == nil {
			leastLastSyncTime = nil

			break
		}

		if leastLastSyncTime == nil {
			leastLastSyncTime = volRep.LastSyncTime

			continue
		}

		if volRep.LastSyncTime != nil && volRep.LastSyncTime.Before(leastLastSyncTime) {
			leastLastSyncTime = volRep.LastSyncTime
		}
	}

	v.instance.Status.LastGroupSyncTime = leastLastSyncTime
}

func (v *VGRInstance) updateVGRLastGroupSyncDuration() {
	v.log.Info("updating lastGroupSyncDuration")
	var maxLastSyncDuration *metav1.Duration

	for indx := range v.volReps {
		volRep := v.volReps[indx].Status
		if maxLastSyncDuration == nil && volRep.LastSyncDuration != nil {
			maxLastSyncDuration = new(metav1.Duration)
			*maxLastSyncDuration = *volRep.LastSyncDuration

			continue
		}

		if volRep.LastSyncDuration != nil &&
			volRep.LastSyncDuration.Duration > maxLastSyncDuration.Duration {
			*maxLastSyncDuration = *volRep.LastSyncDuration
		}
	}

	v.instance.Status.LastGroupSyncDuration = maxLastSyncDuration
}

func (v *VGRInstance) updateVGRLastGroupSyncBytes() {
	v.log.Info("updating lastGroupSyncBytes")
	var totalLastSyncBytes *int64

	for indx := range v.volReps {
		volRep := v.volReps[indx].Status
		if totalLastSyncBytes == nil && volRep.LastSyncBytes != nil {
			totalLastSyncBytes = new(int64)
			*totalLastSyncBytes = *volRep.LastSyncBytes

			continue
		}

		if volRep.LastSyncBytes != nil {
			*totalLastSyncBytes += *volRep.LastSyncBytes
		}
	}

	v.instance.Status.LastGroupSyncBytes = totalLastSyncBytes
}

func setFailureCondition(instance *volGroupRep.VolumeGroupReplication) {
	switch instance.Spec.ReplicationState {
	case volGroupRep.Primary:
		setFailedPromotionCondition(&instance.Status.Conditions, instance.Generation)
	case volGroupRep.Secondary:
		setFailedDemotionCondition(&instance.Status.Conditions, instance.Generation)
	}
}

func getGroupReplicationState(instance *volGroupRep.VolumeGroupReplication) volGroupRep.State {
	switch instance.Spec.ReplicationState {
	case volGroupRep.Primary:
		return volGroupRep.PrimaryState
	case volGroupRep.Secondary:
		return volGroupRep.SecondaryState
	}

	return volGroupRep.UnknownState
}

func getCurrentReplicationState(instance *volGroupRep.VolumeGroupReplication) volGroupRep.State {
	if instance.Status.State == "" {
		return volGroupRep.UnknownState
	}

	return instance.Status.State
}

func getCurrentTime() *metav1.Time {
	metav1NowTime := metav1.NewTime(time.Now())

	return &metav1NowTime
}
