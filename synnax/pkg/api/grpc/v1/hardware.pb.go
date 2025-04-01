// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        (unknown)
// source: synnax/pkg/api/grpc/v1/hardware.proto

package v1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Rack struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Key           uint32                 `protobuf:"varint,1,opt,name=key,proto3" json:"key,omitempty"`
	Name          string                 `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Rack) Reset() {
	*x = Rack{}
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Rack) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Rack) ProtoMessage() {}

func (x *Rack) ProtoReflect() protoreflect.Message {
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Rack.ProtoReflect.Descriptor instead.
func (*Rack) Descriptor() ([]byte, []int) {
	return file_synnax_pkg_api_grpc_v1_hardware_proto_rawDescGZIP(), []int{0}
}

func (x *Rack) GetKey() uint32 {
	if x != nil {
		return x.Key
	}
	return 0
}

func (x *Rack) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

type HardwareCreateRackRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Racks         []*Rack                `protobuf:"bytes,1,rep,name=racks,proto3" json:"racks,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *HardwareCreateRackRequest) Reset() {
	*x = HardwareCreateRackRequest{}
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *HardwareCreateRackRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HardwareCreateRackRequest) ProtoMessage() {}

func (x *HardwareCreateRackRequest) ProtoReflect() protoreflect.Message {
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HardwareCreateRackRequest.ProtoReflect.Descriptor instead.
func (*HardwareCreateRackRequest) Descriptor() ([]byte, []int) {
	return file_synnax_pkg_api_grpc_v1_hardware_proto_rawDescGZIP(), []int{1}
}

func (x *HardwareCreateRackRequest) GetRacks() []*Rack {
	if x != nil {
		return x.Racks
	}
	return nil
}

type HardwareCreateRackResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Racks         []*Rack                `protobuf:"bytes,1,rep,name=racks,proto3" json:"racks,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *HardwareCreateRackResponse) Reset() {
	*x = HardwareCreateRackResponse{}
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *HardwareCreateRackResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HardwareCreateRackResponse) ProtoMessage() {}

func (x *HardwareCreateRackResponse) ProtoReflect() protoreflect.Message {
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HardwareCreateRackResponse.ProtoReflect.Descriptor instead.
func (*HardwareCreateRackResponse) Descriptor() ([]byte, []int) {
	return file_synnax_pkg_api_grpc_v1_hardware_proto_rawDescGZIP(), []int{2}
}

func (x *HardwareCreateRackResponse) GetRacks() []*Rack {
	if x != nil {
		return x.Racks
	}
	return nil
}

type HardwareRetrieveRackRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Keys          []uint32               `protobuf:"varint,1,rep,packed,name=keys,proto3" json:"keys,omitempty"`
	Names         []string               `protobuf:"bytes,2,rep,name=names,proto3" json:"names,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *HardwareRetrieveRackRequest) Reset() {
	*x = HardwareRetrieveRackRequest{}
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *HardwareRetrieveRackRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HardwareRetrieveRackRequest) ProtoMessage() {}

func (x *HardwareRetrieveRackRequest) ProtoReflect() protoreflect.Message {
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HardwareRetrieveRackRequest.ProtoReflect.Descriptor instead.
func (*HardwareRetrieveRackRequest) Descriptor() ([]byte, []int) {
	return file_synnax_pkg_api_grpc_v1_hardware_proto_rawDescGZIP(), []int{3}
}

func (x *HardwareRetrieveRackRequest) GetKeys() []uint32 {
	if x != nil {
		return x.Keys
	}
	return nil
}

func (x *HardwareRetrieveRackRequest) GetNames() []string {
	if x != nil {
		return x.Names
	}
	return nil
}

type HardwareRetrieveRackResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Racks         []*Rack                `protobuf:"bytes,1,rep,name=racks,proto3" json:"racks,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *HardwareRetrieveRackResponse) Reset() {
	*x = HardwareRetrieveRackResponse{}
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *HardwareRetrieveRackResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HardwareRetrieveRackResponse) ProtoMessage() {}

func (x *HardwareRetrieveRackResponse) ProtoReflect() protoreflect.Message {
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HardwareRetrieveRackResponse.ProtoReflect.Descriptor instead.
func (*HardwareRetrieveRackResponse) Descriptor() ([]byte, []int) {
	return file_synnax_pkg_api_grpc_v1_hardware_proto_rawDescGZIP(), []int{4}
}

func (x *HardwareRetrieveRackResponse) GetRacks() []*Rack {
	if x != nil {
		return x.Racks
	}
	return nil
}

type HardwareDeleteRackRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Keys          []uint32               `protobuf:"varint,1,rep,packed,name=keys,proto3" json:"keys,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *HardwareDeleteRackRequest) Reset() {
	*x = HardwareDeleteRackRequest{}
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *HardwareDeleteRackRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HardwareDeleteRackRequest) ProtoMessage() {}

func (x *HardwareDeleteRackRequest) ProtoReflect() protoreflect.Message {
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HardwareDeleteRackRequest.ProtoReflect.Descriptor instead.
func (*HardwareDeleteRackRequest) Descriptor() ([]byte, []int) {
	return file_synnax_pkg_api_grpc_v1_hardware_proto_rawDescGZIP(), []int{5}
}

func (x *HardwareDeleteRackRequest) GetKeys() []uint32 {
	if x != nil {
		return x.Keys
	}
	return nil
}

type Task struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Key           uint64                 `protobuf:"varint,1,opt,name=key,proto3" json:"key,omitempty"`
	Name          string                 `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Type          string                 `protobuf:"bytes,3,opt,name=type,proto3" json:"type,omitempty"`
	Config        string                 `protobuf:"bytes,4,opt,name=config,proto3" json:"config,omitempty"`
	Internal      bool                   `protobuf:"varint,5,opt,name=internal,proto3" json:"internal,omitempty"`
	Snapshot      bool                   `protobuf:"varint,6,opt,name=snapshot,proto3" json:"snapshot,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Task) Reset() {
	*x = Task{}
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Task) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Task) ProtoMessage() {}

func (x *Task) ProtoReflect() protoreflect.Message {
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Task.ProtoReflect.Descriptor instead.
func (*Task) Descriptor() ([]byte, []int) {
	return file_synnax_pkg_api_grpc_v1_hardware_proto_rawDescGZIP(), []int{6}
}

func (x *Task) GetKey() uint64 {
	if x != nil {
		return x.Key
	}
	return 0
}

func (x *Task) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Task) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

func (x *Task) GetConfig() string {
	if x != nil {
		return x.Config
	}
	return ""
}

func (x *Task) GetInternal() bool {
	if x != nil {
		return x.Internal
	}
	return false
}

func (x *Task) GetSnapshot() bool {
	if x != nil {
		return x.Snapshot
	}
	return false
}

type HardwareCreateTaskRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Tasks         []*Task                `protobuf:"bytes,1,rep,name=tasks,proto3" json:"tasks,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *HardwareCreateTaskRequest) Reset() {
	*x = HardwareCreateTaskRequest{}
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *HardwareCreateTaskRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HardwareCreateTaskRequest) ProtoMessage() {}

func (x *HardwareCreateTaskRequest) ProtoReflect() protoreflect.Message {
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HardwareCreateTaskRequest.ProtoReflect.Descriptor instead.
func (*HardwareCreateTaskRequest) Descriptor() ([]byte, []int) {
	return file_synnax_pkg_api_grpc_v1_hardware_proto_rawDescGZIP(), []int{7}
}

func (x *HardwareCreateTaskRequest) GetTasks() []*Task {
	if x != nil {
		return x.Tasks
	}
	return nil
}

type HardwareCreateTaskResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Tasks         []*Task                `protobuf:"bytes,1,rep,name=tasks,proto3" json:"tasks,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *HardwareCreateTaskResponse) Reset() {
	*x = HardwareCreateTaskResponse{}
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[8]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *HardwareCreateTaskResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HardwareCreateTaskResponse) ProtoMessage() {}

func (x *HardwareCreateTaskResponse) ProtoReflect() protoreflect.Message {
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[8]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HardwareCreateTaskResponse.ProtoReflect.Descriptor instead.
func (*HardwareCreateTaskResponse) Descriptor() ([]byte, []int) {
	return file_synnax_pkg_api_grpc_v1_hardware_proto_rawDescGZIP(), []int{8}
}

func (x *HardwareCreateTaskResponse) GetTasks() []*Task {
	if x != nil {
		return x.Tasks
	}
	return nil
}

type HardwareRetrieveTaskRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Rack          uint32                 `protobuf:"varint,1,opt,name=rack,proto3" json:"rack,omitempty"`
	Keys          []uint64               `protobuf:"varint,2,rep,packed,name=keys,proto3" json:"keys,omitempty"`
	Names         []string               `protobuf:"bytes,3,rep,name=names,proto3" json:"names,omitempty"`
	Types         []string               `protobuf:"bytes,4,rep,name=types,proto3" json:"types,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *HardwareRetrieveTaskRequest) Reset() {
	*x = HardwareRetrieveTaskRequest{}
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[9]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *HardwareRetrieveTaskRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HardwareRetrieveTaskRequest) ProtoMessage() {}

func (x *HardwareRetrieveTaskRequest) ProtoReflect() protoreflect.Message {
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[9]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HardwareRetrieveTaskRequest.ProtoReflect.Descriptor instead.
func (*HardwareRetrieveTaskRequest) Descriptor() ([]byte, []int) {
	return file_synnax_pkg_api_grpc_v1_hardware_proto_rawDescGZIP(), []int{9}
}

func (x *HardwareRetrieveTaskRequest) GetRack() uint32 {
	if x != nil {
		return x.Rack
	}
	return 0
}

func (x *HardwareRetrieveTaskRequest) GetKeys() []uint64 {
	if x != nil {
		return x.Keys
	}
	return nil
}

func (x *HardwareRetrieveTaskRequest) GetNames() []string {
	if x != nil {
		return x.Names
	}
	return nil
}

func (x *HardwareRetrieveTaskRequest) GetTypes() []string {
	if x != nil {
		return x.Types
	}
	return nil
}

type HardwareRetrieveTaskResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Tasks         []*Task                `protobuf:"bytes,1,rep,name=tasks,proto3" json:"tasks,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *HardwareRetrieveTaskResponse) Reset() {
	*x = HardwareRetrieveTaskResponse{}
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[10]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *HardwareRetrieveTaskResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HardwareRetrieveTaskResponse) ProtoMessage() {}

func (x *HardwareRetrieveTaskResponse) ProtoReflect() protoreflect.Message {
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[10]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HardwareRetrieveTaskResponse.ProtoReflect.Descriptor instead.
func (*HardwareRetrieveTaskResponse) Descriptor() ([]byte, []int) {
	return file_synnax_pkg_api_grpc_v1_hardware_proto_rawDescGZIP(), []int{10}
}

func (x *HardwareRetrieveTaskResponse) GetTasks() []*Task {
	if x != nil {
		return x.Tasks
	}
	return nil
}

type HardwareDeleteTaskRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Keys          []uint64               `protobuf:"varint,1,rep,packed,name=keys,proto3" json:"keys,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *HardwareDeleteTaskRequest) Reset() {
	*x = HardwareDeleteTaskRequest{}
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[11]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *HardwareDeleteTaskRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HardwareDeleteTaskRequest) ProtoMessage() {}

func (x *HardwareDeleteTaskRequest) ProtoReflect() protoreflect.Message {
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[11]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HardwareDeleteTaskRequest.ProtoReflect.Descriptor instead.
func (*HardwareDeleteTaskRequest) Descriptor() ([]byte, []int) {
	return file_synnax_pkg_api_grpc_v1_hardware_proto_rawDescGZIP(), []int{11}
}

func (x *HardwareDeleteTaskRequest) GetKeys() []uint64 {
	if x != nil {
		return x.Keys
	}
	return nil
}

type Device struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Key           string                 `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Name          string                 `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Rack          uint32                 `protobuf:"varint,3,opt,name=rack,proto3" json:"rack,omitempty"`
	Location      string                 `protobuf:"bytes,4,opt,name=location,proto3" json:"location,omitempty"`
	Make          string                 `protobuf:"bytes,6,opt,name=make,proto3" json:"make,omitempty"`
	Model         string                 `protobuf:"bytes,7,opt,name=model,proto3" json:"model,omitempty"`
	Properties    string                 `protobuf:"bytes,8,opt,name=properties,proto3" json:"properties,omitempty"`
	Configured    bool                   `protobuf:"varint,9,opt,name=configured,proto3" json:"configured,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Device) Reset() {
	*x = Device{}
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[12]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Device) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Device) ProtoMessage() {}

func (x *Device) ProtoReflect() protoreflect.Message {
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[12]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Device.ProtoReflect.Descriptor instead.
func (*Device) Descriptor() ([]byte, []int) {
	return file_synnax_pkg_api_grpc_v1_hardware_proto_rawDescGZIP(), []int{12}
}

func (x *Device) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

func (x *Device) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Device) GetRack() uint32 {
	if x != nil {
		return x.Rack
	}
	return 0
}

func (x *Device) GetLocation() string {
	if x != nil {
		return x.Location
	}
	return ""
}

func (x *Device) GetMake() string {
	if x != nil {
		return x.Make
	}
	return ""
}

func (x *Device) GetModel() string {
	if x != nil {
		return x.Model
	}
	return ""
}

func (x *Device) GetProperties() string {
	if x != nil {
		return x.Properties
	}
	return ""
}

func (x *Device) GetConfigured() bool {
	if x != nil {
		return x.Configured
	}
	return false
}

type HardwareCreateDeviceRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Devices       []*Device              `protobuf:"bytes,1,rep,name=devices,proto3" json:"devices,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *HardwareCreateDeviceRequest) Reset() {
	*x = HardwareCreateDeviceRequest{}
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[13]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *HardwareCreateDeviceRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HardwareCreateDeviceRequest) ProtoMessage() {}

func (x *HardwareCreateDeviceRequest) ProtoReflect() protoreflect.Message {
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[13]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HardwareCreateDeviceRequest.ProtoReflect.Descriptor instead.
func (*HardwareCreateDeviceRequest) Descriptor() ([]byte, []int) {
	return file_synnax_pkg_api_grpc_v1_hardware_proto_rawDescGZIP(), []int{13}
}

func (x *HardwareCreateDeviceRequest) GetDevices() []*Device {
	if x != nil {
		return x.Devices
	}
	return nil
}

type HardwareCreateDeviceResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Devices       []*Device              `protobuf:"bytes,1,rep,name=devices,proto3" json:"devices,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *HardwareCreateDeviceResponse) Reset() {
	*x = HardwareCreateDeviceResponse{}
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[14]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *HardwareCreateDeviceResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HardwareCreateDeviceResponse) ProtoMessage() {}

func (x *HardwareCreateDeviceResponse) ProtoReflect() protoreflect.Message {
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[14]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HardwareCreateDeviceResponse.ProtoReflect.Descriptor instead.
func (*HardwareCreateDeviceResponse) Descriptor() ([]byte, []int) {
	return file_synnax_pkg_api_grpc_v1_hardware_proto_rawDescGZIP(), []int{14}
}

func (x *HardwareCreateDeviceResponse) GetDevices() []*Device {
	if x != nil {
		return x.Devices
	}
	return nil
}

type HardwareRetrieveDeviceRequest struct {
	state          protoimpl.MessageState `protogen:"open.v1"`
	Keys           []string               `protobuf:"bytes,1,rep,name=keys,proto3" json:"keys,omitempty"`
	IgnoreNotFound bool                   `protobuf:"varint,2,opt,name=ignore_not_found,json=ignoreNotFound,proto3" json:"ignore_not_found,omitempty"`
	unknownFields  protoimpl.UnknownFields
	sizeCache      protoimpl.SizeCache
}

func (x *HardwareRetrieveDeviceRequest) Reset() {
	*x = HardwareRetrieveDeviceRequest{}
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[15]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *HardwareRetrieveDeviceRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HardwareRetrieveDeviceRequest) ProtoMessage() {}

func (x *HardwareRetrieveDeviceRequest) ProtoReflect() protoreflect.Message {
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[15]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HardwareRetrieveDeviceRequest.ProtoReflect.Descriptor instead.
func (*HardwareRetrieveDeviceRequest) Descriptor() ([]byte, []int) {
	return file_synnax_pkg_api_grpc_v1_hardware_proto_rawDescGZIP(), []int{15}
}

func (x *HardwareRetrieveDeviceRequest) GetKeys() []string {
	if x != nil {
		return x.Keys
	}
	return nil
}

func (x *HardwareRetrieveDeviceRequest) GetIgnoreNotFound() bool {
	if x != nil {
		return x.IgnoreNotFound
	}
	return false
}

type HardwareRetrieveDeviceResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Devices       []*Device              `protobuf:"bytes,1,rep,name=devices,proto3" json:"devices,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *HardwareRetrieveDeviceResponse) Reset() {
	*x = HardwareRetrieveDeviceResponse{}
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[16]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *HardwareRetrieveDeviceResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HardwareRetrieveDeviceResponse) ProtoMessage() {}

func (x *HardwareRetrieveDeviceResponse) ProtoReflect() protoreflect.Message {
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[16]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HardwareRetrieveDeviceResponse.ProtoReflect.Descriptor instead.
func (*HardwareRetrieveDeviceResponse) Descriptor() ([]byte, []int) {
	return file_synnax_pkg_api_grpc_v1_hardware_proto_rawDescGZIP(), []int{16}
}

func (x *HardwareRetrieveDeviceResponse) GetDevices() []*Device {
	if x != nil {
		return x.Devices
	}
	return nil
}

type HardwareDeleteDeviceRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Keys          []string               `protobuf:"bytes,1,rep,name=keys,proto3" json:"keys,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *HardwareDeleteDeviceRequest) Reset() {
	*x = HardwareDeleteDeviceRequest{}
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[17]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *HardwareDeleteDeviceRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HardwareDeleteDeviceRequest) ProtoMessage() {}

func (x *HardwareDeleteDeviceRequest) ProtoReflect() protoreflect.Message {
	mi := &file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes[17]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HardwareDeleteDeviceRequest.ProtoReflect.Descriptor instead.
func (*HardwareDeleteDeviceRequest) Descriptor() ([]byte, []int) {
	return file_synnax_pkg_api_grpc_v1_hardware_proto_rawDescGZIP(), []int{17}
}

func (x *HardwareDeleteDeviceRequest) GetKeys() []string {
	if x != nil {
		return x.Keys
	}
	return nil
}

var File_synnax_pkg_api_grpc_v1_hardware_proto protoreflect.FileDescriptor

const file_synnax_pkg_api_grpc_v1_hardware_proto_rawDesc = "" +
	"\n" +
	"%synnax/pkg/api/grpc/v1/hardware.proto\x12\x06api.v1\x1a\x1bgoogle/protobuf/empty.proto\",\n" +
	"\x04Rack\x12\x10\n" +
	"\x03key\x18\x01 \x01(\rR\x03key\x12\x12\n" +
	"\x04name\x18\x02 \x01(\tR\x04name\"?\n" +
	"\x19HardwareCreateRackRequest\x12\"\n" +
	"\x05racks\x18\x01 \x03(\v2\f.api.v1.RackR\x05racks\"@\n" +
	"\x1aHardwareCreateRackResponse\x12\"\n" +
	"\x05racks\x18\x01 \x03(\v2\f.api.v1.RackR\x05racks\"G\n" +
	"\x1bHardwareRetrieveRackRequest\x12\x12\n" +
	"\x04keys\x18\x01 \x03(\rR\x04keys\x12\x14\n" +
	"\x05names\x18\x02 \x03(\tR\x05names\"B\n" +
	"\x1cHardwareRetrieveRackResponse\x12\"\n" +
	"\x05racks\x18\x01 \x03(\v2\f.api.v1.RackR\x05racks\"/\n" +
	"\x19HardwareDeleteRackRequest\x12\x12\n" +
	"\x04keys\x18\x01 \x03(\rR\x04keys\"\x90\x01\n" +
	"\x04Task\x12\x10\n" +
	"\x03key\x18\x01 \x01(\x04R\x03key\x12\x12\n" +
	"\x04name\x18\x02 \x01(\tR\x04name\x12\x12\n" +
	"\x04type\x18\x03 \x01(\tR\x04type\x12\x16\n" +
	"\x06config\x18\x04 \x01(\tR\x06config\x12\x1a\n" +
	"\binternal\x18\x05 \x01(\bR\binternal\x12\x1a\n" +
	"\bsnapshot\x18\x06 \x01(\bR\bsnapshot\"?\n" +
	"\x19HardwareCreateTaskRequest\x12\"\n" +
	"\x05tasks\x18\x01 \x03(\v2\f.api.v1.TaskR\x05tasks\"@\n" +
	"\x1aHardwareCreateTaskResponse\x12\"\n" +
	"\x05tasks\x18\x01 \x03(\v2\f.api.v1.TaskR\x05tasks\"q\n" +
	"\x1bHardwareRetrieveTaskRequest\x12\x12\n" +
	"\x04rack\x18\x01 \x01(\rR\x04rack\x12\x12\n" +
	"\x04keys\x18\x02 \x03(\x04R\x04keys\x12\x14\n" +
	"\x05names\x18\x03 \x03(\tR\x05names\x12\x14\n" +
	"\x05types\x18\x04 \x03(\tR\x05types\"B\n" +
	"\x1cHardwareRetrieveTaskResponse\x12\"\n" +
	"\x05tasks\x18\x01 \x03(\v2\f.api.v1.TaskR\x05tasks\"/\n" +
	"\x19HardwareDeleteTaskRequest\x12\x12\n" +
	"\x04keys\x18\x01 \x03(\x04R\x04keys\"\xc8\x01\n" +
	"\x06Device\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x12\n" +
	"\x04name\x18\x02 \x01(\tR\x04name\x12\x12\n" +
	"\x04rack\x18\x03 \x01(\rR\x04rack\x12\x1a\n" +
	"\blocation\x18\x04 \x01(\tR\blocation\x12\x12\n" +
	"\x04make\x18\x06 \x01(\tR\x04make\x12\x14\n" +
	"\x05model\x18\a \x01(\tR\x05model\x12\x1e\n" +
	"\n" +
	"properties\x18\b \x01(\tR\n" +
	"properties\x12\x1e\n" +
	"\n" +
	"configured\x18\t \x01(\bR\n" +
	"configured\"G\n" +
	"\x1bHardwareCreateDeviceRequest\x12(\n" +
	"\adevices\x18\x01 \x03(\v2\x0e.api.v1.DeviceR\adevices\"H\n" +
	"\x1cHardwareCreateDeviceResponse\x12(\n" +
	"\adevices\x18\x01 \x03(\v2\x0e.api.v1.DeviceR\adevices\"]\n" +
	"\x1dHardwareRetrieveDeviceRequest\x12\x12\n" +
	"\x04keys\x18\x01 \x03(\tR\x04keys\x12(\n" +
	"\x10ignore_not_found\x18\x02 \x01(\bR\x0eignoreNotFound\"J\n" +
	"\x1eHardwareRetrieveDeviceResponse\x12(\n" +
	"\adevices\x18\x01 \x03(\v2\x0e.api.v1.DeviceR\adevices\"1\n" +
	"\x1bHardwareDeleteDeviceRequest\x12\x12\n" +
	"\x04keys\x18\x01 \x03(\tR\x04keys2j\n" +
	"\x19HardwareCreateTaskService\x12M\n" +
	"\x04Exec\x12!.api.v1.HardwareCreateTaskRequest\x1a\".api.v1.HardwareCreateTaskResponse2p\n" +
	"\x1bHardwareRetrieveTaskService\x12Q\n" +
	"\x04Exec\x12#.api.v1.HardwareRetrieveTaskRequest\x1a$.api.v1.HardwareRetrieveTaskResponse2^\n" +
	"\x19HardwareDeleteTaskService\x12A\n" +
	"\x04Exec\x12!.api.v1.HardwareDeleteTaskRequest\x1a\x16.google.protobuf.Empty2j\n" +
	"\x19HardwareCreateRackService\x12M\n" +
	"\x04Exec\x12!.api.v1.HardwareCreateRackRequest\x1a\".api.v1.HardwareCreateRackResponse2p\n" +
	"\x1bHardwareRetrieveRackService\x12Q\n" +
	"\x04Exec\x12#.api.v1.HardwareRetrieveRackRequest\x1a$.api.v1.HardwareRetrieveRackResponse2^\n" +
	"\x19HardwareDeleteRackService\x12A\n" +
	"\x04Exec\x12!.api.v1.HardwareDeleteRackRequest\x1a\x16.google.protobuf.Empty2p\n" +
	"\x1bHardwareCreateDeviceService\x12Q\n" +
	"\x04Exec\x12#.api.v1.HardwareCreateDeviceRequest\x1a$.api.v1.HardwareCreateDeviceResponse2v\n" +
	"\x1dHardwareRetrieveDeviceService\x12U\n" +
	"\x04Exec\x12%.api.v1.HardwareRetrieveDeviceRequest\x1a&.api.v1.HardwareRetrieveDeviceResponse2b\n" +
	"\x1bHardwareDeleteDeviceService\x12C\n" +
	"\x04Exec\x12#.api.v1.HardwareDeleteDeviceRequest\x1a\x16.google.protobuf.EmptyB\x82\x01\n" +
	"\n" +
	"com.api.v1B\rHardwareProtoP\x01Z,github.com/synnaxlabs/synnax/pkg/api/grpc/v1\xa2\x02\x03AXX\xaa\x02\x06Api.V1\xca\x02\x06Api\\V1\xe2\x02\x12Api\\V1\\GPBMetadata\xea\x02\aApi::V1b\x06proto3"

var (
	file_synnax_pkg_api_grpc_v1_hardware_proto_rawDescOnce sync.Once
	file_synnax_pkg_api_grpc_v1_hardware_proto_rawDescData []byte
)

func file_synnax_pkg_api_grpc_v1_hardware_proto_rawDescGZIP() []byte {
	file_synnax_pkg_api_grpc_v1_hardware_proto_rawDescOnce.Do(func() {
		file_synnax_pkg_api_grpc_v1_hardware_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_synnax_pkg_api_grpc_v1_hardware_proto_rawDesc), len(file_synnax_pkg_api_grpc_v1_hardware_proto_rawDesc)))
	})
	return file_synnax_pkg_api_grpc_v1_hardware_proto_rawDescData
}

var file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes = make([]protoimpl.MessageInfo, 18)
var file_synnax_pkg_api_grpc_v1_hardware_proto_goTypes = []any{
	(*Rack)(nil),                           // 0: api.v1.Rack
	(*HardwareCreateRackRequest)(nil),      // 1: api.v1.HardwareCreateRackRequest
	(*HardwareCreateRackResponse)(nil),     // 2: api.v1.HardwareCreateRackResponse
	(*HardwareRetrieveRackRequest)(nil),    // 3: api.v1.HardwareRetrieveRackRequest
	(*HardwareRetrieveRackResponse)(nil),   // 4: api.v1.HardwareRetrieveRackResponse
	(*HardwareDeleteRackRequest)(nil),      // 5: api.v1.HardwareDeleteRackRequest
	(*Task)(nil),                           // 6: api.v1.Task
	(*HardwareCreateTaskRequest)(nil),      // 7: api.v1.HardwareCreateTaskRequest
	(*HardwareCreateTaskResponse)(nil),     // 8: api.v1.HardwareCreateTaskResponse
	(*HardwareRetrieveTaskRequest)(nil),    // 9: api.v1.HardwareRetrieveTaskRequest
	(*HardwareRetrieveTaskResponse)(nil),   // 10: api.v1.HardwareRetrieveTaskResponse
	(*HardwareDeleteTaskRequest)(nil),      // 11: api.v1.HardwareDeleteTaskRequest
	(*Device)(nil),                         // 12: api.v1.Device
	(*HardwareCreateDeviceRequest)(nil),    // 13: api.v1.HardwareCreateDeviceRequest
	(*HardwareCreateDeviceResponse)(nil),   // 14: api.v1.HardwareCreateDeviceResponse
	(*HardwareRetrieveDeviceRequest)(nil),  // 15: api.v1.HardwareRetrieveDeviceRequest
	(*HardwareRetrieveDeviceResponse)(nil), // 16: api.v1.HardwareRetrieveDeviceResponse
	(*HardwareDeleteDeviceRequest)(nil),    // 17: api.v1.HardwareDeleteDeviceRequest
	(*emptypb.Empty)(nil),                  // 18: google.protobuf.Empty
}
var file_synnax_pkg_api_grpc_v1_hardware_proto_depIdxs = []int32{
	0,  // 0: api.v1.HardwareCreateRackRequest.racks:type_name -> api.v1.Rack
	0,  // 1: api.v1.HardwareCreateRackResponse.racks:type_name -> api.v1.Rack
	0,  // 2: api.v1.HardwareRetrieveRackResponse.racks:type_name -> api.v1.Rack
	6,  // 3: api.v1.HardwareCreateTaskRequest.tasks:type_name -> api.v1.Task
	6,  // 4: api.v1.HardwareCreateTaskResponse.tasks:type_name -> api.v1.Task
	6,  // 5: api.v1.HardwareRetrieveTaskResponse.tasks:type_name -> api.v1.Task
	12, // 6: api.v1.HardwareCreateDeviceRequest.devices:type_name -> api.v1.Device
	12, // 7: api.v1.HardwareCreateDeviceResponse.devices:type_name -> api.v1.Device
	12, // 8: api.v1.HardwareRetrieveDeviceResponse.devices:type_name -> api.v1.Device
	7,  // 9: api.v1.HardwareCreateTaskService.Exec:input_type -> api.v1.HardwareCreateTaskRequest
	9,  // 10: api.v1.HardwareRetrieveTaskService.Exec:input_type -> api.v1.HardwareRetrieveTaskRequest
	11, // 11: api.v1.HardwareDeleteTaskService.Exec:input_type -> api.v1.HardwareDeleteTaskRequest
	1,  // 12: api.v1.HardwareCreateRackService.Exec:input_type -> api.v1.HardwareCreateRackRequest
	3,  // 13: api.v1.HardwareRetrieveRackService.Exec:input_type -> api.v1.HardwareRetrieveRackRequest
	5,  // 14: api.v1.HardwareDeleteRackService.Exec:input_type -> api.v1.HardwareDeleteRackRequest
	13, // 15: api.v1.HardwareCreateDeviceService.Exec:input_type -> api.v1.HardwareCreateDeviceRequest
	15, // 16: api.v1.HardwareRetrieveDeviceService.Exec:input_type -> api.v1.HardwareRetrieveDeviceRequest
	17, // 17: api.v1.HardwareDeleteDeviceService.Exec:input_type -> api.v1.HardwareDeleteDeviceRequest
	8,  // 18: api.v1.HardwareCreateTaskService.Exec:output_type -> api.v1.HardwareCreateTaskResponse
	10, // 19: api.v1.HardwareRetrieveTaskService.Exec:output_type -> api.v1.HardwareRetrieveTaskResponse
	18, // 20: api.v1.HardwareDeleteTaskService.Exec:output_type -> google.protobuf.Empty
	2,  // 21: api.v1.HardwareCreateRackService.Exec:output_type -> api.v1.HardwareCreateRackResponse
	4,  // 22: api.v1.HardwareRetrieveRackService.Exec:output_type -> api.v1.HardwareRetrieveRackResponse
	18, // 23: api.v1.HardwareDeleteRackService.Exec:output_type -> google.protobuf.Empty
	14, // 24: api.v1.HardwareCreateDeviceService.Exec:output_type -> api.v1.HardwareCreateDeviceResponse
	16, // 25: api.v1.HardwareRetrieveDeviceService.Exec:output_type -> api.v1.HardwareRetrieveDeviceResponse
	18, // 26: api.v1.HardwareDeleteDeviceService.Exec:output_type -> google.protobuf.Empty
	18, // [18:27] is the sub-list for method output_type
	9,  // [9:18] is the sub-list for method input_type
	9,  // [9:9] is the sub-list for extension type_name
	9,  // [9:9] is the sub-list for extension extendee
	0,  // [0:9] is the sub-list for field type_name
}

func init() { file_synnax_pkg_api_grpc_v1_hardware_proto_init() }
func file_synnax_pkg_api_grpc_v1_hardware_proto_init() {
	if File_synnax_pkg_api_grpc_v1_hardware_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_synnax_pkg_api_grpc_v1_hardware_proto_rawDesc), len(file_synnax_pkg_api_grpc_v1_hardware_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   18,
			NumExtensions: 0,
			NumServices:   9,
		},
		GoTypes:           file_synnax_pkg_api_grpc_v1_hardware_proto_goTypes,
		DependencyIndexes: file_synnax_pkg_api_grpc_v1_hardware_proto_depIdxs,
		MessageInfos:      file_synnax_pkg_api_grpc_v1_hardware_proto_msgTypes,
	}.Build()
	File_synnax_pkg_api_grpc_v1_hardware_proto = out.File
	file_synnax_pkg_api_grpc_v1_hardware_proto_goTypes = nil
	file_synnax_pkg_api_grpc_v1_hardware_proto_depIdxs = nil
}
