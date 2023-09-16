#include <gtest/gtest.h>
#include "src/gRPC/service.pb.h"

TEST(basicTest, basicProto)
{
    auto m = test::Message();
    m.set_payload("Hello");

    ASSERT_EQ(m.payload(), "Hello");
}