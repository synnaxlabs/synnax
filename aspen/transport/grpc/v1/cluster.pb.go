// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        (unknown)
// source: aspen/transport/grpc/v1/cluster.proto

package v1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
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

type Node struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Key           uint32                 `protobuf:"varint,1,opt,name=key,proto3" json:"key,omitempty"`
	Address       string                 `protobuf:"bytes,2,opt,name=address,proto3" json:"address,omitempty"`
	State         uint32                 `protobuf:"varint,3,opt,name=state,proto3" json:"state,omitempty"`
	Heartbeat     *Heartbeat             `protobuf:"bytes,4,opt,name=heartbeat,proto3" json:"heartbeat,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Node) Reset() {
	*x = Node{}
	mi := &file_aspen_transport_grpc_v1_cluster_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Node) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Node) ProtoMessage() {}

func (x *Node) ProtoReflect() protoreflect.Message {
	mi := &file_aspen_transport_grpc_v1_cluster_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Node.ProtoReflect.Descriptor instead.
func (*Node) Descriptor() ([]byte, []int) {
	return file_aspen_transport_grpc_v1_cluster_proto_rawDescGZIP(), []int{0}
}

func (x *Node) GetKey() uint32 {
	if x != nil {
		return x.Key
	}
	return 0
}

func (x *Node) GetAddress() string {
	if x != nil {
		return x.Address
	}
	return ""
}

func (x *Node) GetState() uint32 {
	if x != nil {
		return x.State
	}
	return 0
}

func (x *Node) GetHeartbeat() *Heartbeat {
	if x != nil {
		return x.Heartbeat
	}
	return nil
}

type Heartbeat struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Generation    uint32                 `protobuf:"varint,1,opt,name=generation,proto3" json:"generation,omitempty"`
	Version       uint32                 `protobuf:"varint,2,opt,name=version,proto3" json:"version,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Heartbeat) Reset() {
	*x = Heartbeat{}
	mi := &file_aspen_transport_grpc_v1_cluster_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Heartbeat) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Heartbeat) ProtoMessage() {}

func (x *Heartbeat) ProtoReflect() protoreflect.Message {
	mi := &file_aspen_transport_grpc_v1_cluster_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Heartbeat.ProtoReflect.Descriptor instead.
func (*Heartbeat) Descriptor() ([]byte, []int) {
	return file_aspen_transport_grpc_v1_cluster_proto_rawDescGZIP(), []int{1}
}

func (x *Heartbeat) GetGeneration() uint32 {
	if x != nil {
		return x.Generation
	}
	return 0
}

func (x *Heartbeat) GetVersion() uint32 {
	if x != nil {
		return x.Version
	}
	return 0
}

type NodeDigest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            uint32                 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Heartbeat     *Heartbeat             `protobuf:"bytes,2,opt,name=heartbeat,proto3" json:"heartbeat,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *NodeDigest) Reset() {
	*x = NodeDigest{}
	mi := &file_aspen_transport_grpc_v1_cluster_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *NodeDigest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NodeDigest) ProtoMessage() {}

func (x *NodeDigest) ProtoReflect() protoreflect.Message {
	mi := &file_aspen_transport_grpc_v1_cluster_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NodeDigest.ProtoReflect.Descriptor instead.
func (*NodeDigest) Descriptor() ([]byte, []int) {
	return file_aspen_transport_grpc_v1_cluster_proto_rawDescGZIP(), []int{2}
}

func (x *NodeDigest) GetId() uint32 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *NodeDigest) GetHeartbeat() *Heartbeat {
	if x != nil {
		return x.Heartbeat
	}
	return nil
}

type ClusterGossip struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Digests       map[uint32]*NodeDigest `protobuf:"bytes,1,rep,name=digests,proto3" json:"digests,omitempty" protobuf_key:"varint,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	Nodes         map[uint32]*Node       `protobuf:"bytes,2,rep,name=nodes,proto3" json:"nodes,omitempty" protobuf_key:"varint,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ClusterGossip) Reset() {
	*x = ClusterGossip{}
	mi := &file_aspen_transport_grpc_v1_cluster_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ClusterGossip) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ClusterGossip) ProtoMessage() {}

func (x *ClusterGossip) ProtoReflect() protoreflect.Message {
	mi := &file_aspen_transport_grpc_v1_cluster_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ClusterGossip.ProtoReflect.Descriptor instead.
func (*ClusterGossip) Descriptor() ([]byte, []int) {
	return file_aspen_transport_grpc_v1_cluster_proto_rawDescGZIP(), []int{3}
}

func (x *ClusterGossip) GetDigests() map[uint32]*NodeDigest {
	if x != nil {
		return x.Digests
	}
	return nil
}

func (x *ClusterGossip) GetNodes() map[uint32]*Node {
	if x != nil {
		return x.Nodes
	}
	return nil
}

type ClusterPledge struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	ClusterKey    string                 `protobuf:"bytes,1,opt,name=cluster_key,json=clusterKey,proto3" json:"cluster_key,omitempty"`
	NodeKey       uint32                 `protobuf:"varint,2,opt,name=node_key,json=nodeKey,proto3" json:"node_key,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ClusterPledge) Reset() {
	*x = ClusterPledge{}
	mi := &file_aspen_transport_grpc_v1_cluster_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ClusterPledge) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ClusterPledge) ProtoMessage() {}

func (x *ClusterPledge) ProtoReflect() protoreflect.Message {
	mi := &file_aspen_transport_grpc_v1_cluster_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ClusterPledge.ProtoReflect.Descriptor instead.
func (*ClusterPledge) Descriptor() ([]byte, []int) {
	return file_aspen_transport_grpc_v1_cluster_proto_rawDescGZIP(), []int{4}
}

func (x *ClusterPledge) GetClusterKey() string {
	if x != nil {
		return x.ClusterKey
	}
	return ""
}

func (x *ClusterPledge) GetNodeKey() uint32 {
	if x != nil {
		return x.NodeKey
	}
	return 0
}

var File_aspen_transport_grpc_v1_cluster_proto protoreflect.FileDescriptor

const file_aspen_transport_grpc_v1_cluster_proto_rawDesc = "" +
	"\n" +
	"%aspen/transport/grpc/v1/cluster.proto\x12\baspen.v1\"{\n" +
	"\x04Node\x12\x10\n" +
	"\x03key\x18\x01 \x01(\rR\x03key\x12\x18\n" +
	"\aaddress\x18\x02 \x01(\tR\aaddress\x12\x14\n" +
	"\x05state\x18\x03 \x01(\rR\x05state\x121\n" +
	"\theartbeat\x18\x04 \x01(\v2\x13.aspen.v1.HeartbeatR\theartbeat\"E\n" +
	"\tHeartbeat\x12\x1e\n" +
	"\n" +
	"generation\x18\x01 \x01(\rR\n" +
	"generation\x12\x18\n" +
	"\aversion\x18\x02 \x01(\rR\aversion\"O\n" +
	"\n" +
	"NodeDigest\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\rR\x02id\x121\n" +
	"\theartbeat\x18\x02 \x01(\v2\x13.aspen.v1.HeartbeatR\theartbeat\"\xa5\x02\n" +
	"\rClusterGossip\x12>\n" +
	"\adigests\x18\x01 \x03(\v2$.aspen.v1.ClusterGossip.DigestsEntryR\adigests\x128\n" +
	"\x05nodes\x18\x02 \x03(\v2\".aspen.v1.ClusterGossip.NodesEntryR\x05nodes\x1aP\n" +
	"\fDigestsEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\rR\x03key\x12*\n" +
	"\x05value\x18\x02 \x01(\v2\x14.aspen.v1.NodeDigestR\x05value:\x028\x01\x1aH\n" +
	"\n" +
	"NodesEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\rR\x03key\x12$\n" +
	"\x05value\x18\x02 \x01(\v2\x0e.aspen.v1.NodeR\x05value:\x028\x01\"K\n" +
	"\rClusterPledge\x12\x1f\n" +
	"\vcluster_key\x18\x01 \x01(\tR\n" +
	"clusterKey\x12\x19\n" +
	"\bnode_key\x18\x02 \x01(\rR\anodeKey2P\n" +
	"\x14ClusterGossipService\x128\n" +
	"\x04Exec\x12\x17.aspen.v1.ClusterGossip\x1a\x17.aspen.v1.ClusterGossip2I\n" +
	"\rPledgeService\x128\n" +
	"\x04Exec\x12\x17.aspen.v1.ClusterPledge\x1a\x17.aspen.v1.ClusterPledgeB\x8c\x01\n" +
	"\fcom.aspen.v1B\fClusterProtoP\x01Z-github.com/synnaxlabs/aspen/transport/grpc/v1\xa2\x02\x03AXX\xaa\x02\bAspen.V1\xca\x02\bAspen\\V1\xe2\x02\x14Aspen\\V1\\GPBMetadata\xea\x02\tAspen::V1b\x06proto3"

var (
	file_aspen_transport_grpc_v1_cluster_proto_rawDescOnce sync.Once
	file_aspen_transport_grpc_v1_cluster_proto_rawDescData []byte
)

func file_aspen_transport_grpc_v1_cluster_proto_rawDescGZIP() []byte {
	file_aspen_transport_grpc_v1_cluster_proto_rawDescOnce.Do(func() {
		file_aspen_transport_grpc_v1_cluster_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_aspen_transport_grpc_v1_cluster_proto_rawDesc), len(file_aspen_transport_grpc_v1_cluster_proto_rawDesc)))
	})
	return file_aspen_transport_grpc_v1_cluster_proto_rawDescData
}

var file_aspen_transport_grpc_v1_cluster_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_aspen_transport_grpc_v1_cluster_proto_goTypes = []any{
	(*Node)(nil),          // 0: aspen.v1.Node
	(*Heartbeat)(nil),     // 1: aspen.v1.Heartbeat
	(*NodeDigest)(nil),    // 2: aspen.v1.NodeDigest
	(*ClusterGossip)(nil), // 3: aspen.v1.ClusterGossip
	(*ClusterPledge)(nil), // 4: aspen.v1.ClusterPledge
	nil,                   // 5: aspen.v1.ClusterGossip.DigestsEntry
	nil,                   // 6: aspen.v1.ClusterGossip.NodesEntry
}
var file_aspen_transport_grpc_v1_cluster_proto_depIdxs = []int32{
	1, // 0: aspen.v1.Node.heartbeat:type_name -> aspen.v1.Heartbeat
	1, // 1: aspen.v1.NodeDigest.heartbeat:type_name -> aspen.v1.Heartbeat
	5, // 2: aspen.v1.ClusterGossip.digests:type_name -> aspen.v1.ClusterGossip.DigestsEntry
	6, // 3: aspen.v1.ClusterGossip.nodes:type_name -> aspen.v1.ClusterGossip.NodesEntry
	2, // 4: aspen.v1.ClusterGossip.DigestsEntry.value:type_name -> aspen.v1.NodeDigest
	0, // 5: aspen.v1.ClusterGossip.NodesEntry.value:type_name -> aspen.v1.Node
	3, // 6: aspen.v1.ClusterGossipService.Exec:input_type -> aspen.v1.ClusterGossip
	4, // 7: aspen.v1.PledgeService.Exec:input_type -> aspen.v1.ClusterPledge
	3, // 8: aspen.v1.ClusterGossipService.Exec:output_type -> aspen.v1.ClusterGossip
	4, // 9: aspen.v1.PledgeService.Exec:output_type -> aspen.v1.ClusterPledge
	8, // [8:10] is the sub-list for method output_type
	6, // [6:8] is the sub-list for method input_type
	6, // [6:6] is the sub-list for extension type_name
	6, // [6:6] is the sub-list for extension extendee
	0, // [0:6] is the sub-list for field type_name
}

func init() { file_aspen_transport_grpc_v1_cluster_proto_init() }
func file_aspen_transport_grpc_v1_cluster_proto_init() {
	if File_aspen_transport_grpc_v1_cluster_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_aspen_transport_grpc_v1_cluster_proto_rawDesc), len(file_aspen_transport_grpc_v1_cluster_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   2,
		},
		GoTypes:           file_aspen_transport_grpc_v1_cluster_proto_goTypes,
		DependencyIndexes: file_aspen_transport_grpc_v1_cluster_proto_depIdxs,
		MessageInfos:      file_aspen_transport_grpc_v1_cluster_proto_msgTypes,
	}.Build()
	File_aspen_transport_grpc_v1_cluster_proto = out.File
	file_aspen_transport_grpc_v1_cluster_proto_goTypes = nil
	file_aspen_transport_grpc_v1_cluster_proto_depIdxs = nil
}
