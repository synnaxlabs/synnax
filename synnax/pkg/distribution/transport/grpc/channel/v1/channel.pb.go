// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.2
// 	protoc        (unknown)
// source: synnax/pkg/distribution/transport/grpc/channel/v1/channel.proto

package v1

import (
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

type CreateMessage struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Channels []*Channel `protobuf:"bytes,1,rep,name=channels,proto3" json:"channels,omitempty"`
}

func (x *CreateMessage) Reset() {
	*x = CreateMessage{}
	if protoimpl.UnsafeEnabled {
		mi := &file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CreateMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateMessage) ProtoMessage() {}

func (x *CreateMessage) ProtoReflect() protoreflect.Message {
	mi := &file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateMessage.ProtoReflect.Descriptor instead.
func (*CreateMessage) Descriptor() ([]byte, []int) {
	return file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_rawDescGZIP(), []int{0}
}

func (x *CreateMessage) GetChannels() []*Channel {
	if x != nil {
		return x.Channels
	}
	return nil
}

type DeleteRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Keys []uint32 `protobuf:"varint,3,rep,packed,name=keys,proto3" json:"keys,omitempty"`
}

func (x *DeleteRequest) Reset() {
	*x = DeleteRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DeleteRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteRequest) ProtoMessage() {}

func (x *DeleteRequest) ProtoReflect() protoreflect.Message {
	mi := &file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeleteRequest.ProtoReflect.Descriptor instead.
func (*DeleteRequest) Descriptor() ([]byte, []int) {
	return file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_rawDescGZIP(), []int{1}
}

func (x *DeleteRequest) GetKeys() []uint32 {
	if x != nil {
		return x.Keys
	}
	return nil
}

type RenameRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Keys  []uint32 `protobuf:"varint,1,rep,packed,name=keys,proto3" json:"keys,omitempty"`
	Names []string `protobuf:"bytes,2,rep,name=names,proto3" json:"names,omitempty"`
}

func (x *RenameRequest) Reset() {
	*x = RenameRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RenameRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RenameRequest) ProtoMessage() {}

func (x *RenameRequest) ProtoReflect() protoreflect.Message {
	mi := &file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RenameRequest.ProtoReflect.Descriptor instead.
func (*RenameRequest) Descriptor() ([]byte, []int) {
	return file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_rawDescGZIP(), []int{2}
}

func (x *RenameRequest) GetKeys() []uint32 {
	if x != nil {
		return x.Keys
	}
	return nil
}

func (x *RenameRequest) GetNames() []string {
	if x != nil {
		return x.Names
	}
	return nil
}

type Channel struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name         string  `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	NodeId       int32   `protobuf:"varint,2,opt,name=node_id,json=nodeId,proto3" json:"node_id,omitempty"`
	StorageKey   uint32  `protobuf:"varint,3,opt,name=storage_key,json=storageKey,proto3" json:"storage_key,omitempty"`
	Rate         float64 `protobuf:"fixed64,4,opt,name=rate,proto3" json:"rate,omitempty"`
	DataType     string  `protobuf:"bytes,5,opt,name=data_type,json=dataType,proto3" json:"data_type,omitempty"`
	StorageIndex int32   `protobuf:"varint,6,opt,name=storage_index,json=storageIndex,proto3" json:"storage_index,omitempty"`
	IsIndex      bool    `protobuf:"varint,7,opt,name=is_index,json=isIndex,proto3" json:"is_index,omitempty"`
}

func (x *Channel) Reset() {
	*x = Channel{}
	if protoimpl.UnsafeEnabled {
		mi := &file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Channel) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Channel) ProtoMessage() {}

func (x *Channel) ProtoReflect() protoreflect.Message {
	mi := &file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Channel.ProtoReflect.Descriptor instead.
func (*Channel) Descriptor() ([]byte, []int) {
	return file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_rawDescGZIP(), []int{3}
}

func (x *Channel) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Channel) GetNodeId() int32 {
	if x != nil {
		return x.NodeId
	}
	return 0
}

func (x *Channel) GetStorageKey() uint32 {
	if x != nil {
		return x.StorageKey
	}
	return 0
}

func (x *Channel) GetRate() float64 {
	if x != nil {
		return x.Rate
	}
	return 0
}

func (x *Channel) GetDataType() string {
	if x != nil {
		return x.DataType
	}
	return ""
}

func (x *Channel) GetStorageIndex() int32 {
	if x != nil {
		return x.StorageIndex
	}
	return 0
}

func (x *Channel) GetIsIndex() bool {
	if x != nil {
		return x.IsIndex
	}
	return false
}

var File_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto protoreflect.FileDescriptor

var file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_rawDesc = []byte{
	0x0a, 0x3f, 0x73, 0x79, 0x6e, 0x6e, 0x61, 0x78, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x64, 0x69, 0x73,
	0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x70,
	0x6f, 0x72, 0x74, 0x2f, 0x67, 0x72, 0x70, 0x63, 0x2f, 0x63, 0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c,
	0x2f, 0x76, 0x31, 0x2f, 0x63, 0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x0a, 0x63, 0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x2e, 0x76, 0x31, 0x1a, 0x1b, 0x67,
	0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x65,
	0x6d, 0x70, 0x74, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x40, 0x0a, 0x0d, 0x43, 0x72,
	0x65, 0x61, 0x74, 0x65, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x2f, 0x0a, 0x08, 0x63,
	0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x13, 0x2e,
	0x63, 0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x68, 0x61, 0x6e, 0x6e,
	0x65, 0x6c, 0x52, 0x08, 0x63, 0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x73, 0x22, 0x23, 0x0a, 0x0d,
	0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x12, 0x0a,
	0x04, 0x6b, 0x65, 0x79, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0d, 0x52, 0x04, 0x6b, 0x65, 0x79,
	0x73, 0x22, 0x39, 0x0a, 0x0d, 0x52, 0x65, 0x6e, 0x61, 0x6d, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x6b, 0x65, 0x79, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0d,
	0x52, 0x04, 0x6b, 0x65, 0x79, 0x73, 0x12, 0x14, 0x0a, 0x05, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x18,
	0x02, 0x20, 0x03, 0x28, 0x09, 0x52, 0x05, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x22, 0xc8, 0x01, 0x0a,
	0x07, 0x43, 0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x17, 0x0a, 0x07,
	0x6e, 0x6f, 0x64, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x06, 0x6e,
	0x6f, 0x64, 0x65, 0x49, 0x64, 0x12, 0x1f, 0x0a, 0x0b, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65,
	0x5f, 0x6b, 0x65, 0x79, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x0a, 0x73, 0x74, 0x6f, 0x72,
	0x61, 0x67, 0x65, 0x4b, 0x65, 0x79, 0x12, 0x12, 0x0a, 0x04, 0x72, 0x61, 0x74, 0x65, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x01, 0x52, 0x04, 0x72, 0x61, 0x74, 0x65, 0x12, 0x1b, 0x0a, 0x09, 0x64, 0x61,
	0x74, 0x61, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x64,
	0x61, 0x74, 0x61, 0x54, 0x79, 0x70, 0x65, 0x12, 0x23, 0x0a, 0x0d, 0x73, 0x74, 0x6f, 0x72, 0x61,
	0x67, 0x65, 0x5f, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x18, 0x06, 0x20, 0x01, 0x28, 0x05, 0x52, 0x0c,
	0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x49, 0x6e, 0x64, 0x65, 0x78, 0x12, 0x19, 0x0a, 0x08,
	0x69, 0x73, 0x5f, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x18, 0x07, 0x20, 0x01, 0x28, 0x08, 0x52, 0x07,
	0x69, 0x73, 0x49, 0x6e, 0x64, 0x65, 0x78, 0x32, 0x56, 0x0a, 0x14, 0x43, 0x68, 0x61, 0x6e, 0x6e,
	0x65, 0x6c, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12,
	0x3e, 0x0a, 0x04, 0x45, 0x78, 0x65, 0x63, 0x12, 0x19, 0x2e, 0x63, 0x68, 0x61, 0x6e, 0x6e, 0x65,
	0x6c, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x4d, 0x65, 0x73, 0x73, 0x61,
	0x67, 0x65, 0x1a, 0x19, 0x2e, 0x63, 0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x2e, 0x76, 0x31, 0x2e,
	0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x22, 0x00, 0x32,
	0x53, 0x0a, 0x14, 0x43, 0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65,
	0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x3b, 0x0a, 0x04, 0x45, 0x78, 0x65, 0x63, 0x12,
	0x19, 0x2e, 0x63, 0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x2e, 0x76, 0x31, 0x2e, 0x44, 0x65, 0x6c,
	0x65, 0x74, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70,
	0x74, 0x79, 0x22, 0x00, 0x32, 0x53, 0x0a, 0x14, 0x43, 0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x52,
	0x65, 0x6e, 0x61, 0x6d, 0x65, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x3b, 0x0a, 0x04,
	0x45, 0x78, 0x65, 0x63, 0x12, 0x19, 0x2e, 0x63, 0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x2e, 0x76,
	0x31, 0x2e, 0x52, 0x65, 0x6e, 0x61, 0x6d, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a,
	0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22, 0x00, 0x42, 0xb0, 0x01, 0x0a, 0x0e, 0x63, 0x6f,
	0x6d, 0x2e, 0x63, 0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x2e, 0x76, 0x31, 0x42, 0x0c, 0x43, 0x68,
	0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x47, 0x67, 0x69,
	0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x73, 0x79, 0x6e, 0x6e, 0x61, 0x78, 0x6c,
	0x61, 0x62, 0x73, 0x2f, 0x73, 0x79, 0x6e, 0x6e, 0x61, 0x78, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x64,
	0x69, 0x73, 0x74, 0x72, 0x69, 0x62, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x74, 0x72, 0x61, 0x6e,
	0x73, 0x70, 0x6f, 0x72, 0x74, 0x2f, 0x67, 0x72, 0x70, 0x63, 0x2f, 0x63, 0x68, 0x61, 0x6e, 0x6e,
	0x65, 0x6c, 0x2f, 0x76, 0x31, 0xa2, 0x02, 0x03, 0x43, 0x58, 0x58, 0xaa, 0x02, 0x0a, 0x43, 0x68,
	0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x2e, 0x56, 0x31, 0xca, 0x02, 0x0a, 0x43, 0x68, 0x61, 0x6e, 0x6e,
	0x65, 0x6c, 0x5c, 0x56, 0x31, 0xe2, 0x02, 0x16, 0x43, 0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x5c,
	0x56, 0x31, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02,
	0x0b, 0x43, 0x68, 0x61, 0x6e, 0x6e, 0x65, 0x6c, 0x3a, 0x3a, 0x56, 0x31, 0x62, 0x06, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_rawDescOnce sync.Once
	file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_rawDescData = file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_rawDesc
)

func file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_rawDescGZIP() []byte {
	file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_rawDescOnce.Do(func() {
		file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_rawDescData = protoimpl.X.CompressGZIP(file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_rawDescData)
	})
	return file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_rawDescData
}

var file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_goTypes = []any{
	(*CreateMessage)(nil), // 0: channel.v1.CreateMessage
	(*DeleteRequest)(nil), // 1: channel.v1.DeleteRequest
	(*RenameRequest)(nil), // 2: channel.v1.RenameRequest
	(*Channel)(nil),       // 3: channel.v1.Channel
	(*emptypb.Empty)(nil), // 4: google.protobuf.Empty
}
var file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_depIdxs = []int32{
	3, // 0: channel.v1.CreateMessage.channels:type_name -> channel.v1.Channel
	0, // 1: channel.v1.ChannelCreateService.Exec:input_type -> channel.v1.CreateMessage
	1, // 2: channel.v1.ChannelDeleteService.Exec:input_type -> channel.v1.DeleteRequest
	2, // 3: channel.v1.ChannelRenameService.Exec:input_type -> channel.v1.RenameRequest
	0, // 4: channel.v1.ChannelCreateService.Exec:output_type -> channel.v1.CreateMessage
	4, // 5: channel.v1.ChannelDeleteService.Exec:output_type -> google.protobuf.Empty
	4, // 6: channel.v1.ChannelRenameService.Exec:output_type -> google.protobuf.Empty
	4, // [4:7] is the sub-list for method output_type
	1, // [1:4] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_init() }
func file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_init() {
	if File_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_msgTypes[0].Exporter = func(v any, i int) any {
			switch v := v.(*CreateMessage); i {
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
		file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_msgTypes[1].Exporter = func(v any, i int) any {
			switch v := v.(*DeleteRequest); i {
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
		file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_msgTypes[2].Exporter = func(v any, i int) any {
			switch v := v.(*RenameRequest); i {
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
		file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_msgTypes[3].Exporter = func(v any, i int) any {
			switch v := v.(*Channel); i {
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
			RawDescriptor: file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   3,
		},
		GoTypes:           file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_goTypes,
		DependencyIndexes: file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_depIdxs,
		MessageInfos:      file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_msgTypes,
	}.Build()
	File_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto = out.File
	file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_rawDesc = nil
	file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_goTypes = nil
	file_synnax_pkg_distribution_transport_grpc_channel_v1_channel_proto_depIdxs = nil
}
