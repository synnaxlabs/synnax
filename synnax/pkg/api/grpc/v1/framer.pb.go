// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.2
// 	protoc        (unknown)
// source: synnax/pkg/api/grpc/v1/framer.proto

package v1

import (
	control "github.com/synnaxlabs/x/control"
	errors "github.com/synnaxlabs/x/errors"
	telem "github.com/synnaxlabs/x/telem"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Frame struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Keys   []uint32          `protobuf:"varint,1,rep,packed,name=keys,proto3" json:"keys,omitempty"`
	Series []*telem.PBSeries `protobuf:"bytes,2,rep,name=series,proto3" json:"series,omitempty"`
}

func (x *Frame) Reset() {
	*x = Frame{}
	if protoimpl.UnsafeEnabled {
		mi := &file_synnax_pkg_api_grpc_v1_framer_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Frame) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Frame) ProtoMessage() {}

func (x *Frame) ProtoReflect() protoreflect.Message {
	mi := &file_synnax_pkg_api_grpc_v1_framer_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Frame.ProtoReflect.Descriptor instead.
func (*Frame) Descriptor() ([]byte, []int) {
	return file_synnax_pkg_api_grpc_v1_framer_proto_rawDescGZIP(), []int{0}
}

func (x *Frame) GetKeys() []uint32 {
	if x != nil {
		return x.Keys
	}
	return nil
}

func (x *Frame) GetSeries() []*telem.PBSeries {
	if x != nil {
		return x.Series
	}
	return nil
}

type FrameIteratorRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Command int32              `protobuf:"varint,1,opt,name=command,proto3" json:"command,omitempty"`
	Span    int64              `protobuf:"varint,2,opt,name=span,proto3" json:"span,omitempty"`
	Range   *telem.PBTimeRange `protobuf:"bytes,3,opt,name=range,proto3" json:"range,omitempty"`
	Stamp   int64              `protobuf:"varint,4,opt,name=stamp,proto3" json:"stamp,omitempty"`
	Keys    []uint32           `protobuf:"varint,5,rep,packed,name=keys,proto3" json:"keys,omitempty"`
}

func (x *FrameIteratorRequest) Reset() {
	*x = FrameIteratorRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_synnax_pkg_api_grpc_v1_framer_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FrameIteratorRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FrameIteratorRequest) ProtoMessage() {}

func (x *FrameIteratorRequest) ProtoReflect() protoreflect.Message {
	mi := &file_synnax_pkg_api_grpc_v1_framer_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FrameIteratorRequest.ProtoReflect.Descriptor instead.
func (*FrameIteratorRequest) Descriptor() ([]byte, []int) {
	return file_synnax_pkg_api_grpc_v1_framer_proto_rawDescGZIP(), []int{1}
}

func (x *FrameIteratorRequest) GetCommand() int32 {
	if x != nil {
		return x.Command
	}
	return 0
}

func (x *FrameIteratorRequest) GetSpan() int64 {
	if x != nil {
		return x.Span
	}
	return 0
}

func (x *FrameIteratorRequest) GetRange() *telem.PBTimeRange {
	if x != nil {
		return x.Range
	}
	return nil
}

func (x *FrameIteratorRequest) GetStamp() int64 {
	if x != nil {
		return x.Stamp
	}
	return 0
}

func (x *FrameIteratorRequest) GetKeys() []uint32 {
	if x != nil {
		return x.Keys
	}
	return nil
}

type FrameIteratorResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Variant int32             `protobuf:"varint,1,opt,name=variant,proto3" json:"variant,omitempty"`
	Command int32             `protobuf:"varint,2,opt,name=command,proto3" json:"command,omitempty"`
	Frame   *Frame            `protobuf:"bytes,3,opt,name=frame,proto3" json:"frame,omitempty"`
	NodeKey int32             `protobuf:"varint,43,opt,name=node_key,json=nodeKey,proto3" json:"node_key,omitempty"`
	Ack     bool              `protobuf:"varint,5,opt,name=ack,proto3" json:"ack,omitempty"`
	SeqNum  int32             `protobuf:"varint,6,opt,name=seq_num,json=seqNum,proto3" json:"seq_num,omitempty"`
	Error   *errors.PBPayload `protobuf:"bytes,7,opt,name=error,proto3" json:"error,omitempty"`
}

func (x *FrameIteratorResponse) Reset() {
	*x = FrameIteratorResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_synnax_pkg_api_grpc_v1_framer_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FrameIteratorResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FrameIteratorResponse) ProtoMessage() {}

func (x *FrameIteratorResponse) ProtoReflect() protoreflect.Message {
	mi := &file_synnax_pkg_api_grpc_v1_framer_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FrameIteratorResponse.ProtoReflect.Descriptor instead.
func (*FrameIteratorResponse) Descriptor() ([]byte, []int) {
	return file_synnax_pkg_api_grpc_v1_framer_proto_rawDescGZIP(), []int{2}
}

func (x *FrameIteratorResponse) GetVariant() int32 {
	if x != nil {
		return x.Variant
	}
	return 0
}

func (x *FrameIteratorResponse) GetCommand() int32 {
	if x != nil {
		return x.Command
	}
	return 0
}

func (x *FrameIteratorResponse) GetFrame() *Frame {
	if x != nil {
		return x.Frame
	}
	return nil
}

func (x *FrameIteratorResponse) GetNodeKey() int32 {
	if x != nil {
		return x.NodeKey
	}
	return 0
}

func (x *FrameIteratorResponse) GetAck() bool {
	if x != nil {
		return x.Ack
	}
	return false
}

func (x *FrameIteratorResponse) GetSeqNum() int32 {
	if x != nil {
		return x.SeqNum
	}
	return 0
}

func (x *FrameIteratorResponse) GetError() *errors.PBPayload {
	if x != nil {
		return x.Error
	}
	return nil
}

type FrameWriterConfig struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Keys                     []uint32                `protobuf:"varint,1,rep,packed,name=keys,proto3" json:"keys,omitempty"`
	Authorities              []uint32                `protobuf:"varint,2,rep,packed,name=authorities,proto3" json:"authorities,omitempty"`
	Start                    int64                   `protobuf:"varint,3,opt,name=start,proto3" json:"start,omitempty"`
	ControlSubject           *control.ControlSubject `protobuf:"bytes,4,opt,name=control_subject,json=controlSubject,proto3" json:"control_subject,omitempty"`
	Mode                     int32                   `protobuf:"varint,5,opt,name=mode,proto3" json:"mode,omitempty"`
	EnableAutoCommit         bool                    `protobuf:"varint,6,opt,name=enable_auto_commit,json=enableAutoCommit,proto3" json:"enable_auto_commit,omitempty"`
	AutoIndexPersistInterval int64                   `protobuf:"varint,7,opt,name=auto_index_persist_interval,json=autoIndexPersistInterval,proto3" json:"auto_index_persist_interval,omitempty"`
	ErrOnUnauthorized        bool                    `protobuf:"varint,8,opt,name=err_on_unauthorized,json=errOnUnauthorized,proto3" json:"err_on_unauthorized,omitempty"`
}

func (x *FrameWriterConfig) Reset() {
	*x = FrameWriterConfig{}
	if protoimpl.UnsafeEnabled {
		mi := &file_synnax_pkg_api_grpc_v1_framer_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FrameWriterConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FrameWriterConfig) ProtoMessage() {}

func (x *FrameWriterConfig) ProtoReflect() protoreflect.Message {
	mi := &file_synnax_pkg_api_grpc_v1_framer_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FrameWriterConfig.ProtoReflect.Descriptor instead.
func (*FrameWriterConfig) Descriptor() ([]byte, []int) {
	return file_synnax_pkg_api_grpc_v1_framer_proto_rawDescGZIP(), []int{3}
}

func (x *FrameWriterConfig) GetKeys() []uint32 {
	if x != nil {
		return x.Keys
	}
	return nil
}

func (x *FrameWriterConfig) GetAuthorities() []uint32 {
	if x != nil {
		return x.Authorities
	}
	return nil
}

func (x *FrameWriterConfig) GetStart() int64 {
	if x != nil {
		return x.Start
	}
	return 0
}

func (x *FrameWriterConfig) GetControlSubject() *control.ControlSubject {
	if x != nil {
		return x.ControlSubject
	}
	return nil
}

func (x *FrameWriterConfig) GetMode() int32 {
	if x != nil {
		return x.Mode
	}
	return 0
}

func (x *FrameWriterConfig) GetEnableAutoCommit() bool {
	if x != nil {
		return x.EnableAutoCommit
	}
	return false
}

func (x *FrameWriterConfig) GetAutoIndexPersistInterval() int64 {
	if x != nil {
		return x.AutoIndexPersistInterval
	}
	return 0
}

func (x *FrameWriterConfig) GetErrOnUnauthorized() bool {
	if x != nil {
		return x.ErrOnUnauthorized
	}
	return false
}

type FrameWriterRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Command int32              `protobuf:"varint,1,opt,name=command,proto3" json:"command,omitempty"`
	Config  *FrameWriterConfig `protobuf:"bytes,2,opt,name=config,proto3" json:"config,omitempty"`
	Frame   *Frame             `protobuf:"bytes,3,opt,name=frame,proto3" json:"frame,omitempty"`
}

func (x *FrameWriterRequest) Reset() {
	*x = FrameWriterRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_synnax_pkg_api_grpc_v1_framer_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FrameWriterRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FrameWriterRequest) ProtoMessage() {}

func (x *FrameWriterRequest) ProtoReflect() protoreflect.Message {
	mi := &file_synnax_pkg_api_grpc_v1_framer_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FrameWriterRequest.ProtoReflect.Descriptor instead.
func (*FrameWriterRequest) Descriptor() ([]byte, []int) {
	return file_synnax_pkg_api_grpc_v1_framer_proto_rawDescGZIP(), []int{4}
}

func (x *FrameWriterRequest) GetCommand() int32 {
	if x != nil {
		return x.Command
	}
	return 0
}

func (x *FrameWriterRequest) GetConfig() *FrameWriterConfig {
	if x != nil {
		return x.Config
	}
	return nil
}

func (x *FrameWriterRequest) GetFrame() *Frame {
	if x != nil {
		return x.Frame
	}
	return nil
}

type FrameWriterResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Command int32             `protobuf:"varint,1,opt,name=command,proto3" json:"command,omitempty"`
	Ack     bool              `protobuf:"varint,2,opt,name=ack,proto3" json:"ack,omitempty"`
	NodeKey int32             `protobuf:"varint,3,opt,name=node_key,json=nodeKey,proto3" json:"node_key,omitempty"`
	Counter int32             `protobuf:"varint,4,opt,name=counter,proto3" json:"counter,omitempty"`
	Error   *errors.PBPayload `protobuf:"bytes,5,opt,name=error,proto3" json:"error,omitempty"`
	End     int64             `protobuf:"varint,6,opt,name=end,proto3" json:"end,omitempty"`
}

func (x *FrameWriterResponse) Reset() {
	*x = FrameWriterResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_synnax_pkg_api_grpc_v1_framer_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FrameWriterResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FrameWriterResponse) ProtoMessage() {}

func (x *FrameWriterResponse) ProtoReflect() protoreflect.Message {
	mi := &file_synnax_pkg_api_grpc_v1_framer_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FrameWriterResponse.ProtoReflect.Descriptor instead.
func (*FrameWriterResponse) Descriptor() ([]byte, []int) {
	return file_synnax_pkg_api_grpc_v1_framer_proto_rawDescGZIP(), []int{5}
}

func (x *FrameWriterResponse) GetCommand() int32 {
	if x != nil {
		return x.Command
	}
	return 0
}

func (x *FrameWriterResponse) GetAck() bool {
	if x != nil {
		return x.Ack
	}
	return false
}

func (x *FrameWriterResponse) GetNodeKey() int32 {
	if x != nil {
		return x.NodeKey
	}
	return 0
}

func (x *FrameWriterResponse) GetCounter() int32 {
	if x != nil {
		return x.Counter
	}
	return 0
}

func (x *FrameWriterResponse) GetError() *errors.PBPayload {
	if x != nil {
		return x.Error
	}
	return nil
}

func (x *FrameWriterResponse) GetEnd() int64 {
	if x != nil {
		return x.End
	}
	return 0
}

type FrameStreamerRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Start int64    `protobuf:"varint,1,opt,name=start,proto3" json:"start,omitempty"`
	Keys  []uint32 `protobuf:"varint,2,rep,packed,name=keys,proto3" json:"keys,omitempty"`
}

func (x *FrameStreamerRequest) Reset() {
	*x = FrameStreamerRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_synnax_pkg_api_grpc_v1_framer_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FrameStreamerRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FrameStreamerRequest) ProtoMessage() {}

func (x *FrameStreamerRequest) ProtoReflect() protoreflect.Message {
	mi := &file_synnax_pkg_api_grpc_v1_framer_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FrameStreamerRequest.ProtoReflect.Descriptor instead.
func (*FrameStreamerRequest) Descriptor() ([]byte, []int) {
	return file_synnax_pkg_api_grpc_v1_framer_proto_rawDescGZIP(), []int{6}
}

func (x *FrameStreamerRequest) GetStart() int64 {
	if x != nil {
		return x.Start
	}
	return 0
}

func (x *FrameStreamerRequest) GetKeys() []uint32 {
	if x != nil {
		return x.Keys
	}
	return nil
}

type FrameStreamerResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Frame *Frame            `protobuf:"bytes,1,opt,name=frame,proto3" json:"frame,omitempty"`
	Error *errors.PBPayload `protobuf:"bytes,2,opt,name=error,proto3" json:"error,omitempty"`
}

func (x *FrameStreamerResponse) Reset() {
	*x = FrameStreamerResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_synnax_pkg_api_grpc_v1_framer_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FrameStreamerResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FrameStreamerResponse) ProtoMessage() {}

func (x *FrameStreamerResponse) ProtoReflect() protoreflect.Message {
	mi := &file_synnax_pkg_api_grpc_v1_framer_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FrameStreamerResponse.ProtoReflect.Descriptor instead.
func (*FrameStreamerResponse) Descriptor() ([]byte, []int) {
	return file_synnax_pkg_api_grpc_v1_framer_proto_rawDescGZIP(), []int{7}
}

func (x *FrameStreamerResponse) GetFrame() *Frame {
	if x != nil {
		return x.Frame
	}
	return nil
}

func (x *FrameStreamerResponse) GetError() *errors.PBPayload {
	if x != nil {
		return x.Error
	}
	return nil
}

type FrameDeleteRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Keys   []uint32           `protobuf:"varint,1,rep,packed,name=keys,proto3" json:"keys,omitempty"`
	Names  []string           `protobuf:"bytes,2,rep,name=names,proto3" json:"names,omitempty"`
	Bounds *telem.PBTimeRange `protobuf:"bytes,3,opt,name=bounds,proto3" json:"bounds,omitempty"`
}

func (x *FrameDeleteRequest) Reset() {
	*x = FrameDeleteRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_synnax_pkg_api_grpc_v1_framer_proto_msgTypes[8]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FrameDeleteRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FrameDeleteRequest) ProtoMessage() {}

func (x *FrameDeleteRequest) ProtoReflect() protoreflect.Message {
	mi := &file_synnax_pkg_api_grpc_v1_framer_proto_msgTypes[8]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FrameDeleteRequest.ProtoReflect.Descriptor instead.
func (*FrameDeleteRequest) Descriptor() ([]byte, []int) {
	return file_synnax_pkg_api_grpc_v1_framer_proto_rawDescGZIP(), []int{8}
}

func (x *FrameDeleteRequest) GetKeys() []uint32 {
	if x != nil {
		return x.Keys
	}
	return nil
}

func (x *FrameDeleteRequest) GetNames() []string {
	if x != nil {
		return x.Names
	}
	return nil
}

func (x *FrameDeleteRequest) GetBounds() *telem.PBTimeRange {
	if x != nil {
		return x.Bounds
	}
	return nil
}

var File_synnax_pkg_api_grpc_v1_framer_proto protoreflect.FileDescriptor

var file_synnax_pkg_api_grpc_v1_framer_proto_rawDesc = []byte{
	0x0a, 0x23, 0x73, 0x79, 0x6e, 0x6e, 0x61, 0x78, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x61, 0x70, 0x69,
	0x2f, 0x67, 0x72, 0x70, 0x63, 0x2f, 0x76, 0x31, 0x2f, 0x66, 0x72, 0x61, 0x6d, 0x65, 0x72, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x06, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x1a, 0x16, 0x78,
	0x2f, 0x67, 0x6f, 0x2f, 0x74, 0x65, 0x6c, 0x65, 0x6d, 0x2f, 0x74, 0x65, 0x6c, 0x65, 0x6d, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x18, 0x78, 0x2f, 0x67, 0x6f, 0x2f, 0x65, 0x72, 0x72, 0x6f,
	0x72, 0x73, 0x2f, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a,
	0x1a, 0x78, 0x2f, 0x67, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c, 0x2f, 0x63, 0x6f,
	0x6e, 0x74, 0x72, 0x6f, 0x6c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1b, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x65, 0x6d, 0x70,
	0x74, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x44, 0x0a, 0x05, 0x46, 0x72, 0x61, 0x6d,
	0x65, 0x12, 0x12, 0x0a, 0x04, 0x6b, 0x65, 0x79, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0d, 0x52,
	0x04, 0x6b, 0x65, 0x79, 0x73, 0x12, 0x27, 0x0a, 0x06, 0x73, 0x65, 0x72, 0x69, 0x65, 0x73, 0x18,
	0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x0f, 0x2e, 0x74, 0x65, 0x6c, 0x65, 0x6d, 0x2e, 0x50, 0x42,
	0x53, 0x65, 0x72, 0x69, 0x65, 0x73, 0x52, 0x06, 0x73, 0x65, 0x72, 0x69, 0x65, 0x73, 0x22, 0x98,
	0x01, 0x0a, 0x14, 0x46, 0x72, 0x61, 0x6d, 0x65, 0x49, 0x74, 0x65, 0x72, 0x61, 0x74, 0x6f, 0x72,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x6f, 0x6d, 0x6d, 0x61,
	0x6e, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x07, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e,
	0x64, 0x12, 0x12, 0x0a, 0x04, 0x73, 0x70, 0x61, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52,
	0x04, 0x73, 0x70, 0x61, 0x6e, 0x12, 0x28, 0x0a, 0x05, 0x72, 0x61, 0x6e, 0x67, 0x65, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x74, 0x65, 0x6c, 0x65, 0x6d, 0x2e, 0x50, 0x42, 0x54,
	0x69, 0x6d, 0x65, 0x52, 0x61, 0x6e, 0x67, 0x65, 0x52, 0x05, 0x72, 0x61, 0x6e, 0x67, 0x65, 0x12,
	0x14, 0x0a, 0x05, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x04, 0x20, 0x01, 0x28, 0x03, 0x52, 0x05,
	0x73, 0x74, 0x61, 0x6d, 0x70, 0x12, 0x12, 0x0a, 0x04, 0x6b, 0x65, 0x79, 0x73, 0x18, 0x05, 0x20,
	0x03, 0x28, 0x0d, 0x52, 0x04, 0x6b, 0x65, 0x79, 0x73, 0x22, 0xdf, 0x01, 0x0a, 0x15, 0x46, 0x72,
	0x61, 0x6d, 0x65, 0x49, 0x74, 0x65, 0x72, 0x61, 0x74, 0x6f, 0x72, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x76, 0x61, 0x72, 0x69, 0x61, 0x6e, 0x74, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x05, 0x52, 0x07, 0x76, 0x61, 0x72, 0x69, 0x61, 0x6e, 0x74, 0x12, 0x18, 0x0a,
	0x07, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x07,
	0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x12, 0x23, 0x0a, 0x05, 0x66, 0x72, 0x61, 0x6d, 0x65,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0d, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e,
	0x46, 0x72, 0x61, 0x6d, 0x65, 0x52, 0x05, 0x66, 0x72, 0x61, 0x6d, 0x65, 0x12, 0x19, 0x0a, 0x08,
	0x6e, 0x6f, 0x64, 0x65, 0x5f, 0x6b, 0x65, 0x79, 0x18, 0x2b, 0x20, 0x01, 0x28, 0x05, 0x52, 0x07,
	0x6e, 0x6f, 0x64, 0x65, 0x4b, 0x65, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x61, 0x63, 0x6b, 0x18, 0x05,
	0x20, 0x01, 0x28, 0x08, 0x52, 0x03, 0x61, 0x63, 0x6b, 0x12, 0x17, 0x0a, 0x07, 0x73, 0x65, 0x71,
	0x5f, 0x6e, 0x75, 0x6d, 0x18, 0x06, 0x20, 0x01, 0x28, 0x05, 0x52, 0x06, 0x73, 0x65, 0x71, 0x4e,
	0x75, 0x6d, 0x12, 0x27, 0x0a, 0x05, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x18, 0x07, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x11, 0x2e, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x73, 0x2e, 0x50, 0x42, 0x50, 0x61, 0x79,
	0x6c, 0x6f, 0x61, 0x64, 0x52, 0x05, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x22, 0xd2, 0x02, 0x0a, 0x11,
	0x46, 0x72, 0x61, 0x6d, 0x65, 0x57, 0x72, 0x69, 0x74, 0x65, 0x72, 0x43, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x12, 0x12, 0x0a, 0x04, 0x6b, 0x65, 0x79, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0d, 0x52,
	0x04, 0x6b, 0x65, 0x79, 0x73, 0x12, 0x20, 0x0a, 0x0b, 0x61, 0x75, 0x74, 0x68, 0x6f, 0x72, 0x69,
	0x74, 0x69, 0x65, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0d, 0x52, 0x0b, 0x61, 0x75, 0x74, 0x68,
	0x6f, 0x72, 0x69, 0x74, 0x69, 0x65, 0x73, 0x12, 0x14, 0x0a, 0x05, 0x73, 0x74, 0x61, 0x72, 0x74,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x03, 0x52, 0x05, 0x73, 0x74, 0x61, 0x72, 0x74, 0x12, 0x40, 0x0a,
	0x0f, 0x63, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c, 0x5f, 0x73, 0x75, 0x62, 0x6a, 0x65, 0x63, 0x74,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x17, 0x2e, 0x63, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c,
	0x2e, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c, 0x53, 0x75, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x52,
	0x0e, 0x63, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c, 0x53, 0x75, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x12,
	0x12, 0x0a, 0x04, 0x6d, 0x6f, 0x64, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x05, 0x52, 0x04, 0x6d,
	0x6f, 0x64, 0x65, 0x12, 0x2c, 0x0a, 0x12, 0x65, 0x6e, 0x61, 0x62, 0x6c, 0x65, 0x5f, 0x61, 0x75,
	0x74, 0x6f, 0x5f, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x18, 0x06, 0x20, 0x01, 0x28, 0x08, 0x52,
	0x10, 0x65, 0x6e, 0x61, 0x62, 0x6c, 0x65, 0x41, 0x75, 0x74, 0x6f, 0x43, 0x6f, 0x6d, 0x6d, 0x69,
	0x74, 0x12, 0x3d, 0x0a, 0x1b, 0x61, 0x75, 0x74, 0x6f, 0x5f, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x5f,
	0x70, 0x65, 0x72, 0x73, 0x69, 0x73, 0x74, 0x5f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x76, 0x61, 0x6c,
	0x18, 0x07, 0x20, 0x01, 0x28, 0x03, 0x52, 0x18, 0x61, 0x75, 0x74, 0x6f, 0x49, 0x6e, 0x64, 0x65,
	0x78, 0x50, 0x65, 0x72, 0x73, 0x69, 0x73, 0x74, 0x49, 0x6e, 0x74, 0x65, 0x72, 0x76, 0x61, 0x6c,
	0x12, 0x2e, 0x0a, 0x13, 0x65, 0x72, 0x72, 0x5f, 0x6f, 0x6e, 0x5f, 0x75, 0x6e, 0x61, 0x75, 0x74,
	0x68, 0x6f, 0x72, 0x69, 0x7a, 0x65, 0x64, 0x18, 0x08, 0x20, 0x01, 0x28, 0x08, 0x52, 0x11, 0x65,
	0x72, 0x72, 0x4f, 0x6e, 0x55, 0x6e, 0x61, 0x75, 0x74, 0x68, 0x6f, 0x72, 0x69, 0x7a, 0x65, 0x64,
	0x22, 0x86, 0x01, 0x0a, 0x12, 0x46, 0x72, 0x61, 0x6d, 0x65, 0x57, 0x72, 0x69, 0x74, 0x65, 0x72,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x6f, 0x6d, 0x6d, 0x61,
	0x6e, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x07, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e,
	0x64, 0x12, 0x31, 0x0a, 0x06, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x19, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x46, 0x72, 0x61, 0x6d, 0x65,
	0x57, 0x72, 0x69, 0x74, 0x65, 0x72, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52, 0x06, 0x63, 0x6f,
	0x6e, 0x66, 0x69, 0x67, 0x12, 0x23, 0x0a, 0x05, 0x66, 0x72, 0x61, 0x6d, 0x65, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x0d, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x46, 0x72, 0x61,
	0x6d, 0x65, 0x52, 0x05, 0x66, 0x72, 0x61, 0x6d, 0x65, 0x22, 0xb1, 0x01, 0x0a, 0x13, 0x46, 0x72,
	0x61, 0x6d, 0x65, 0x57, 0x72, 0x69, 0x74, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x05, 0x52, 0x07, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x12, 0x10, 0x0a, 0x03, 0x61,
	0x63, 0x6b, 0x18, 0x02, 0x20, 0x01, 0x28, 0x08, 0x52, 0x03, 0x61, 0x63, 0x6b, 0x12, 0x19, 0x0a,
	0x08, 0x6e, 0x6f, 0x64, 0x65, 0x5f, 0x6b, 0x65, 0x79, 0x18, 0x03, 0x20, 0x01, 0x28, 0x05, 0x52,
	0x07, 0x6e, 0x6f, 0x64, 0x65, 0x4b, 0x65, 0x79, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x6f, 0x75, 0x6e,
	0x74, 0x65, 0x72, 0x18, 0x04, 0x20, 0x01, 0x28, 0x05, 0x52, 0x07, 0x63, 0x6f, 0x75, 0x6e, 0x74,
	0x65, 0x72, 0x12, 0x27, 0x0a, 0x05, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x18, 0x05, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x11, 0x2e, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x73, 0x2e, 0x50, 0x42, 0x50, 0x61, 0x79,
	0x6c, 0x6f, 0x61, 0x64, 0x52, 0x05, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x12, 0x10, 0x0a, 0x03, 0x65,
	0x6e, 0x64, 0x18, 0x06, 0x20, 0x01, 0x28, 0x03, 0x52, 0x03, 0x65, 0x6e, 0x64, 0x22, 0x40, 0x0a,
	0x14, 0x46, 0x72, 0x61, 0x6d, 0x65, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x65, 0x72, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x14, 0x0a, 0x05, 0x73, 0x74, 0x61, 0x72, 0x74, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x03, 0x52, 0x05, 0x73, 0x74, 0x61, 0x72, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x6b,
	0x65, 0x79, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0d, 0x52, 0x04, 0x6b, 0x65, 0x79, 0x73, 0x22,
	0x65, 0x0a, 0x15, 0x46, 0x72, 0x61, 0x6d, 0x65, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x65, 0x72,
	0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x23, 0x0a, 0x05, 0x66, 0x72, 0x61, 0x6d,
	0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0d, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31,
	0x2e, 0x46, 0x72, 0x61, 0x6d, 0x65, 0x52, 0x05, 0x66, 0x72, 0x61, 0x6d, 0x65, 0x12, 0x27, 0x0a,
	0x05, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x11, 0x2e, 0x65,
	0x72, 0x72, 0x6f, 0x72, 0x73, 0x2e, 0x50, 0x42, 0x50, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x52,
	0x05, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x22, 0x6a, 0x0a, 0x12, 0x46, 0x72, 0x61, 0x6d, 0x65, 0x44,
	0x65, 0x6c, 0x65, 0x74, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x12, 0x0a, 0x04,
	0x6b, 0x65, 0x79, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0d, 0x52, 0x04, 0x6b, 0x65, 0x79, 0x73,
	0x12, 0x14, 0x0a, 0x05, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x09, 0x52,
	0x05, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x12, 0x2a, 0x0a, 0x06, 0x62, 0x6f, 0x75, 0x6e, 0x64, 0x73,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x74, 0x65, 0x6c, 0x65, 0x6d, 0x2e, 0x50,
	0x42, 0x54, 0x69, 0x6d, 0x65, 0x52, 0x61, 0x6e, 0x67, 0x65, 0x52, 0x06, 0x62, 0x6f, 0x75, 0x6e,
	0x64, 0x73, 0x32, 0x61, 0x0a, 0x14, 0x46, 0x72, 0x61, 0x6d, 0x65, 0x49, 0x74, 0x65, 0x72, 0x61,
	0x74, 0x6f, 0x72, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x49, 0x0a, 0x04, 0x45, 0x78,
	0x65, 0x63, 0x12, 0x1c, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x46, 0x72, 0x61, 0x6d,
	0x65, 0x49, 0x74, 0x65, 0x72, 0x61, 0x74, 0x6f, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x1a, 0x1d, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x46, 0x72, 0x61, 0x6d, 0x65, 0x49,
	0x74, 0x65, 0x72, 0x61, 0x74, 0x6f, 0x72, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22,
	0x00, 0x28, 0x01, 0x30, 0x01, 0x32, 0x5b, 0x0a, 0x12, 0x46, 0x72, 0x61, 0x6d, 0x65, 0x57, 0x72,
	0x69, 0x74, 0x65, 0x72, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x45, 0x0a, 0x04, 0x45,
	0x78, 0x65, 0x63, 0x12, 0x1a, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x46, 0x72, 0x61,
	0x6d, 0x65, 0x57, 0x72, 0x69, 0x74, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a,
	0x1b, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x46, 0x72, 0x61, 0x6d, 0x65, 0x57, 0x72,
	0x69, 0x74, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x28, 0x01,
	0x30, 0x01, 0x32, 0x61, 0x0a, 0x14, 0x46, 0x72, 0x61, 0x6d, 0x65, 0x53, 0x74, 0x72, 0x65, 0x61,
	0x6d, 0x65, 0x72, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x49, 0x0a, 0x04, 0x45, 0x78,
	0x65, 0x63, 0x12, 0x1c, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x46, 0x72, 0x61, 0x6d,
	0x65, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x1a, 0x1d, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x46, 0x72, 0x61, 0x6d, 0x65, 0x53,
	0x74, 0x72, 0x65, 0x61, 0x6d, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22,
	0x00, 0x28, 0x01, 0x30, 0x01, 0x32, 0x52, 0x0a, 0x12, 0x46, 0x72, 0x61, 0x6d, 0x65, 0x44, 0x65,
	0x6c, 0x65, 0x74, 0x65, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x3c, 0x0a, 0x04, 0x45,
	0x78, 0x65, 0x63, 0x12, 0x1a, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x46, 0x72, 0x61,
	0x6d, 0x65, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a,
	0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22, 0x00, 0x42, 0x80, 0x01, 0x0a, 0x0a, 0x63, 0x6f,
	0x6d, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x42, 0x0b, 0x46, 0x72, 0x61, 0x6d, 0x65, 0x72,
	0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x2c, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e,
	0x63, 0x6f, 0x6d, 0x2f, 0x73, 0x79, 0x6e, 0x6e, 0x61, 0x78, 0x6c, 0x61, 0x62, 0x73, 0x2f, 0x73,
	0x79, 0x6e, 0x6e, 0x61, 0x78, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x67, 0x72,
	0x70, 0x63, 0x2f, 0x76, 0x31, 0xa2, 0x02, 0x03, 0x41, 0x58, 0x58, 0xaa, 0x02, 0x06, 0x41, 0x70,
	0x69, 0x2e, 0x56, 0x31, 0xca, 0x02, 0x06, 0x41, 0x70, 0x69, 0x5c, 0x56, 0x31, 0xe2, 0x02, 0x12,
	0x41, 0x70, 0x69, 0x5c, 0x56, 0x31, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61,
	0x74, 0x61, 0xea, 0x02, 0x07, 0x41, 0x70, 0x69, 0x3a, 0x3a, 0x56, 0x31, 0x62, 0x06, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_synnax_pkg_api_grpc_v1_framer_proto_rawDescOnce sync.Once
	file_synnax_pkg_api_grpc_v1_framer_proto_rawDescData = file_synnax_pkg_api_grpc_v1_framer_proto_rawDesc
)

func file_synnax_pkg_api_grpc_v1_framer_proto_rawDescGZIP() []byte {
	file_synnax_pkg_api_grpc_v1_framer_proto_rawDescOnce.Do(func() {
		file_synnax_pkg_api_grpc_v1_framer_proto_rawDescData = protoimpl.X.CompressGZIP(file_synnax_pkg_api_grpc_v1_framer_proto_rawDescData)
	})
	return file_synnax_pkg_api_grpc_v1_framer_proto_rawDescData
}

var file_synnax_pkg_api_grpc_v1_framer_proto_msgTypes = make([]protoimpl.MessageInfo, 9)
var file_synnax_pkg_api_grpc_v1_framer_proto_goTypes = []any{
	(*Frame)(nil),                  // 0: api.v1.Frame
	(*FrameIteratorRequest)(nil),   // 1: api.v1.FrameIteratorRequest
	(*FrameIteratorResponse)(nil),  // 2: api.v1.FrameIteratorResponse
	(*FrameWriterConfig)(nil),      // 3: api.v1.FrameWriterConfig
	(*FrameWriterRequest)(nil),     // 4: api.v1.FrameWriterRequest
	(*FrameWriterResponse)(nil),    // 5: api.v1.FrameWriterResponse
	(*FrameStreamerRequest)(nil),   // 6: api.v1.FrameStreamerRequest
	(*FrameStreamerResponse)(nil),  // 7: api.v1.FrameStreamerResponse
	(*FrameDeleteRequest)(nil),     // 8: api.v1.FrameDeleteRequest
	(*telem.PBSeries)(nil),         // 9: telem.PBSeries
	(*telem.PBTimeRange)(nil),      // 10: telem.PBTimeRange
	(*errors.PBPayload)(nil),       // 11: errors.PBPayload
	(*control.ControlSubject)(nil), // 12: control.ControlSubject
	(*emptypb.Empty)(nil),          // 13: google.protobuf.Empty
}
var file_synnax_pkg_api_grpc_v1_framer_proto_depIdxs = []int32{
	9,  // 0: api.v1.Frame.series:type_name -> telem.PBSeries
	10, // 1: api.v1.FrameIteratorRequest.range:type_name -> telem.PBTimeRange
	0,  // 2: api.v1.FrameIteratorResponse.frame:type_name -> api.v1.Frame
	11, // 3: api.v1.FrameIteratorResponse.error:type_name -> errors.PBPayload
	12, // 4: api.v1.FrameWriterConfig.control_subject:type_name -> control.ControlSubject
	3,  // 5: api.v1.FrameWriterRequest.config:type_name -> api.v1.FrameWriterConfig
	0,  // 6: api.v1.FrameWriterRequest.frame:type_name -> api.v1.Frame
	11, // 7: api.v1.FrameWriterResponse.error:type_name -> errors.PBPayload
	0,  // 8: api.v1.FrameStreamerResponse.frame:type_name -> api.v1.Frame
	11, // 9: api.v1.FrameStreamerResponse.error:type_name -> errors.PBPayload
	10, // 10: api.v1.FrameDeleteRequest.bounds:type_name -> telem.PBTimeRange
	1,  // 11: api.v1.FrameIteratorService.Exec:input_type -> api.v1.FrameIteratorRequest
	4,  // 12: api.v1.FrameWriterService.Exec:input_type -> api.v1.FrameWriterRequest
	6,  // 13: api.v1.FrameStreamerService.Exec:input_type -> api.v1.FrameStreamerRequest
	8,  // 14: api.v1.FrameDeleteService.Exec:input_type -> api.v1.FrameDeleteRequest
	2,  // 15: api.v1.FrameIteratorService.Exec:output_type -> api.v1.FrameIteratorResponse
	5,  // 16: api.v1.FrameWriterService.Exec:output_type -> api.v1.FrameWriterResponse
	7,  // 17: api.v1.FrameStreamerService.Exec:output_type -> api.v1.FrameStreamerResponse
	13, // 18: api.v1.FrameDeleteService.Exec:output_type -> google.protobuf.Empty
	15, // [15:19] is the sub-list for method output_type
	11, // [11:15] is the sub-list for method input_type
	11, // [11:11] is the sub-list for extension type_name
	11, // [11:11] is the sub-list for extension extendee
	0,  // [0:11] is the sub-list for field type_name
}

func init() { file_synnax_pkg_api_grpc_v1_framer_proto_init() }
func file_synnax_pkg_api_grpc_v1_framer_proto_init() {
	if File_synnax_pkg_api_grpc_v1_framer_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_synnax_pkg_api_grpc_v1_framer_proto_msgTypes[0].Exporter = func(v any, i int) any {
			switch v := v.(*Frame); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_synnax_pkg_api_grpc_v1_framer_proto_msgTypes[1].Exporter = func(v any, i int) any {
			switch v := v.(*FrameIteratorRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_synnax_pkg_api_grpc_v1_framer_proto_msgTypes[2].Exporter = func(v any, i int) any {
			switch v := v.(*FrameIteratorResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_synnax_pkg_api_grpc_v1_framer_proto_msgTypes[3].Exporter = func(v any, i int) any {
			switch v := v.(*FrameWriterConfig); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_synnax_pkg_api_grpc_v1_framer_proto_msgTypes[4].Exporter = func(v any, i int) any {
			switch v := v.(*FrameWriterRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_synnax_pkg_api_grpc_v1_framer_proto_msgTypes[5].Exporter = func(v any, i int) any {
			switch v := v.(*FrameWriterResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_synnax_pkg_api_grpc_v1_framer_proto_msgTypes[6].Exporter = func(v any, i int) any {
			switch v := v.(*FrameStreamerRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_synnax_pkg_api_grpc_v1_framer_proto_msgTypes[7].Exporter = func(v any, i int) any {
			switch v := v.(*FrameStreamerResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_synnax_pkg_api_grpc_v1_framer_proto_msgTypes[8].Exporter = func(v any, i int) any {
			switch v := v.(*FrameDeleteRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_synnax_pkg_api_grpc_v1_framer_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   9,
			NumExtensions: 0,
			NumServices:   4,
		},
		GoTypes:           file_synnax_pkg_api_grpc_v1_framer_proto_goTypes,
		DependencyIndexes: file_synnax_pkg_api_grpc_v1_framer_proto_depIdxs,
		MessageInfos:      file_synnax_pkg_api_grpc_v1_framer_proto_msgTypes,
	}.Build()
	File_synnax_pkg_api_grpc_v1_framer_proto = out.File
	file_synnax_pkg_api_grpc_v1_framer_proto_rawDesc = nil
	file_synnax_pkg_api_grpc_v1_framer_proto_goTypes = nil
	file_synnax_pkg_api_grpc_v1_framer_proto_depIdxs = nil
}
