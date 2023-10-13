// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// internal
#include "synnax/framer/framer.h"
#include "synnax/ranger/ranger.h"
#include "synnax/channel/channel.h"
#include "synnax/transport.h"

using namespace Synnax;

struct Config {
    std::string host;
    std::uint16_t port;
    bool secure;
    std::string username;
    std::string password;
};

namespace Synnax {
    class Client {
    public:
        Channel::Client channels = Channel::Client(nullptr, nullptr);
        Ranger::Client ranges = Ranger::Client(nullptr, nullptr, nullptr, nullptr, nullptr);
        Framer::Client telem = Framer::Client(nullptr, nullptr, nullptr);

        explicit Client(const Config &cfg) {
            auto t = Transport(cfg.port, cfg.host);
            auth = Auth::Client(t.auth_login, cfg.username, cfg.password);
            t.use(auth.middleware());
            channels = Channel::Client(t.chan_retrieve, t.chan_create);
            ranges = Ranger::Client(
                    t.range_retrieve,
                    t.range_create,
                    t.range_kv_delete,
                    t.range_kv_get,
                    t.range_kv_set
            );
            telem = Framer::Client(t.frame_iter, t.frame_stream, t.frame_write);
        }


    private:
        Auth::Client auth = Auth::Client(nullptr, "", "");
    };
}