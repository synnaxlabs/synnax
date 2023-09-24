#include "src/gRPC/protos/service.grpc.pb.h"
#include "client.h"

class stream_t {};

int main()
{
    auto a = gRPC<test::Message, test::Message, stream_t, test::messageService>();
    auto mes = test::Message();
    mes.set_payload("Sending to Server");
    a.send("localhost:8080", mes);
}