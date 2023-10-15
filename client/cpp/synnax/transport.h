// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// freighter
#include "freighter/freighter.h"

/// internal
#include "synnax/framer/framer.h"
#include "synnax/channel/channel.h"
#include "synnax/ranger/ranger.h"
#include "synnax/auth/auth.h"

class Transport {
public:
    Transport(uint16_t port, const std::string &ip);

    void use(Freighter::Middleware *mw) const;

    Auth::LoginClient *auth_login;
    Framer::IteratorClient *frame_iter;
    Framer::StreamerClient *frame_stream;
    Framer::WriterClient *frame_write;
    Channel::CreateClient *chan_create;
    Channel::RetrieveClient *chan_retrieve;
    Ranger::RetrieveClient *range_retrieve;
    Ranger::CreateClient *range_create;
    Ranger::KVDeleteClient *range_kv_delete;
    Ranger::KVGetClient *range_kv_get;
    Ranger::KVSetClient *range_kv_set;
};
