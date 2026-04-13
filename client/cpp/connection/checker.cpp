// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>
#include <sstream>

#include "client/cpp/connection/checker.h"

namespace synnax::connection {
const std::string TROUBLESHOOTING_URL =
    "https://docs.synnaxlabs.com/reference/client/resources/troubleshooting";

std::pair<int, int> parse_version(const std::string &v) {
    std::istringstream ss(v);
    int major = 0, minor = 0;
    char dot;
    ss >> major >> dot >> minor;
    return {major, minor};
}

bool versions_compatible(const std::string &v1, const std::string &v2) {
    return parse_version(v1) == parse_version(v2);
}

bool client_is_newer(const std::string &client_ver, const std::string &node_ver) {
    return parse_version(client_ver) > parse_version(node_ver);
}

std::string create_version_warning(
    const std::string &node_version,
    const std::string &client_version,
    const bool is_client_newer
) {
    const auto to_upgrade = is_client_newer ? "Core" : "client";
    const auto age = is_client_newer ? "old" : "new";
    std::ostringstream ss;
    ss << "The Synnax core version ";
    if (!node_version.empty()) ss << node_version << " ";
    ss << "is too " << age << " for client version " << client_version
       << ". This may cause compatibility issues. We recommend updating the "
       << to_upgrade << ". For more information, see " << TROUBLESHOOTING_URL << "#old-"
       << to_upgrade << "-version";
    return ss.str();
}

Checker::Checker(
    std::unique_ptr<CheckClient> client,
    const x::telem::TimeSpan poll_freq,
    std::string client_version,
    std::string name,
    const x::telem::TimeSpan clock_skew_threshold
):
    client(std::move(client)),
    poll_freq(poll_freq),
    client_version(std::move(client_version)),
    name(std::move(name)),
    clock_skew_threshold(clock_skew_threshold) {
    this->_state.client_version = this->client_version;
    this->running = true;
    this->check();
    this->poll_thread = std::thread([this] { this->run(); });
}

Checker::~Checker() {
    this->stop();
}

void Checker::stop() {
    {
        std::lock_guard lk(this->cv_mu);
        this->running = false;
    }
    this->cv.notify_all();
    if (this->poll_thread.joinable()) this->poll_thread.join();
}

State Checker::check() {
    Status prev_status;
    bool prev_skew_exceeded;
    {
        std::lock_guard lk(this->mu);
        prev_status = this->_state.status;
        prev_skew_exceeded = this->_state.clock_skew_exceeded;
    }

    this->skew_calc.start();
    google::protobuf::Empty req;
    auto [res, err] = this->client->send("/connectivity/check", req);

    State snapshot;
    {
        std::lock_guard lk(this->mu);
        if (err) {
            this->_state.status = Status::FAILED;
            this->_state.error = err;
            this->_state.message = err.message();
        } else {
            this->skew_calc.end(x::telem::TimeStamp(res.node_time()));
            this->_state.clock_skew = this->skew_calc.skew();
            this->_state.clock_skew_exceeded = this->skew_calc.exceeds(
                this->clock_skew_threshold
            );
            if (this->_state.clock_skew_exceeded) {
                auto direction = this->skew_calc.skew() > x::telem::TimeSpan::ZERO()
                                   ? "ahead of"
                                   : "behind";
                LOG(WARNING) << "Measured excessive clock skew between this host and "
                                "the Synnax core. This host is "
                             << direction << " the Synnax core by approximately "
                             << this->skew_calc.skew().abs() << ".";
            }
            const auto &nv = res.node_version();
            if (nv.empty()) {
                this->_state.client_server_compatible = false;
                if (!this->version_warned) {
                    LOG(WARNING)
                        << create_version_warning("", this->client_version, true);
                    this->version_warned = true;
                }
            } else if (!versions_compatible(this->client_version, nv)) {
                this->_state.client_server_compatible = false;
                if (!this->version_warned) {
                    LOG(WARNING) << create_version_warning(
                        nv,
                        this->client_version,
                        client_is_newer(this->client_version, nv)
                    );
                    this->version_warned = true;
                }
            } else {
                this->_state.client_server_compatible = true;
            }
            this->_state.status = Status::CONNECTED;
            this->_state.message = "Connected to " +
                                   (this->name.empty() ? "cluster" : this->name);
            this->_state.cluster_key = res.cluster_key();
            this->_state.node_version = nv;
        }
        snapshot = this->_state;
    }

    const bool changed = prev_status != snapshot.status ||
                         prev_skew_exceeded != snapshot.clock_skew_exceeded;
    if (changed) this->fire_handlers(snapshot);
    return snapshot;
}

State Checker::state() const {
    std::lock_guard lk(this->mu);
    return this->_state;
}

void Checker::on_change(std::function<void(const State &)> handler) {
    this->on_change_handlers.push_back(std::move(handler));
}

void Checker::run() {
    while (true) {
        std::unique_lock lk(this->cv_mu);
        const auto nanos = this->poll_freq.nanoseconds();
        if (this->cv.wait_for(lk, std::chrono::nanoseconds(nanos), [this] {
                return !this->running.load();
            }))
            return;
        lk.unlock();
        this->check();
    }
}

void Checker::fire_handlers(const State &s) {
    for (const auto &handler: this->on_change_handlers)
        handler(s);
}
}
