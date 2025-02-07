//
// Created by Emiliano Bonilla on 1/21/25.
//

/// external.
#include "gtest/gtest.h"

/// internal.
#include "driver/sequence/sequence.h"
#include "driver/sequence/channel_set_operator.h"
#include "driver/sequence/json_operator.h"

class TestSink : public Sink {
public:
    freighter::Error write(synnax::Frame& frame) override {
        frames.push_back(std::move(frame));
        return freighter::NIL;
    }
    
    std::vector<synnax::Frame> frames;
};

// TEST(SequenceTest, JSONSourceAndSetOperator) {
//     std::unordered_map<std::string, synnax::Channel> channels;
//     synnax::Channel ch_one;
//     ch_one.key = 1;
//     ch_one.name = "x";
//     ch_one.data_type = synnax::FLOAT64;
//     synnax::Channel ch_two;
//     channels["x"] = ch_one;
//     ch_two.key = 2;
//     ch_two.name = "y";
//     ch_two.data_type = synnax::FLOAT64;
//     channels["y"] = ch_two;
//
//     // Create sink and operator
//     auto sink = std::make_shared<TestSink>();
//     auto op = std::make_shared<ChannelSetOperator>(sink, channels);
//
//     // Create JSON source with test data
//     json source_data = {
//         {"x_values", 1.0},
//         {"y_values", 3.0},
//     };
//     auto source = std::make_shared<sequence::JSONSource>(source_data);
//
//     // Create Lua script that uses the JSON data and set operator
//     std::string script = R"(
//             set("x", x_values)
//             set("y", y_values)
//     )";
//
//     // Create and run sequence
//     auto [seq, err] = sequence::Sequence::create(op, source, script);
//     ASSERT_FALSE(err) << err.message();
//     ASSERT_NE(seq, nullptr);
//
//     err = seq->next();
//     ASSERT_FALSE(err) << err.message();
//
//     // Verify results
//     ASSERT_EQ(sink->frames.size(), 1);
//     // Add more specific assertions about frame contents here
//     // get the first frame
//     synnax::Frame frame = std::move(sink->frames[0]);
//     // assert that there are two values in the frame
//     ASSERT_EQ(frame.size(), 2);
//
// }
