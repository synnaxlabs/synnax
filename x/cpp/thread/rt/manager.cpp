// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>

#include "glog/logging.h"

#include "x/cpp/thread/rt/rt.h"

namespace x::thread::rt {

Handle::Handle(int core, Config resolved, std::function<void(int)> release_fn):
    core(core),
    resolved(std::move(resolved)),
    release_fn(std::move(release_fn)),
    released(false) {}

Handle::Handle(Handle &&other) noexcept:
    core(other.core),
    resolved(std::move(other.resolved)),
    release_fn(std::move(other.release_fn)),
    released(other.released) {
    other.released = true;
}

Handle &Handle::operator=(Handle &&other) noexcept {
    if (this != &other) {
        this->release();
        this->core = other.core;
        this->resolved = std::move(other.resolved);
        this->release_fn = std::move(other.release_fn);
        this->released = other.released;
        other.released = true;
    }
    return *this;
}

Handle::~Handle() {
    this->release();
}

void Handle::apply() {
    apply_config(this->resolved);
}

void Handle::release() {
    if (this->released || this->core == CPU_AFFINITY_NONE) return;
    this->released = true;
    if (this->release_fn) this->release_fn(this->core);
}

void Manager::release_core(int core) {
    std::lock_guard lock(this->mu);
    if (std::find(this->all_cores.begin(), this->all_cores.end(), core) ==
        this->all_cores.end())
        return;
    if (std::find(this->available.begin(), this->available.end(), core) !=
        this->available.end())
        return;
    this->available.push_back(core);
}

Manager::Manager(): all_cores(discover_rt_cores()), available(this->all_cores) {
    const auto total = this->all_cores.size();
    if (total > 0)
        LOG(INFO) << "[rt.manager] managing " << total << " RT cores";
    else
        VLOG(1) << "[rt.manager] no isolated cores found, "
                << "RT tasks will run without core pinning";
}

Handle Manager::allocate(Config cfg) {
    int core;
    {
        std::lock_guard lock(this->mu);
        if (this->available.empty()) {
            core = CPU_AFFINITY_NONE;
        } else {
            core = this->available.back();
            this->available.pop_back();
        }
    }
    cfg.cpu_affinity = core;
    const auto caps = capabilities();
    if (caps.mmcss) cfg.use_mmcss = true;
    if (caps.memory_locking) cfg.lock_memory = true;
    if (core != CPU_AFFINITY_NONE) VLOG(1) << "[rt.manager] allocated core " << core;
    auto weak = weak_from_this();
    return Handle(core, std::move(cfg), [weak](int c) {
        if (auto mgr = weak.lock()) mgr->release_core(c);
    });
}

size_t Manager::available_cores() const {
    std::lock_guard lock(this->mu);
    return this->available.size();
}

size_t Manager::total_cores() const {
    std::lock_guard lock(this->mu);
    return this->all_cores.size();
}
}
