/// Local headers.
#include "client.h"

/// std.
#include <string>

/// grpc.
// #include <grpc/grpc.h>
// #include <grpcpp/channel.h>

// TODO: Encompass some error handling
template <typename response_t, typename request_t, typename stream_t>
response_t gRPC<response_t, request_t, stream_t>::send(std::string target, request_t &request)
{
    auto res = stub->Exec(request);
    return res;
}