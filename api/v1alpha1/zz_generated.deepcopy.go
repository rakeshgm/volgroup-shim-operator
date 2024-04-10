//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VolumeGroupReplication) DeepCopyInto(out *VolumeGroupReplication) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VolumeGroupReplication.
func (in *VolumeGroupReplication) DeepCopy() *VolumeGroupReplication {
	if in == nil {
		return nil
	}
	out := new(VolumeGroupReplication)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VolumeGroupReplication) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VolumeGroupReplicationClass) DeepCopyInto(out *VolumeGroupReplicationClass) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VolumeGroupReplicationClass.
func (in *VolumeGroupReplicationClass) DeepCopy() *VolumeGroupReplicationClass {
	if in == nil {
		return nil
	}
	out := new(VolumeGroupReplicationClass)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VolumeGroupReplicationClass) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VolumeGroupReplicationClassList) DeepCopyInto(out *VolumeGroupReplicationClassList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]VolumeGroupReplicationClass, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VolumeGroupReplicationClassList.
func (in *VolumeGroupReplicationClassList) DeepCopy() *VolumeGroupReplicationClassList {
	if in == nil {
		return nil
	}
	out := new(VolumeGroupReplicationClassList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VolumeGroupReplicationClassList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VolumeGroupReplicationClassSpec) DeepCopyInto(out *VolumeGroupReplicationClassSpec) {
	*out = *in
	if in.Parameters != nil {
		in, out := &in.Parameters, &out.Parameters
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VolumeGroupReplicationClassSpec.
func (in *VolumeGroupReplicationClassSpec) DeepCopy() *VolumeGroupReplicationClassSpec {
	if in == nil {
		return nil
	}
	out := new(VolumeGroupReplicationClassSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VolumeGroupReplicationClassStatus) DeepCopyInto(out *VolumeGroupReplicationClassStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VolumeGroupReplicationClassStatus.
func (in *VolumeGroupReplicationClassStatus) DeepCopy() *VolumeGroupReplicationClassStatus {
	if in == nil {
		return nil
	}
	out := new(VolumeGroupReplicationClassStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VolumeGroupReplicationList) DeepCopyInto(out *VolumeGroupReplicationList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]VolumeGroupReplication, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VolumeGroupReplicationList.
func (in *VolumeGroupReplicationList) DeepCopy() *VolumeGroupReplicationList {
	if in == nil {
		return nil
	}
	out := new(VolumeGroupReplicationList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VolumeGroupReplicationList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VolumeGroupReplicationSpec) DeepCopyInto(out *VolumeGroupReplicationSpec) {
	*out = *in
	if in.Selector != nil {
		in, out := &in.Selector, &out.Selector
		*out = new(v1.LabelSelector)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VolumeGroupReplicationSpec.
func (in *VolumeGroupReplicationSpec) DeepCopy() *VolumeGroupReplicationSpec {
	if in == nil {
		return nil
	}
	out := new(VolumeGroupReplicationSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VolumeGroupReplicationStatus) DeepCopyInto(out *VolumeGroupReplicationStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.LastStartTime != nil {
		in, out := &in.LastStartTime, &out.LastStartTime
		*out = (*in).DeepCopy()
	}
	if in.LastCompletionTime != nil {
		in, out := &in.LastCompletionTime, &out.LastCompletionTime
		*out = (*in).DeepCopy()
	}
	if in.LastGroupSyncTime != nil {
		in, out := &in.LastGroupSyncTime, &out.LastGroupSyncTime
		*out = (*in).DeepCopy()
	}
	if in.LastGroupSyncDuration != nil {
		in, out := &in.LastGroupSyncDuration, &out.LastGroupSyncDuration
		*out = new(v1.Duration)
		**out = **in
	}
	if in.LastGroupSyncBytes != nil {
		in, out := &in.LastGroupSyncBytes, &out.LastGroupSyncBytes
		*out = new(int64)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VolumeGroupReplicationStatus.
func (in *VolumeGroupReplicationStatus) DeepCopy() *VolumeGroupReplicationStatus {
	if in == nil {
		return nil
	}
	out := new(VolumeGroupReplicationStatus)
	in.DeepCopyInto(out)
	return out
}
