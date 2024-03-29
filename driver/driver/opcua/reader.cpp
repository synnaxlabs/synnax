/// std.
#include <set>

/// external.
#include "open62541/client_highlevel.h"
#include "glog/logging.h"

/// internal.
#include "driver/driver/opcua/reader.h"

#include "driver/driver/opcua/util.h"
#include "driver/driver/config/config.h"
#include "open62541/client_config_default.h"
#include "open62541/client_subscriptions.h"
#include "open62541/types.h"
#include "open62541/plugin/log_stdout.h"


using namespace opcua;

ReaderConfig::ReaderConfig(config::Parser& parser): connection(
    parser.child("connection")) {
    rate = Rate(parser.required<std::float_t>("rate"));
    parser.iter("channels", [&](config::Parser& channel_builder) {
        channels.push_back(ReaderChannelConfig(channel_builder));
    });
}

class ReaderSource final : public pipeline::Source {
public:
    ReaderConfig cfg;
    UA_Client* client;
    std::set<ChannelKey> indexes;

    ReaderSource(
        ReaderConfig cfg,
        UA_Client* client,
        std::set<synnax::ChannelKey> indexes
    )
        : cfg(std::move(cfg)), client(client), indexes(std::move(indexes)) {
    }

    std::pair<Frame, freighter::Error> read() override {
        auto fr = Frame(cfg.channels.size() + indexes.size());
        for (auto i = 0; i < cfg.channels.size(); i++) {
            auto ch = cfg.channels[i];
            UA_NodeId node_id = UA_NODEID_STRING_ALLOC(ch.ns, ch.node.c_str());
            UA_Variant* value = UA_Variant_new();
            UA_StatusCode status = UA_Client_readValueAttribute(client, node_id, value);
            if (status != UA_STATUSCODE_GOOD) {
                LOG(ERROR) << "Unable to read value from OPCUA server";
            } else {
                auto val = val_to_series(value, ch.ch.data_type);
                fr.add(ch.key, val);
                UA_Variant_delete(value);
                // UA_NodeId_clear(&node_id);
            }
        }
        auto now = synnax::TimeStamp::now();
        for (const auto& idx: indexes)
            fr.add(idx, Series(now));
        return std::make_pair(std::move(fr), freighter::NIL);
    }
};


Reader::Reader(std::shared_ptr<task::Context> ctx, synnax::Task task): ctx(ctx) {
    // Step 1. Parse the configuration to ensure that it is valid.
    auto parser = config::Parser(task.config);
    cfg = ReaderConfig(parser);
    if (!parser.ok()) {
        LOG(ERROR) << "failed to parse configuration for " << task.name;
        ctx->setState({
            .task = task.key,
            .type = "failed",
            .details = parser.error_json(),
        });
        return;
    }

    LOG(INFO) << "successfully parsed configuration for " << task.name;

    auto breaker_config = breaker::Config{
        .name = task.name,
        .base_interval = 1 * SECOND,
        .scale = 1.2,
        .max_retries = 20,
    };
    breaker = breaker::Breaker(breaker_config);

    // Fetch additional index channels we also need as part of the configuration.
    auto [res, err] = retrieveAdditionalChannelInfo();
    if (err) {
        ctx->setState({
            .task = task.key,
            .type = "failed",
            .details = json{
                {"message", err.message()}
            }
        });
        return;
    }
    auto [channelKeys, indexes] = res;

    // Connect to the OPC UA server.
    auto [ua_client, ok] = opcua::connect(cfg.connection, task, ctx);
    if (!ok) return;

    for (auto i = 0; i < cfg.channels.size(); i++) {
        auto ch = cfg.channels[i];
        UA_Variant* value = UA_Variant_new();
        UA_NodeId myIntegerNodeID = UA_NODEID_STRING_ALLOC(1, ch.node.c_str());
        UA_StatusCode status = UA_Client_readValueAttribute(
            ua_client, myIntegerNodeID, value);
        if (status != UA_STATUSCODE_GOOD) {
            if (status == UA_STATUSCODE_BADNODEIDUNKNOWN) {
                parser.field_err("channels." + std::to_string(i),
                                 "opcua node not found");
            } else {
                parser.field_err("channels." + std::to_string(i),
                                 "failed to read value" + std::string(
                                     UA_StatusCode_name(status)));
            }
            LOG(ERROR) << "failed to read value for channel " << ch.node;
        }
        UA_Variant_delete(value);
        UA_NodeId_clear(&myIntegerNodeID);
    }

    if (!parser.ok()) {
        ctx->setState({
            .task = task.key,
            .type = "failed",
            .details = parser.error_json(),
        });
        return;
    }

    auto source = std::make_unique<ReaderSource>(
        cfg,
        ua_client,
        indexes
    );

    auto writer_cfg = synnax::WriterConfig{
        .channels = channelKeys,
        .start = TimeStamp::now(),
        .mode = synnax::WriterStreamOnly
    };

    pipe = pipeline::Acquisition(
        ctx,
        writer_cfg,
        std::move(source),
        breaker_config
    );
    pipe.start();
}
