/// Local headers.
#include "client.h"

/// std.
#include <string>

/// grpc.
// #include <grpc/grpc.h>
// #include <grpcpp/channel.h>

// TODO: Encompass some error handling
template <typename response_t, typename request_t, typename stream_t, typename rpc_t>
response_t gRPC<response_t, request_t, stream_t, rpc_t>::send(std::string target, request_t &request)
{
    // To abstract the interface, we construct the stub only if needed.
    if (!stub || target != last_target)
    {
        // TODO: Set up crypto context.
        auto channel = grpc::CreateChannel(target, grpc::InsecureChannelCredentials());
        stub = rpc_t::NewStub(channel);
    }

    auto res = stub->Exec(request);
    return res;
}