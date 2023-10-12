#include "synnax/transport.h"
#include "freighter/freighter.h"
#include "freighter/gRPC/client.h"
#include "v1/framer.pb.h"
#include "v1/framer.grpc.pb.h"
#include "v1/ranger.pb.h"
#include "v1/ranger.grpc.pb.h"
#include "v1/channel.pb.h"
#include "v1/channel.grpc.pb.h"

using namespace api;

struct Empty {
};


Transport::Transport(uint16_t port, const std::string &ip) {
    auto url = Freighter::URL(ip, port);
    auto pool = new GRPCPool();

//    frame_iter = new GRPCStreamClient<
//            v1::FrameIteratorResponse,
//            v1::FrameIteratorRequest,
//            grpc::Status,
//            v1::FrameService
//    >(url, pool);

//    frame_stream = new GRPCStreamClient<
//            v1::FrameStreamerResponse,
//            v1::FrameStreamerRequest,
//            grpc::Status,
//            v1::FrameStreamerService,
//    >(url, pool);
//
//    frame_write = new GRPCStreamClient<
//            v1::FrameWriterResponse,
//            v1::FrameWriterRequest,
//            grpc::Status,
//            v1::FrameWriterService,
//    >(url, pool);
//
//    chan_create = new GRPCUnaryClient<
//            v1::ChannelCreateResponse,
//            v1::ChannelCreateRequest,
//            grpc::Status,
//            ChannelCreateService
//    >(url, pool);
//
    auto v = new GRPCUnaryClient<
            v1::ChannelRetrieveResponse,
            v1::ChannelRetrieveRequest,
            grpc::Status,
            v1::ChannelRetrieveService
    >(pool, url.toString());
//
//    range_retrieve = new GRPCUnaryClient<
//            v1::RangeRetrieveResponse,
//            v1::RangeRetrieveRequest,
//            grpc::Status,
//            v1::RangeRetrieveService,
//    >(url, pool);
//
//    range_create = new GRPCUnaryClient<
//            v1::RangeCreateResponse,
//            v1::RangeCreateRequest,
//            grpc::Status,
//            v1::RangeCreateService,
//    >(
//            url, pool);
//    range_kv_delete = new GRPCUnaryClient<Empty, v1::RangeKVDeleteRequest, grpc::Status, RangeKVDeleteService>(
//            url, pool);
//    range_kv_get = new GRPCUnaryClient<v1::RangeKVGetResponse, v1::RangeKVGetRequest, grpc::Status, RangeKVGetService>(
//            url, pool);
//    range_kv_set = new GRPCUnaryClient<Empty, v1::RangeKVSetRequest, grpc::Status, RangeKVSetService>(url, pool);
}
