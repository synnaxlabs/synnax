#include "freighter.h"
#include "service.pb.h"
#include "service.grpc.pb.h"

using request_t = masa::Values;

struct response_t
{
    request_t payload;
    grpc::Status status;
}

struct stream_t {};

int main()
{
    auto client = gRPC<response_t, request_t, stream_t, rpc_t>();
    
}