// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.
// File added by Elham Islam

#include <driver/opc/writer.h>

using namespace opc;

WriterChannelConfig::WriterChannelConfig(
    config::Parser &parser
) : node_id(parser.required<std::string>("node_id")),
   node(parseNodeId("node_id", parser)),
   channel(parser.required<ChannelKey>("channel")),
    enabled(parser.optional<bool>("enabled", true)) {
}

WriterConfig::WriterConfig(
    config::Parser &parser
) : device(parser.required<std::string>("device")),
    update_rate(parser.required<std::float_t>("update_rate")),
    channels(parser.required<std::vector<WriterChannelConfig>>("channels")) {

    parser.iter("channels", [&](config::Parser &channel_builder) {
        auto channel = WriterChannelConfig(channel_builder);
        if (channel.enabled) channels.push_back(channel);
    }
}

WriterSink::WriterSink(){

}

