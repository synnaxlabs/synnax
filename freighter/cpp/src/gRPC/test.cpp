#include <gtest/gtest.h>
#include "src/gRPC/protos/service.grpc.pb.h"
#include "client.h"

TEST(testGRPC, basicProto)
{
    auto m = test::Message();
    m.set_payload("Hello");

    ASSERT_EQ(m.payload(), "Hello");
}

TEST(testGRPC, testCtor)
{
    // auto a = gRPC<void *, void *, void *, freighter::messageService>();
}