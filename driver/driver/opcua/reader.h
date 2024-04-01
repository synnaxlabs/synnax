#pragma once

/// internal.
#include "opcua.h"
#include  "driver/driver/config/config.h"
#include "driver/driver/task/task.h"
#include "driver/driver/pipeline/acquisition.h"


namespace opcua {
struct ReaderChannelConfig {
    /// @brief the namespace index of the node.
    std::uint32_t ns;
    /// @brief the node id.
    std::string node;
    /// @brief the corresponding channel key to write the variable for the node from.
    ChannelKey key;
    /// @brief the channel fetched from the Synnax server. This does not need to
    /// be provided via the JSON configuration.
    Channel ch;

    ReaderChannelConfig() = default;

    explicit ReaderChannelConfig(
        config::Parser& parser
    ): ns(parser.required<std::uint32_t>("ns")),
       node(parser.required<std::string>("node")),
       key(parser.required<ChannelKey>("key")) {
    }
};

struct ReaderConfig {
    ConnectionConfig connection;
    /// @brief sets the acquisition rate.
    Rate rate;


    /// @brief the list of channels to read from the server.
    std::vector<ReaderChannelConfig> channels;

    ReaderConfig() = default;

    explicit ReaderConfig(config::Parser& parser);

    std::vector<ChannelKey> channelKeys() const {
        auto keys = std::vector<ChannelKey>(channels.size());
        for (std::size_t i = 0; i < channels.size(); i++) keys[i] = channels[i].key;
        return keys;
    }
};

/// @brief a task that reads values from an OPC UA server.
class Reader final : public task::Task {
public:
    explicit Reader(const std::shared_ptr<task::Context> &ctx, synnax::Task task);

    void exec(task::Command& cmd) override {};

    void stop() override {
        pipe.stop();
    }

private:
    std::shared_ptr<task::Context> ctx;
    ReaderConfig cfg;

    breaker::Breaker breaker;

    pipeline::Acquisition pipe;

    std::pair<std::pair<std::vector<ChannelKey>, std::set<ChannelKey>>, freighter::Error>
    retrieveAdditionalChannelInfo() {
        auto channelKeys = cfg.channelKeys();
        auto indexes = std::set<ChannelKey>();
        auto [channels, c_err] = ctx->client->channels.retrieve(cfg.channelKeys());
        if (c_err) {
            if (c_err.matches(freighter::UNREACHABLE) && breaker.wait(c_err.message()))
                return retrieveAdditionalChannelInfo();
            return {{channelKeys, indexes}, c_err};
        }
        for (auto i = 0; i < channels.size(); i++) {
            const auto ch = channels[i];
            if (std::count(channelKeys.begin(), channelKeys.end(), ch.index) == 0) {
                channelKeys.push_back(ch.index);
                indexes.insert(ch.index);
            }
            cfg.channels[i].ch = ch;
        }
        return {{channelKeys, indexes}, freighter::Error()};
    }
};
}
