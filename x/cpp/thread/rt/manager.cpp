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

class ManagerImpl {
    mutable std::mutex mu;
    std::vector<int> all_cores;
    std::vector<int> available;

public:
    explicit ManagerImpl(std::vector<int> cores): all_cores(cores), available(cores) {}

    int acquire() {
        std::lock_guard lock(this->mu);
        if (this->available.empty()) return CPU_AFFINITY_NONE;
        const int core = this->available.back();
        this->available.pop_back();
        return core;
    }

    void release(int core) {
        std::lock_guard lock(this->mu);
        if (std::find(this->all_cores.begin(), this->all_cores.end(), core) ==
            this->all_cores.end())
            return;
        if (std::find(this->available.begin(), this->available.end(), core) !=
            this->available.end())
            return;
        this->available.push_back(core);
    }

    size_t available_count() const {
        std::lock_guard lock(this->mu);
        return this->available.size();
    }

    size_t total_count() const {
        std::lock_guard lock(this->mu);
        return this->all_cores.size();
    }
};

Handle::Handle(int core, Config resolved, std::weak_ptr<ManagerImpl> impl):
    core(core), resolved(std::move(resolved)), impl(std::move(impl)), released(false) {}

Handle::Handle(Handle &&other) noexcept:
    core(other.core),
    resolved(std::move(other.resolved)),
    impl(std::move(other.impl)),
    released(other.released) {
    other.released = true;
}

Handle &Handle::operator=(Handle &&other) noexcept {
    if (this != &other) {
        this->release();
        this->core = other.core;
        this->resolved = std::move(other.resolved);
        this->impl = std::move(other.impl);
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
    if (auto mgr = this->impl.lock()) mgr->release(this->core);
}

Manager::Manager(): impl(std::make_shared<ManagerImpl>(discover_rt_cores())) {
    const auto total = this->impl->total_count();
    if (total > 0)
        LOG(INFO) << "[rt.manager] managing " << total << " RT cores";
    else
        VLOG(1) << "[rt.manager] no isolated cores found, "
                << "RT tasks will run without core pinning";
}

Handle Manager::allocate(Config cfg) {
    const int core = this->impl->acquire();
    cfg.cpu_affinity = core;
    const auto caps = get_capabilities();
    if (caps.mmcss) cfg.use_mmcss = true;
    if (caps.memory_locking) cfg.lock_memory = true;
    if (core != CPU_AFFINITY_NONE) VLOG(1) << "[rt.manager] allocated core " << core;
    return Handle(core, std::move(cfg), this->impl);
}

size_t Manager::available_cores() const {
    return this->impl->available_count();
}

size_t Manager::total_cores() const {
    return this->impl->total_count();
}
}
