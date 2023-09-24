#include <gtest/gtest.h>
#include "src/gRPC/protos/service.grpc.pb.h"
#include "client.h"

struct stream_t
{
};

TEST(testGRPC, basicProto)
{
    auto m = test::Message();
    m.set_payload("Hello");

    ASSERT_EQ(m.payload(), "Hello");
}

TEST(testGRPC, testBasicClientServer)
{
    auto a = gRPC<test::Message, test::Message, stream_t, test::messageService>();
    auto mes = test::Message();
    a.send("localhost", mes);
}