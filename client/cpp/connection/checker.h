// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <atomic>
#include <condition_variable>
#include <functional>
#include <mutex>
#include <string>
#include <thread>
#include <vector>

#include "glog/logging.h"
#include "google/protobuf/empty.pb.h"

#include "freighter/cpp/freighter.h"
#include "x/cpp/telem/clock_skew.h"
#include "x/cpp/telem/telem.h"

#include "core/pkg/api/grpc/connectivity/connectivity.pb.h"

namespace synnax::connection {
enum class Status { DISCONNECTED, CONNECTING, CONNECTED, FAILED };

struct State {
    Status status = Status::DISCONNECTED;
    x::errors::Error error;
    std::string message = "Disconnected";
    std::string cluster_key;
    std::string client_version;
    bool client_server_compatible = false;
    std::string node_version;
    x::telem::TimeSpan clock_skew = x::telem::TimeSpan::ZERO();
    bool clock_skew_exceeded = false;
};

using CheckClient = freighter::
    UnaryClient<google::protobuf::Empty, grpc::connectivity::CheckResponse>;

bool versions_compatible(const std::string &v1, const std::string &v2);

class Checker {
    std::unique_ptr<CheckClient> client;
    x::telem::TimeSpan poll_freq;
    std::string client_version;
    std::string name;
    x::telem::TimeSpan clock_skew_threshold;
    x::telem::ClockSkewCalculator skew_calc;

    mutable std::mutex mu;
    State _state;
    std::vector<std::function<void(const State &)>> on_change_handlers;
    bool version_warned = false;

    std::thread poll_thread;
    std::atomic<bool> running{false};
    std::condition_variable cv;
    std::mutex cv_mu;

    void run();

    void fire_handlers(const State &s);

public:
    Checker(
        std::unique_ptr<CheckClient> client,
        x::telem::TimeSpan poll_freq,
        std::string client_version,
        std::string name = "",
        x::telem::TimeSpan clock_skew_threshold = x::telem::SECOND
    );

    ~Checker();

    void stop();

    State check();

    State state() const;

    void on_change(std::function<void(const State &)> handler);
};
}
