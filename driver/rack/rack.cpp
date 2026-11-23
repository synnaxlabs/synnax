// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "absl/log/log.h"

#include "client/cpp/errors/errors.h"
#include "x/cpp/defer/defer.h"
#include "x/cpp/thread/rt/rt.h"
#include "x/cpp/thread/thread.h"

#include "driver/journal/journal.h"
#include "driver/outbox/outbox.h"
#include "driver/rack/state.h"
#include "rack.h"

namespace driver::rack {
namespace {
/// @returns the text of err as a sentence. Some error texts end in a full stop.
std::string sentence(const x::errors::Error &err) {
    return err.data.ends_with('.') ? err.data : err.data + ".";
}
}

x::errors::Error
outbox_failure(const std::filesystem::path &dir, const x::errors::Error &err) {
    const auto path = dir.string();
    std::string fix;
    if (err.matches(outbox::UNKNOWN_VERSION) || err.matches(journal::UNKNOWN_VERSION) ||
        err.matches(journal::UNKNOWN_TYPE) || err.matches(journal::UNKNOWN_FLAG))
        fix = "A newer Driver wrote it. Update the Driver, or move " + path +
              " aside to discard the changes it holds for the Core.";
    else if (
        err.matches(outbox::CORRUPT) || err.matches(journal::CORRUPT) ||
        err.matches(journal::UNKNOWN_MAGIC)
    )
        fix = "Its files are damaged. Move " + path +
              " aside to discard the changes it holds for the Core.";
    else if (err.matches(x::fs::LOCKED))
        fix = "Another Driver uses it. Stop the other Driver, or give each Driver its "
              "own state directory.";
    else
        fix = "Make sure that the Driver can write to " + path +
              " and that the disk has free space. On Linux, run 'sudo synnax-driver "
              "install' to update the service.";
    return {
        err,
        "The Driver cannot open its outbox in " + path + ": " + sentence(err) + " " +
            fix + " Then restart the Driver."
    };
}

status::Health outbox_stuck(
    const std::filesystem::path &dir,
    const x::telem::TimeSpan &stuck_after,
    const x::errors::Error &err
) {
    return {
        .variant = synnax::status::VARIANT_WARNING,
        .message = OUTBOX_STUCK_MSG,
        .description = "The Core has not taken an Arc status set for " +
                       stuck_after.to_string() + ": " + sentence(err) +
                       " Check that the Driver can reach the Core and that the Core "
                       "is running. The Driver keeps the sets in " +
                       dir.string() + " and sends them once the Core takes them.",
    };
}

bool Rack::should_exit(
    const x::errors::Error &err,
    const std::function<void()> &on_shutdown,
    const bool reconnecting
) {
    if (!err) return false;
    if (reconnecting && synnax::errors::is_unreachable(err)) {
        if (this->breaker.wait_unlimited(err)) return false;
    } else if (
        synnax::errors::is_temporarily_unavailable(err) && this->breaker.wait(err)
    )
        return false;
    // A stop came first, so the error did not stop the rack.
    if (!this->breaker.running()) return true;
    this->run_err = err;
    if (on_shutdown) on_shutdown();
    return true;
}

Rack::~Rack() {
    this->stop();
    // A detached worker still uses the task manager, so it lives until the process
    // exits.
    if (this->task_manager != nullptr && this->task_manager->worker_detached()) {
        [[maybe_unused]] const auto *const leaked = this->task_manager.release();
    }
}

void Rack::run(x::args::Parser &args, const std::function<void()> &on_shutdown) {
    x::thread::set_name("rack");
    LOG(INFO) << x::thread::rt::capabilities();
    Config cfg;
    while (true) {
        auto [loaded, err] = Config::load(args, this->breaker);
        if (!err) {
            cfg = std::move(loaded);
            break;
        }
        if (this->should_exit(err, on_shutdown)) return;
    }
    VLOG(1) << "loaded config. starting task manager";
    const auto rt_manager = std::make_shared<x::thread::rt::Manager>();
    const auto client = cfg.new_client();
    const auto data = cfg.state_dir / DATA_DIR;
    auto outbox_cfg = cfg.outbox;
    outbox_cfg.dir = data / OUTBOX_DIR;
    outbox_cfg.statuses = client->statuses;
    outbox_cfg.ranges = client->ranges;
    outbox_cfg.on_stuck =
        [alert = cfg.outbox_alert,
         dir = outbox_cfg.dir,
         after = outbox_cfg.stuck_after](const x::errors::Error &err) {
            if (!err) {
                alert->clear();
                return;
            }
            auto health = outbox_stuck(dir, after, err);
            LOG(WARNING) << health.message << ". " << health.description;
            alert->raise(std::move(health));
        };
    // The outbox serves only Arc status sets and range calls, so the rack runs
    // without it.
    auto [outbox, outbox_err] =
        [&]() -> std::pair<std::unique_ptr<outbox::Outbox>, x::errors::Error> {
        std::error_code ec;
        std::filesystem::create_directories(data, ec);
        if (ec)
            return {
                nullptr,
                x::errors::Error(
                    STATE_ERROR,
                    "failed to create " + data.string() + ": " + ec.message()
                )
            };
        auto [hook, hook_err] = state::owner_hook(data);
        if (hook_err) return {nullptr, hook_err};
        cfg.on_create = hook;
        outbox_cfg.on_create = std::move(hook);
        return outbox::Outbox::open(outbox_cfg);
    }();
    if (outbox_err) {
        cfg.outbox_err = outbox_failure(outbox_cfg.dir, outbox_err);
        LOG(ERROR) << OUTBOX_FAILURE_MSG << ". " << cfg.outbox_err.data;
    }
    {
        std::lock_guard lock{this->mu};
        // A stop that came first sees no task manager, so none starts.
        if (!this->breaker.running()) return;
        // One task manager serves every run, so a worker that it detached never
        // outlives it. A Core outage only reopens its streamer.
        this->task_manager = std::make_unique<task::Manager>(
            cfg.rack,
            client,
            cfg.new_factory(client, rt_manager),
            std::pair<std::shared_ptr<outbox::Outbox>, x::errors::Error>{
                std::move(outbox),
                cfg.outbox_err
            },
            cfg.manager
        );
    }
    // A run that lost the Core leaves its tasks running for the next run.
    x::defer::defer close([this] { this->task_manager->close(); });
    while (true) {
        const auto err = this->task_manager->run([this]() { this->breaker.reset(); });
        if (this->should_exit(err, on_shutdown, true) || !this->breaker.running())
            return;
        // The factories hold the Core version and cluster from startup, so a new one
        // needs a new Driver process. No run starts before a check passes.
        while (const auto check_err = cfg.check_core()) {
            if (check_err.matches(CORE_CHANGED))
                LOG(ERROR) << "Core changed cluster or version, restarting: "
                           << check_err.data;
            if (this->should_exit(check_err, on_shutdown, true)) return;
        }
    }
}

void Rack::start(x::args::Parser &args, std::function<void()> on_shutdown) {
    this->breaker.start();
    this->run_thread = std::thread(
        [this, args, callback = std::move(on_shutdown)]() mutable {
            this->run(args, callback);
        }
    );
}

x::errors::Error Rack::stop() {
    if (!this->breaker.stop()) return x::errors::NIL;
    {
        std::lock_guard lock{this->mu};
        if (this->task_manager != nullptr) this->task_manager->stop();
    }
    this->run_thread.join();
    return this->run_err;
}
}
