// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// internal
#include "synnax/framer/framer.h"
#include "synnax/channel/channel.h"
#include "synnax/ranger/ranger.h"
#include "synnax/auth/auth.h"

class Transport {
public:
    Transport(uint16_t port, const std::string &ip);

    void use(freighter::Middleware *mw) const;

    AuthLoginClient *auth_login;
    IteratorClient *frame_iter;
    StreamerClient *frame_stream;
    WriterClient *frame_write;
    ChannelCreateClient *chan_create;
    ChannelRetrieveClient *chan_retrieve;
    RangeRetrieveClient *range_retrieve;
    RangeCreateClient *range_create;
    RangeKVDeleteClient *range_kv_delete;
    RangeKVGetClient *range_kv_get;
    RangeKVSetClient *range_kv_set;
};
