// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <array>
#include <cmath>
#include <format>
#include <iterator>
#include <numeric>
#include <set>
#include <span>
#include <tuple>
#include <utility>

#include "absl/log/log.h"

#include "x/cpp/defer/defer.h"
#include "x/cpp/loop/loop.h"

#include "driver/ethercat/engine/clock.h"
#include "driver/ethercat/engine/engine.h"
#include "driver/ethercat/engine/phase_lock.h"
#include "driver/ethercat/errors/errors.h"
#include "driver/ethercat/telem/telem.h"

namespace driver::ethercat::engine {
namespace {
/// @brief the slowest rate the bus cycles at, well inside the 100 ms sync manager
/// watchdog most slaves default to.
const x::telem::Rate MIN_CYCLE_RATE(100);
/// @brief the fastest rate the bus cycles at.
const x::telem::Rate MAX_CYCLE_RATE(10000);
/// @brief how often the supervisor reads slave states.
const auto SUPERVISOR_INTERVAL = x::telem::MILLISECOND * 100;
/// @brief the longest wait for the frames in flight when the bus stops.
const auto LAST_FRAME_TIMEOUT = x::telem::MILLISECOND * 10;
/// @brief the wait between receives of a frame in flight when the bus stops.
const auto LAST_FRAME_POLL = x::telem::MICROSECOND * 100;
/// @brief how often a call that needs the stopped bus tries mu again.
const auto LOCK_POLL_INTERVAL = x::telem::MILLISECOND * 10;
/// @brief how long each step of a start may run before the task warns.
const auto WAIT_STEP_LIMIT = x::telem::SECOND * 10;
const auto SCAN_STEP_LIMIT = x::telem::SECOND * 10;
const auto ACTIVATE_STEP_LIMIT = x::telem::SECOND * 30;
const auto RESET_STEP_LIMIT = x::telem::SECOND * 10;
/// @brief how long a retry step may run past its wait.
const auto RETRY_STEP_SLACK = x::telem::SECOND * 10;

/// @brief sends step to the report of start, if it has one.
void report(const Engine::Start &start, const Engine::Step &step) {
    if (start.report) start.report(step);
}

/// @brief returns the error of an open whose cancel stopped before the bus started. The
/// task answers a start that ends with it as stopped.
x::errors::Error start_canceled_error(const std::string &network) {
    return x::errors::Error(
        x::errors::CANCELLED,
        "The Driver stopped the start on " + network + "."
    );
}

/// @brief returns the error of a start whose deadline passed before its first try.
x::errors::Error start_late_error(const std::string &network) {
    return x::errors::Error(
        errors::ACTIVATION_ERROR,
        "The start on " + network + " ran out of time."
    );
}

/// @brief returns the error of a start whose deadline passed while or after it waited
/// for another task.
x::errors::Error start_blocked_error(const std::string &network) {
    return x::errors::Error(
        errors::ACTIVATION_ERROR,
        "Another task is still using " + network + "."
    );
}
/// @brief the length of the sample buffer of each reader, in seconds of samples.
constexpr double BUFFER_SECONDS = 2;
constexpr size_t MIN_BUFFER_SAMPLES = 16;
/// @brief AL status codes of a slave that kept the layout of an earlier mode or master.
/// The slave takes its new layout when the next start resets it.
constexpr std::array<uint16_t, 2> STALE_LAYOUT_CODES{0x001D, 0x001E};
/// @brief the AL status code of "Invalid DC SYNC configuration".
constexpr uint16_t INVALID_DC_SYNC = 0x0030;

uint64_t millihertz(const x::telem::Rate rate) {
    return static_cast<uint64_t>(std::llround(rate.hz() * 1000));
}

/// @brief returns the slowest bus rate in millihertz that is a multiple of every read
/// rate, at least every write rate, and at least MIN_CYCLE_RATE.
std::pair<uint64_t, x::errors::Error> bus_rate(
    const std::vector<x::telem::Rate> &reads,
    const std::vector<x::telem::Rate> &writes,
    const std::string &network
) {
    const uint64_t max = millihertz(MAX_CYCLE_RATE);
    uint64_t floor = millihertz(MIN_CYCLE_RATE);
    for (const auto &rate: writes)
        floor = std::max(floor, millihertz(rate));
    if (floor > max)
        return {
            0,
            x::errors::Error(
                errors::RATE_MISMATCH,
                std::format(
                    "A write task on {} runs at {} Hz, above the maximum bus rate of "
                    "{} Hz.",
                    network,
                    static_cast<double>(floor) / 1000,
                    MAX_CYCLE_RATE.hz()
                )
            )
        };
    uint64_t base = 1;
    for (const auto &rate: reads) {
        const auto mhz = millihertz(rate);
        // Allow the rounding error of the double a Rate holds.
        const double error = std::abs(rate.hz() * 1000 - static_cast<double>(mhz));
        if (mhz == 0 || error > static_cast<double>(mhz) * 1e-6)
            return {
                0,
                x::errors::Error(
                    errors::RATE_MISMATCH,
                    std::format(
                        "A read task on {} samples at {} Hz, but sample rates must be "
                        "a positive multiple of 0.001 Hz.",
                        network,
                        rate.hz()
                    )
                )
            };
        base = std::lcm(base, mhz);
        if (base > max) break;
    }
    const uint64_t rate = (floor + base - 1) / base * base;
    if (rate <= max) return {rate, {}};
    std::string rates;
    for (const auto &r: reads)
        rates += (rates.empty() ? "" : ", ") + std::format("{} Hz", r.hz());
    return {
        0,
        x::errors::Error(
            errors::RATE_MISMATCH,
            std::format(
                "No bus rate at or below {} Hz on {} is a multiple of every read task "
                "sample rate ({}). Choose sample rates with a smaller common multiple.",
                MAX_CYCLE_RATE.hz(),
                network,
                rates
            )
        )
    };
}

/// @brief true when a bus rate of mhz millihertz is a multiple of every read rate and
/// at least every write rate.
bool serves(
    const uint64_t mhz,
    const std::vector<x::telem::Rate> &reads,
    const std::vector<x::telem::Rate> &writes
) {
    return std::ranges::all_of(
               reads,
               [mhz](const x::telem::Rate r) { return mhz % millihertz(r) == 0; }
           ) &&
           std::ranges::all_of(writes, [mhz](const x::telem::Rate w) {
               return millihertz(w) <= mhz;
           });
}

/// @brief tells the user to switch a slave to Synnax, or to restart its power after a
/// switch, when it refused its sync configuration while it held the mode or the layout
/// of another program at the scan. op_mode is the sync mode that the user picked.
void advise_switch(
    const slave::Properties &props,
    const std::optional<dc::Choice> &op_mode,
    slave::Refusal &refusal
) {
    if (refusal.code != INVALID_DC_SYNC || !props.settings) return;
    if (props.switched()) {
        refusal.advice = "Restart the power of the device.";
        return;
    }
    const auto program = props.shared_with();
    if (program.empty()) return;
    refusal.advice = std::format(
        "{}The device is set up for {}. In Configure, switch the device to Synnax.",
        op_mode ? slave::DC_SYNC_ADVICE + " " : "",
        program
    );
}

/// @brief returns the error for an entry on a slave that the bus started without.
x::errors::Error left_out_error(
    const uint16_t position,
    const std::string &name,
    const slave::Refusal &refusal
) {
    auto msg = std::format(
        "Module {} ({}) is left out of the network: it {}. Fix the device, then stop "
        "every task on the network to try it again.",
        position,
        name,
        refusal.reason
    );
    if (!refusal.advice.empty()) msg.append(" ").append(refusal.advice);
    return x::errors::Error(errors::STATE_CHANGE_REFUSED, msg);
}

/// @brief returns a sentence for each slave of bus that another program sets up and
/// that has startup parameters in counts, keyed by position.
std::string unapplied(
    const std::vector<slave::DiscoveryResult> &bus,
    const std::map<uint16_t, size_t> &counts
) {
    std::string text;
    for (const auto &s: bus) {
        const auto &props = s.properties;
        const auto it = counts.find(props.position);
        if (it == counts.end() || it->second == 0 || !props.shared()) continue;
        const bool one = it->second == 1;
        if (!text.empty()) text += ' ';
        text += std::format(
            "Module {} ({}) is set up for {}, so Synnax did not apply its {} startup "
            "setting{}. Remove {}, or switch the device to Synnax in Configure.",
            props.position,
            props.name,
            props.shared_with(),
            it->second,
            one ? "" : "s",
            one ? "it" : "them"
        );
    }
    return text;
}
}

x::errors::Error Engine::start_error(
    x::errors::Error err,
    const std::map<uint16_t, slave::Refusal> &refused,
    const std::map<uint16_t, LeftOut> &left_out
) {
    err.data = errors::sentence(std::move(err.data));
    std::vector<std::string_view> advice;
    const auto add_advice = [&advice](const slave::Refusal &refusal) {
        if (!refusal.advice.empty() &&
            std::ranges::find(advice, refusal.advice) == advice.end())
            advice.push_back(refusal.advice);
    };
    for (const auto &[_, refusal]: refused)
        add_advice(refusal);
    for (const auto &[pos, left]: left_out) {
        err.data += " The Driver left out module " + std::to_string(pos) +
                    " because it " + left.refusal.reason + ".";
        add_advice(left.refusal);
    }
    for (const auto a: advice)
        err.data.append(" ").append(a);
    return err;
}

Engine::Input::Input(
    const std::vector<pdo::Entry> &entries,
    const std::vector<pdo::Offset> &offsets,
    const std::vector<slave::DiscoveryResult> &bus,
    const x::telem::Rate rate,
    const size_t samples_per_read
):
    rate(rate), samples_per_read(samples_per_read) {
    const auto slaves = slaves_of(entries, bus);
    this->slaves = std::vector<Slave>(slaves.size());
    for (size_t i = 0; i < slaves.size(); ++i)
        std::tie(this->slaves[i].position, this->slaves[i].name) = slaves[i];
    this->entries.reserve(entries.size());
    for (size_t i = 0; i < entries.size(); ++i) {
        const auto &entry = entries[i];
        const auto &offset = offsets[i];
        const auto length = telem::pdo_required_bytes(offset.bit, entry.bit_length);
        this->entries.push_back({
            .image_byte = offset.byte,
            .sample_offset = this->sample_size,
            .length = length,
            .decoder = telem::Decoder(offset.bit, entry.bit_length, entry.data_type),
        });
        this->sample_size += length;
    }
    this->capacity = std::max(
        {this->samples_per_read * 4,
         static_cast<size_t>(std::ceil(this->rate.hz() * BUFFER_SECONDS)),
         MIN_BUFFER_SAMPLES}
    );
    this->samples = std::make_unique<uint8_t[]>(this->capacity * this->sample_size);
    this->times = std::make_unique<x::telem::TimeStamp[]>(this->capacity);
}

void Engine::Input::push(const uint8_t *image, const x::telem::TimeStamp time) {
    const uint64_t h = this->head.load(std::memory_order_relaxed);
    if (h - this->tail.load(std::memory_order_acquire) >= this->capacity) {
        this->overflowed_samples.fetch_add(1, std::memory_order_relaxed);
        return;
    }
    const size_t slot = h % this->capacity;
    uint8_t *sample = this->samples.get() + slot * this->sample_size;
    for (const auto &entry: this->entries)
        std::memcpy(
            sample + entry.sample_offset,
            image + entry.image_byte,
            entry.length
        );
    this->times[slot] = time;
    // Pairs with the reader's store of waiting, so one of the two sees the other.
    this->head.store(h + 1, std::memory_order_seq_cst);
    if (this->waiting.load(std::memory_order_seq_cst) &&
        h + 1 - this->tail.load(std::memory_order_acquire) >= this->samples_per_read) {
        std::lock_guard lock(this->mu);
        this->cv.notify_one();
    }
}

void Engine::Output::update(const Outage &out) {
    bool any = false;
    for (auto &slave: this->slaves) {
        const bool now = std::ranges::binary_search(out.positions, slave.position);
        const bool silent = std::ranges::binary_search(out.silent, slave.position);
        const bool was = slave.out.exchange(now, std::memory_order_relaxed);
        if (slave.silent.exchange(silent, std::memory_order_relaxed) == silent &&
            was == now)
            continue;
        if (was && !now) slave.returned.store(true, std::memory_order_relaxed);
        any = true;
    }
    if (any && this->changed) this->changed();
}

std::vector<std::pair<uint16_t, std::string>> Engine::slaves_of(
    const std::vector<pdo::Entry> &entries,
    const std::vector<slave::DiscoveryResult> &bus
) {
    std::vector<std::pair<uint16_t, std::string>> slaves;
    slaves.reserve(entries.size());
    for (const auto &entry: entries)
        slaves.emplace_back(entry.slave_position, "");
    std::ranges::sort(slaves);
    const auto [last, end] = std::ranges::unique(slaves);
    slaves.erase(last, end);
    for (auto &[position, name]: slaves) {
        const auto it = std::ranges::find(bus, position, [](const auto &s) {
            return s.properties.position;
        });
        if (it != bus.end()) name = it->properties.name;
    }
    return slaves;
}

std::string
Engine::out_of_op(const uint16_t position, const std::string &name, const bool silent) {
    if (silent) return std::format("Lost contact with module {} ({}).", position, name);
    return std::format("Module {} ({}) is not in OP.", position, name);
}

bool Engine::Input::lose(
    const std::vector<uint16_t> &positions,
    const std::vector<uint16_t> &silent
) {
    bool any = false;
    for (auto &slave: this->slaves)
        if (std::ranges::binary_search(positions, slave.position)) {
            slave.silent.store(
                std::ranges::binary_search(silent, slave.position),
                std::memory_order_relaxed
            );
            slave.lost.store(true, std::memory_order_release);
            any = true;
        }
    return any;
}

void Engine::run() {
    std::shared_ptr<const Schedule> sched;
    uint64_t version = 0;
    Outage outage;
    outage.positions.reserve(this->image.slaves.size());
    outage.silent.reserve(this->image.slaves.size());
    outage.inputless.reserve(this->image.slaves.size());
    // The slaves that a cycle drops, when the frame adds inputless ones to the outage.
    std::vector<uint16_t> cycle_dropped;
    cycle_dropped.reserve(this->image.slaves.size());
    uint64_t outage_version = 0;
    // How far the time of the latest full cycle ran ahead of its send. A short cycle
    // keeps it, since its DC time can be stale.
    x::telem::TimeSpan lead;
    // Whether a sample time came from the host clock since the latest full cycle.
    bool host_timed = false;
    x::loop::Timer timer;
    std::optional<PhaseLock> phase;
    uint64_t late_frames = 0;
    uint64_t receive_failures = 0;
    // The cycle_err that this thread stored last.
    x::errors::Error stored_err;
    uint64_t send_failures = 0;
    // The send time of each frame in flight, oldest first.
    std::vector<x::telem::TimeStamp> in_flight;
    in_flight.reserve(master::MAX_FRAMES_IN_FLIGHT + 1);
    // Whether a frame is left to take before the next wait.
    bool taking = false;
    // The newest DC time of the frames taken since the last wait. The timer shifts
    // once a wait, since an older frame measured a phase that this shift corrects too.
    std::optional<x::telem::TimeStamp> wake_dc;
    Clock clock;
    // The working counter falls short until the slaves reach OP.
    const auto shortfall = [this](const x::errors::Error &err) {
        return this->starting.load(std::memory_order_acquire) &&
               err.matches(errors::WORKING_COUNTER_ERROR);
    };
    // How much further than the outage the slaves that a cycle found with no inputs can
    // keep the working counter from a full cycle.
    const auto inputless_shortfall =
        [this, &outage](const std::span<const uint16_t> inputless) {
            uint32_t extra = 0;
            for (const auto pos: inputless) {
                const auto share = this->image.working_counter_shares.find(pos);
                if (share == this->image.working_counter_shares.end() ||
                    std::ranges::binary_search(outage.inputless, pos))
                    continue;
                extra += share->second.inputs;
                if (!std::ranges::binary_search(outage.positions, pos))
                    extra += share->second.outputs;
            }
            return extra;
        };

    const auto send = [&] {
        in_flight.push_back(this->config.host_clock.now());
        if (auto err = this->master->send(); err) {
            if (send_failures++ == 0)
                LOG(WARNING) << "[ethercat] send failed on " << this->iface << ": "
                             << err.message();
        } else if (send_failures > 0) {
            LOG(INFO) << "[ethercat] send recovered on " << this->iface << " after "
                      << send_failures << " failures";
            send_failures = 0;
        }
    };

    std::optional<x::thread::rt::Handle> rt;
    if (this->config.rt_manager != nullptr) {
        rt.emplace(this->config.rt_manager->allocate({.enabled = true}));
        rt->apply();
    }
    this->write_outputs(false);
    send();
    while (true) {
        if (this->schedule_version.load(std::memory_order_acquire) != version) {
            const auto prev = sched == nullptr ? x::telem::Rate(0) : sched->rate;
            {
                std::lock_guard lock(this->schedule_mu);
                this->retired = std::move(sched);
                sched = this->schedule;
                version = this->schedule_version.load(std::memory_order_relaxed);
            }
            if (sched->rate != prev) {
                LOG(INFO) << "[ethercat] bus on " << this->iface << " cycles at "
                          << sched->rate;
                std::vector<PhaseLock::Sync0> syncs;
                for (const auto &regs: this->image.dc_sync)
                    if (regs.marks_data())
                        syncs.push_back({
                            .cycle = x::telem::TimeSpan(regs.sync0_cycle),
                            .event = master::DC_EPOCH +
                                     x::telem::TimeSpan(regs.sync0_shift),
                        });
                // The lock window is a part of a SYNC0 cycle, which a late wake misses.
                timer = this->config.host_clock.timer(
                    sched->rate.period(),
                    !syncs.empty()
                );
                if (syncs.empty())
                    phase.reset();
                else
                    phase.emplace(sched->rate, syncs);
                wake_dc.reset();
                // Moves each reader's next sample to the first cycle at or after the
                // time it was due.
                if (prev > x::telem::Rate(0))
                    for (const auto &entry: sched->inputs) {
                        auto &countdown = entry.first->countdown;
                        countdown = (countdown * millihertz(sched->rate) +
                                     millihertz(prev) - 1) /
                                    millihertz(prev);
                    }
            }
        }
        if (!taking) {
            if (wake_dc && phase) {
                timer.shift(phase->update(*wake_dc));
                if (!this->starting.load(std::memory_order_acquire) && phase->lost() &&
                    late_frames++ == 0)
                    LOG(WARNING) << "[ethercat] cycles on " << this->iface
                                 << " run late: a frame passed the reference clock "
                                    "near a SYNC0 event";
            }
            wake_dc.reset();
            timer.wait();
            if (!this->bus.running()) break;
        }
        // Takes each frame that came back, oldest first, as the data of its own cycle.
        const auto &err = this->master->receive();
        if (err.matches(errors::FRAME_IN_FLIGHT)) {
            taking = false;
            this->write_outputs(false);
            send();
            continue;
        }
        const auto frame_sent_at = in_flight.front();
        in_flight.erase(in_flight.begin());
        taking = !in_flight.empty();
        // The slaves latched these inputs when the frame passed them.
        auto latched_at = frame_sent_at;
        const bool starting = this->starting.load(std::memory_order_acquire);
        // A full cycle drops the inputs of the slaves that give none, since a slave
        // below SAFE-OP can keep its share of the working counter. A cycle short by
        // what the slaves out of OP can miss drops the inputs of each of them. The
        // cycle adds the slaves that it found with no inputs, and what they can miss,
        // before the supervisor sees them.
        const std::vector<uint16_t> *dropped = nullptr;
        if (!starting) {
            if (this->outage_version.load(std::memory_order_acquire) !=
                outage_version) {
                std::lock_guard lock(this->outage_mu);
                outage = this->outage;
                outage_version = this->outage_version.load(std::memory_order_relaxed);
            }
            const auto inputless = this->master->inputless();
            if (!err) {
                if (!outage.inputless.empty()) dropped = &outage.inputless;
            } else if (
                const auto missed = this->master->shortfall();
                err.matches(errors::WORKING_COUNTER_ERROR) && missed &&
                *missed >= outage.least_shortfall &&
                *missed <= outage.shortfall + inputless_shortfall(inputless)
            ) {
                dropped = &outage.positions;
                latched_at = frame_sent_at + lead;
                host_timed = true;
            }
            if (!inputless.empty() && (!err || dropped != nullptr)) {
                cycle_dropped.clear();
                std::ranges::set_union(
                    dropped == nullptr ? std::span<const uint16_t>() : *dropped,
                    inputless,
                    std::back_inserter(cycle_dropped)
                );
                dropped = &cycle_dropped;
            }
        }
        // After the slaves reach OP, a short frame carries the DC time only when it
        // passes a reference clock in OP. SOEM repeats the last DC time otherwise.
        const auto &ref = this->image.reference_clock;
        const bool timed = !err || shortfall(err) ||
                           (dropped != nullptr && ref &&
                            std::ranges::find(*dropped, *ref) == dropped->end());
        const auto dc = timed ? this->master->dc_time() : std::nullopt;
        if (dc) wake_dc = dc;
        if (!err) {
            if (dc && host_timed) {
                // The distributed clock can drift from the host times of the short
                // cycles, so its times go on from where those left off.
                clock = Clock();
                latched_at = clock.map(*dc, frame_sent_at + lead);
            } else if (dc)
                latched_at = clock.map(*dc, frame_sent_at);
            host_timed = false;
            lead = latched_at - frame_sent_at;
            if (receive_failures > 0) {
                LOG(INFO) << "[ethercat] cycles recovered on " << this->iface
                          << " after " << receive_failures << " failures";
                receive_failures = 0;
            }
        } else if (dropped == nullptr && !shortfall(err)) {
            if (receive_failures++ == 0)
                LOG(WARNING) << "[ethercat] cycle failed on " << this->iface << ": "
                             << err.message();
            // The reason can change during a run of failures, such as late frames that
            // turn into a dead link.
            if (!err.matches(stored_err) || err.data != stored_err.data) {
                stored_err = err;
                std::lock_guard lock(this->cycle_err_mu);
                this->cycle_err = err;
            }
        }
        const uint8_t *inputs = this->image.inputs.data();
        for (const auto &[input, decimation]: sched->inputs) {
            if (input->countdown > 0) {
                input->countdown--;
                continue;
            }
            input->countdown = decimation - 1;
            if (err && dropped == nullptr)
                input->failed_samples.fetch_add(1, std::memory_order_relaxed);
            else if (dropped == nullptr || !input->lose(*dropped, outage.silent))
                input->push(inputs, latched_at);
        }

        if (taking) continue;
        this->write_outputs(false);
        send();
    }
    const auto take_in_flight = [&] {
        const auto deadline = this->config.host_clock.now() + LAST_FRAME_TIMEOUT;
        x::errors::Error err;
        while (!in_flight.empty()) {
            err = this->master->receive();
            if (!err.matches(errors::FRAME_IN_FLIGHT))
                in_flight.erase(in_flight.begin());
            else if (this->config.host_clock.now() < deadline)
                this->config.host_clock.sleep(LAST_FRAME_POLL);
            else
                break;
        }
        return err;
    };
    // A master that keeps no frames in flight takes the frames of all its sends at
    // once.
    take_in_flight();
    // Sends the stop values of the last writer to close before the slaves leave OP.
    this->write_outputs(true);
    send();
    const auto last_err = take_in_flight();
    if (last_err && !shortfall(last_err))
        LOG(WARNING) << "[ethercat] last cycle failed on " << this->iface << ": "
                     << last_err.message();
    if (late_frames > 0)
        LOG(WARNING) << "[ethercat] " << late_frames << " frames on " << this->iface
                     << " passed the reference clock near a SYNC0 event";
    LOG(INFO) << "[ethercat] bus stopped on " << this->iface;
}

void Engine::supervise() {
    std::vector<slave::Health> last;
    Outage published;
    while (this->bus.running()) {
        auto health = this->master->check(this->bus);
        for (const auto &[pos, left]: this->left_out)
            health.push_back({.position = pos, .refusal = left.refusal});
        std::vector<uint16_t> restarted;
        for (const auto &h: health) {
            if (h.refusal) continue;
            const auto prev = std::ranges::find(
                last,
                h.position,
                &slave::Health::position
            );
            // A power restart brings a lost slave back in INIT or BOOT, and a cable
            // pull that trips its watchdog brings it back in SAFE-OP.
            if (prev != last.end() && prev->lost && !h.lost &&
                (h.state == slave::State::INIT || h.state == slave::State::BOOT))
                restarted.push_back(h.position);
            const bool was_operational = prev == last.end() || prev->operational();
            if (was_operational && !h.operational())
                LOG(WARNING) << "[ethercat] slave " << h.position << " on "
                             << this->iface << " left OP: " << h.to_string();
            else if (!was_operational && h.operational())
                LOG(INFO) << "[ethercat] slave " << h.position << " on " << this->iface
                          << " returned to OP";
            else if (!h.operational() && *prev != h)
                LOG(WARNING) << "[ethercat] slave " << h.position << " on "
                             << this->iface << ": " << h.to_string();
        }
        Outage next;
        for (const auto &h: health) {
            if (h.refusal || (!h.lost && h.state == slave::State::OP)) continue;
            next.positions.push_back(h.position);
            if (h.lost) next.silent.push_back(h.position);
            // Recovery takes a slave in SAFE-OP whose clock restore failed back to
            // PRE-OP after the publish.
            const bool inputs = !h.lost && h.state == slave::State::SAFE_OP &&
                                h.dc_restore_error.empty();
            if (!inputs) next.inputless.push_back(h.position);
            const auto &shares = this->image.working_counter_shares;
            if (const auto it = shares.find(h.position); it != shares.end()) {
                next.least_shortfall += it->second.outputs;
                next.shortfall += it->second.outputs + (inputs ? 0 : it->second.inputs);
            }
        }
        if (!restarted.empty()) {
            const auto now = this->config.host_clock.now();
            std::lock_guard lock(this->restarts_mu);
            for (const auto &s: this->image.slaves)
                if (std::ranges::find(restarted, s.properties.position) !=
                    restarted.end()) {
                    LOG(INFO) << "[ethercat] slave " << s.properties.position << " on "
                              << this->iface << " restarted";
                    this->restarts[restart_key(s.properties)] = {
                        .at = now,
                        .position = s.properties.position,
                    };
                }
        }
        std::ranges::sort(next.positions);
        std::ranges::sort(next.silent);
        std::ranges::sort(next.inputless);
        if (next != published) {
            std::lock_guard outputs_lock(this->outputs_mu);
            {
                std::lock_guard lock(this->outage_mu);
                this->outage = next;
                this->outage_version.fetch_add(1, std::memory_order_release);
            }
            for (const auto &output: this->outputs)
                output->update(next);
            published = std::move(next);
        }
        {
            std::lock_guard health_lock(this->health_mu);
            this->slave_health = health;
        }
        last = std::move(health);
        // Recovering after the publish keeps the samples of the other readers while a
        // slave recovers.
        this->master->recover(this->bus);
        this->bus.wait_for(SUPERVISOR_INTERVAL);
    }
}

x::errors::Error Engine::start_locked(
    const uint64_t mhz,
    const std::vector<SavedSlave> &saved,
    const Settings &settings,
    const Start &start
) {
    const auto start_cycle = [this, mhz](const master::Image &image) {
        this->image = image;
        {
            std::lock_guard lock(this->write_mu);
            this->write_staging.assign(image.outputs.size(), 0);
            this->write_active.assign(image.outputs.size(), 0);
        }
        this->publish_schedule_locked(mhz);
        this->starting.store(true, std::memory_order_release);
        this->bus.start();
        this->cycle_thread = std::thread(&Engine::run, this);
    };
    master::Plan plan{
        .period = x::telem::Rate(static_cast<double>(mhz) / 1000).period(),
        .cancel = &start.cancel,
        .progress = [this, &start](const size_t operational, const size_t total) {
            report(
                start,
                {
                    .message = "Starting " + this->network_name,
                    .description = std::format(
                        "{} of {} modules in OP.",
                        operational,
                        total
                    ),
                    .limit = ACTIVATE_STEP_LIMIT,
                }
            );
        },
    };
    this->left_out.clear();
    // The activation brings a restarted slave up, so changed() misses it afterwards.
    auto [change, change_err] = this->changed_locked();
    if (change_err) return change_err;
    auto &restarted = change.restarted;
    // The cached scan holds the setup from before the restart.
    if (!restarted.empty()) this->master->deactivate();
    std::set<uint16_t> reset;
    const auto &clock = this->config.host_clock;
    const auto &retry = this->config.start_retry;
    int failures = 0;
    x::telem::TimeSpan longest_try(0);
    if (clock.steady() >= start.deadline)
        return start.blocked ? start_blocked_error(this->network_name)
                             : start_late_error(this->network_name);
    while (true) {
        if (!start.cancel.running()) return start_canceled_error(this->network_name);
        const auto try_started = clock.steady();
        auto [slaves, err] = this->scan_locked();
        std::map<uint16_t, slave::Refusal> refused;
        if (!err) {
            this->slaves = std::move(slaves);
            this->track_restarts_locked(std::exchange(restarted, {}));
            std::tie(this->device_configs, err) = this->plan_locked(saved);
            plan.slaves = this->device_configs;
            for (const auto &[pos, params]: settings)
                plan.slaves[pos].add_startup_parameters(params);
            // The next scan reads the serial numbers again.
            if (err.matches(errors::ACTIVATION_ERROR)) this->master->deactivate();
        }
        if (!err) {
            for (const auto &s: this->slaves)
                if (const auto it = plan.slaves.find(s.properties.position);
                    it != plan.slaves.end() && s.properties.shared())
                    it->second.startup_parameters.clear();
            const auto starting = std::ranges::count_if(
                this->slaves,
                [&plan](const slave::DiscoveryResult &s) {
                    const auto it = plan.slaves.find(s.properties.position);
                    return it == plan.slaves.end() || it->second.enabled;
                }
            );
            report(
                start,
                {
                    .message = "Starting " + this->network_name,
                    .description = std::format(
                        "Bringing {} module{} to OP.",
                        starting,
                        starting == 1 ? "" : "s"
                    ),
                    .limit = ACTIVATE_STEP_LIMIT,
                }
            );
            this->activating.store(true, std::memory_order_release);
            auto activation = this->master->activate(plan, start_cycle);
            this->activating.store(false, std::memory_order_release);
            err = activation.err;
            if (!err) break;
            refused = std::move(activation.refusals);
            for (const auto &s: this->slaves)
                if (const auto it = refused.find(s.properties.position);
                    it != refused.end()) {
                    const auto cfg = plan.slaves.find(it->first);
                    advise_switch(
                        s.properties,
                        cfg == plan.slaves.end() ? std::nullopt : cfg->second.op_mode,
                        it->second
                    );
                }
            std::map<uint16_t, slave::Refusal> leaving;
            for (const auto &[pos, refusal]: refused) {
                if (std::ranges::find(STALE_LAYOUT_CODES, refusal.code) !=
                        STALE_LAYOUT_CODES.end() &&
                    reset.insert(pos).second) {
                    LOG(WARNING)
                        << "[ethercat] slave " << pos << " on " << this->iface << " "
                        << refusal.reason << ", starting again with it reset";
                    const auto s = std::ranges::find_if(
                        this->slaves,
                        [pos](const auto &d) { return d.properties.position == pos; }
                    );
                    report(
                        start,
                        {
                            .message = std::format(
                                "Resetting module {} ({})",
                                pos,
                                s == this->slaves.end() ? "" : s->properties.name
                            ),
                            .description = errors::sentence("It " + refusal.reason) +
                                           " Starting " + this->network_name +
                                           " again with it reset.",
                            .limit = RESET_STEP_LIMIT,
                        }
                    );
                    continue;
                }
                leaving.insert({pos, refusal});
            }
            // A slave with no process data, such as a coupler, gives a task nothing to
            // run.
            const bool io_left = std::ranges::any_of(
                this->image.offsets,
                [&leaving](const auto &offset) {
                    return !leaving.contains(offset.first.slave_position);
                }
            );
            if (this->bus.running())
                this->stop_locked();
            else
                this->master->deactivate();
            if (!start.cancel.running())
                return start_canceled_error(this->network_name);
            if (!refused.empty() && io_left) {
                for (const auto &[pos, refusal]: leaving)
                    LOG(WARNING)
                        << "[ethercat] starting " << this->iface << " without slave "
                        << pos << ": it " << refusal.reason;
                for (const auto &s: this->slaves)
                    if (const auto it = leaving.find(s.properties.position);
                        it != leaving.end())
                        this->left_out[it->first] = {
                            .name = s.properties.name,
                            .refusal = it->second,
                        };
                const auto now = clock.steady();
                longest_try = std::max(
                    longest_try,
                    x::telem::TimeSpan(now - try_started)
                );
                // The deadline check below fails a pass whose next try cannot end in
                // time.
                if (now + longest_try.chrono() <= start.deadline) continue;
            }
        }
        const auto now = clock.steady();
        longest_try = std::max(longest_try, x::telem::TimeSpan(now - try_started));
        const auto wait = retry.interval(static_cast<size_t>(++failures));
        // A retry cannot clear a refused state, a setting a slave aborts, or a slave
        // config that cannot run. A try that ends after the deadline hides its error
        // behind the task timeout.
        if (err.matches(errors::STATE_CHANGE_REFUSED) ||
            err.matches(errors::STARTUP_PARAMETER_REJECTED) ||
            err.matches(errors::SLAVE_CONFIG_ERROR) ||
            (retry.max_retries != x::breaker::RETRY_INFINITELY &&
             failures > retry.max_retries) ||
            now + (retry.max_interval + longest_try).chrono() > start.deadline)
            return start_error(err, refused, this->left_out);
        LOG(WARNING) << "[ethercat] failed to start " << this->iface
                     << ", trying again in " << wait << ": " << err.message();
        const auto why = errors::sentence(err.data);
        report(
            start,
            {
                .message = "Retrying the start",
                .description = (why.empty() ? "" : why + " ") + "Trying again in " +
                               wait.to_string() + ".",
                .limit = wait + RETRY_STEP_SLACK,
            }
        );
        start.cancel.wait_for(wait);
    }
    this->slaves = this->image.slaves;
    this->unchecked.clear();
    for (const auto &[pos, cfg]: plan.slaves)
        if (!cfg.enabled) this->unchecked.insert(pos);
    this->track_restarts_locked({});
    this->starting.store(false, std::memory_order_release);
    this->supervisor_thread = std::thread(&Engine::supervise, this);
    return x::errors::NIL;
}

void Engine::stop_locked() {
    if (!this->bus.stop()) return;
    this->stops++;
    this->cycle_thread.join();
    if (this->supervisor_thread.joinable()) this->supervisor_thread.join();
    // The supervisor counts the restarts of the slaves it checks, where changed() also
    // reports a slave stuck in INIT.
    if (auto [change, err] = this->changed_locked(); err)
        LOG(WARNING) << "[ethercat] failed to check " << this->iface
                     << " for restarted slaves: " << err.message();
    else {
        std::erase_if(change.restarted, [this](const uint16_t pos) {
            return !this->unchecked.contains(pos);
        });
        this->track_restarts_locked(change.restarted);
        // A slave off the bus now restarts when its power returns, so the scan that
        // finds it again gives it a new time.
        if (change.stale)
            if (const auto scan_err = this->rescan_locked({}).second)
                LOG(WARNING) << "[ethercat] failed to scan " << this->iface
                             << " for slaves off the bus: " << scan_err.message();
    }
    this->master->deactivate();
    this->image = {};
    this->cycle_mhz.store(0, std::memory_order_release);
    {
        std::lock_guard lock(this->schedule_mu);
        this->schedule.reset();
        this->retired.reset();
    }
    {
        std::lock_guard lock(this->outage_mu);
        this->outage = {};
        this->outage_version.fetch_add(1, std::memory_order_release);
    }
    std::lock_guard lock(this->health_mu);
    this->slave_health.clear();
}

std::pair<std::map<uint16_t, slave::Config>, x::errors::Error>
Engine::plan_locked(const std::vector<SavedSlave> &saved) const {
    std::vector<slave::Identity> identities;
    identities.reserve(saved.size());
    for (const auto &s: saved)
        identities.push_back(s.identity);
    const auto found = slave::locate(identities, this->slaves);
    std::map<uint16_t, slave::Config> configs;
    std::map<uint16_t, const slave::Identity *> owners;
    for (size_t i = 0; i < saved.size(); ++i) {
        const auto position = found[i];
        if (!position) continue;
        const auto &[identity, config] = saved[i];
        const auto [owner, added] = owners.emplace(*position, &identity);
        if (added) {
            configs[*position] = config;
            continue;
        }
        if (configs[*position] == config) continue;
        const auto label = [](const slave::Identity &id) {
            return id.serial == 0 ? id.name
                                  : std::format("{} (serial {})", id.name, id.serial);
        };
        auto first = label(*owner->second);
        auto second = label(identity);
        if (second < first) std::swap(first, second);
        // A scan that reads the serial number tells which of the two is on the bus.
        if (const auto live = std::ranges::find(
                this->slaves,
                *position,
                [](const slave::DiscoveryResult &d) { return d.properties.position; }
            );
            (owner->second->serial != 0 || identity.serial != 0) &&
            live != this->slaves.end() && live->serial_unread())
            return {
                {},
                x::errors::Error(
                    errors::ACTIVATION_ERROR,
                    errors::sentence(
                        std::format(
                            "{} and {} are both saved for module {} on {} with "
                            "different settings, and the module did not give its "
                            "serial number: {}",
                            first,
                            second,
                            *position,
                            this->network_name,
                            live->status.starting ? "it is still starting up"
                                                  : live->status.serial_error
                        )
                    )
                )
            };
        return {
            {},
            x::errors::Error(
                errors::SLAVE_CONFIG_ERROR,
                std::format(
                    "{} and {} are both saved for module {} on {} with different "
                    "settings. Give them the same settings, or delete the device that "
                    "is no longer on the bus.",
                    first,
                    second,
                    *position,
                    this->network_name
                )
            )
        };
    }
    for (const auto &[pos, _]: this->left_out)
        configs[pos].enabled = false;
    return {configs, x::errors::NIL};
}

std::pair<std::vector<SavedSlave>, x::errors::Error> Engine::read_slave_configs(
    std::unique_lock<std::mutex> &lock,
    const Start &start
) const {
    while (!this->running()) {
        const auto stops = this->stops;
        lock.unlock();
        auto result = this->config.slave_configs();
        if (auto err = this->lock_for(lock, start)) return {{}, err};
        // A bus that ran during the read may have run with newer settings.
        if (this->stops == stops) return result;
    }
    return {};
}

std::vector<x::telem::Rate> Engine::read_rates_locked() const {
    std::vector<x::telem::Rate> rates;
    rates.reserve(this->inputs.size());
    for (const auto &input: this->inputs)
        rates.push_back(input->rate);
    return rates;
}

std::vector<x::telem::Rate> Engine::write_rates_locked() const {
    std::vector<x::telem::Rate> rates;
    rates.reserve(this->outputs.size());
    for (const auto &output: this->outputs)
        rates.push_back(output->rate);
    return rates;
}

void Engine::publish_schedule_locked(const uint64_t mhz) {
    auto next = std::make_shared<Schedule>();
    next->rate = x::telem::Rate(static_cast<double>(mhz) / 1000);
    for (const auto &input: this->inputs)
        next->inputs.emplace_back(input, mhz / millihertz(input->rate));
    std::shared_ptr<const Schedule> retired;
    {
        std::lock_guard lock(this->schedule_mu);
        retired = std::move(this->retired);
        this->schedule = std::move(next);
        this->schedule_version.fetch_add(1, std::memory_order_release);
    }
    this->cycle_mhz.store(mhz, std::memory_order_release);
}

std::pair<uint64_t, x::errors::Error> Engine::rate_locked(
    const std::vector<x::telem::Rate> &reads,
    const std::vector<x::telem::Rate> &writes
) const {
    auto [mhz, err] = bus_rate(reads, writes, this->network_name);
    if (err || !this->running() || this->image.dc_sync.empty()) return {mhz, err};
    // The SYNC0 and SYNC1 cycles came from the bus period at activation.
    const auto fixed = this->cycle_mhz.load(std::memory_order_acquire);
    if (serves(fixed, reads, writes)) return {fixed, x::errors::NIL};
    return {
        0,
        x::errors::Error(
            errors::RATE_MISMATCH,
            std::format(
                "The bus on {0} cycles at {1} Hz while a module runs DC sync, and the "
                "rate cannot change until every task on the port stops. Use a sample "
                "rate that {1} Hz is a multiple of, or an execution rate of at most "
                "{1} Hz.",
                this->network_name,
                static_cast<double>(fixed) / 1000
            )
        )
    };
}

void Engine::reschedule_locked() {
    if (this->inputs.empty() && this->outputs.empty()) {
        this->stop_locked();
        return;
    }
    // Rates that had a bus rate keep one when a task leaves.
    this->publish_schedule_locked(
        this->rate_locked(this->read_rates_locked(), this->write_rates_locked()).first
    );
}

void Engine::write_outputs(const bool wait) {
    std::unique_lock lock(this->write_mu, std::try_to_lock);
    if (!lock.owns_lock() && wait) lock.lock();
    if (lock.owns_lock()) this->write_active = this->write_staging;
    const auto outputs = this->image.outputs;
    std::memcpy(outputs.data(), this->write_active.data(), this->write_active.size());
}

x::errors::Error Engine::cycle_error() const {
    x::errors::Error err;
    {
        std::lock_guard lock(this->cycle_err_mu);
        err = this->cycle_err;
    }
    if (!err.matches(errors::WORKING_COUNTER_ERROR)) return err;
    const auto &slaves = this->image.slaves;
    for (const auto &h: this->health()) {
        if (h.refusal || h.operational()) continue;
        const auto slave = std::ranges::find(slaves, h.position, [](const auto &s) {
            return s.properties.position;
        });
        err.data += std::format(
            ". Module {} ({}) is not in OP: {}",
            h.position,
            slave == slaves.end() ? "" : slave->properties.name,
            h.to_string()
        );
    }
    return err;
}

void Engine::close_input(const std::shared_ptr<Input> &input) {
    std::lock_guard lock(this->mu);
    std::erase(this->inputs, input);
    this->reschedule_locked();
}

void Engine::close_output(const std::shared_ptr<Output> &output) {
    std::lock_guard lock(this->mu);
    {
        std::lock_guard write_lock(this->write_mu);
        {
            std::lock_guard outputs_lock(this->outputs_mu);
            std::erase(this->outputs, output);
        }
        for (size_t i = 0; i < output->entries.size(); ++i) {
            const auto &entry = output->entries[i];
            const auto &offset = output->offsets[i];
            telem::write_pdo_from_value(
                this->write_staging.data() + offset.byte,
                offset.bit,
                entry.bit_length,
                entry.data_type,
                output->stop_values[i]
            );
        }
    }
    this->reschedule_locked();
}

Engine::Engine(
    std::string iface,
    std::shared_ptr<master::Master> master,
    Config config
):
    config(std::move(config)),
    iface(std::move(iface)),
    master(std::move(master)),
    network_name(this->master->name()) {}

Engine::~Engine() {
    std::lock_guard lock(this->mu);
    this->stop_locked();
}

bool Engine::shared_locked(const uint16_t position) const {
    const auto slave = std::ranges::find(this->slaves, position, [](const auto &s) {
        return s.properties.position;
    });
    return slave != this->slaves.end() && slave->properties.shared();
}

x::errors::Error Engine::check_settings_locked(const Settings &settings) const {
    for (const auto &input: this->inputs)
        for (const auto &[pos, params]: settings) {
            if (this->shared_locked(pos)) continue;
            const auto held = input->settings.find(pos);
            if (held == input->settings.end()) continue;
            for (const auto &param: params)
                for (const auto &other: held->second)
                    if (other.index == param.index &&
                        other.sub_index == param.sub_index &&
                        (other.value != param.value ||
                         other.data_type != param.data_type))
                        return x::errors::Error(
                            errors::SETTING_CONFLICT,
                            std::format(
                                "Another task sets {} of module {} to {}. Stop it, or "
                                "give both tasks the same value.",
                                telem::format_index_sub_index(
                                    param.index,
                                    param.sub_index
                                ),
                                pos,
                                other.value
                            )
                        );
        }
    return x::errors::NIL;
}

std::pair<Engine::Joined, x::errors::Error> Engine::join_locked(
    std::unique_lock<std::mutex> &lock,
    const std::vector<pdo::Entry> &entries,
    const x::telem::Rate rate,
    const bool output,
    const Settings &settings,
    const Start &start
) {
    const auto [saved, configs_err] = this->read_slave_configs(lock, start);
    if (!lock.owns_lock()) return {{}, configs_err};
    if (auto err = this->check_settings_locked(settings)) return {{}, err};
    auto reads = this->read_rates_locked();
    auto writes = this->write_rates_locked();
    (output ? writes : reads).push_back(rate);
    const auto [mhz, rate_err] = this->rate_locked(reads, writes);
    if (rate_err) return {{}, rate_err};
    const bool started = !this->running();
    if (started) {
        if (configs_err) return {{}, configs_err};
        if (auto err = this->start_locked(mhz, saved, settings, start))
            return {{}, err};
    }
    std::set<uint16_t> positions;
    for (const auto &entry: entries)
        positions.insert(entry.slave_position);
    for (const auto &[pos, _]: settings)
        positions.insert(pos);
    std::map<uint16_t, size_t> counts;
    for (const auto pos: positions) {
        const auto it = this->device_configs.find(pos);
        auto cfg = it == this->device_configs.end() ? slave::Config{} : it->second;
        if (const auto params = settings.find(pos); params != settings.end())
            cfg.add_startup_parameters(params->second);
        counts[pos] = cfg.startup_parameters.size();
    }
    auto [offsets, err] = this->offsets_locked(entries);
    if (err) {
        if (started)
            this->stop_locked();
        else if (err.matches(errors::PDO_NOT_MAPPED))
            err.data += " Stop every task on " + this->network_name +
                        ", then start them again.";
        return {{}, err};
    }
    // A reader that the running bus cannot map must leave the slaves as they are.
    if (!started)
        for (const auto &[pos, params]: settings) {
            if (this->shared_locked(pos)) continue;
            if (auto write_err = this->master->write(pos, params))
                return {{}, write_err};
        }
    return {
        {
            .mhz = mhz,
            .offsets = std::move(offsets),
            .unapplied = unapplied(this->slaves, counts),
        },
        x::errors::NIL
    };
}

Engine::Start Engine::begin_start(
    x::breaker::Breaker &cancel,
    std::function<void(const Step &)> report
) const {
    return {
        .deadline = this->config.host_clock.steady() +
                    this->config.start_timeout.chrono(),
        .cancel = cancel,
        .report = std::move(report),
    };
}

std::pair<std::unique_ptr<Engine::Reader>, x::errors::Error> Engine::open_reader(
    const std::vector<pdo::Entry> &entries,
    const x::telem::Rate sample_rate,
    const size_t samples_per_read,
    const Start &start,
    const Settings &settings
) {
    this->opens.fetch_add(1, std::memory_order_acq_rel);
    x::defer::defer done([this] {
        this->opens.fetch_sub(1, std::memory_order_acq_rel);
    });
    std::unique_lock lock(this->mu, std::defer_lock);
    if (auto lock_err = this->lock_for(lock, start)) return {nullptr, lock_err};
    auto [joined, err] = this->join_locked(
        lock,
        entries,
        sample_rate,
        false,
        settings,
        start
    );
    if (err) return {nullptr, err};
    auto input = std::make_shared<Input>(
        entries,
        joined.offsets,
        this->image.slaves,
        sample_rate,
        samples_per_read
    );
    input->settings = settings;
    input->unapplied = std::move(joined.unapplied);
    this->inputs.push_back(input);
    this->publish_schedule_locked(joined.mhz);
    return {std::make_unique<Reader>(*this, std::move(input)), x::errors::NIL};
}

std::pair<std::unique_ptr<Engine::Writer>, x::errors::Error> Engine::open_writer(
    const std::vector<Writer::Entry> &entries,
    const x::telem::Rate execution_rate,
    const Start &start,
    std::function<void()> changed
) {
    std::vector<pdo::Entry> pdo_entries;
    std::vector<x::telem::SampleValue> stop_values;
    pdo_entries.reserve(entries.size());
    stop_values.reserve(entries.size());
    for (const auto &[entry, stop_value]: entries) {
        pdo_entries.push_back(entry);
        stop_values.push_back(stop_value);
    }
    this->opens.fetch_add(1, std::memory_order_acq_rel);
    x::defer::defer done([this] {
        this->opens.fetch_sub(1, std::memory_order_acq_rel);
    });
    std::unique_lock lock(this->mu, std::defer_lock);
    if (auto lock_err = this->lock_for(lock, start)) return {nullptr, lock_err};
    auto [joined, err] = this->join_locked(
        lock,
        pdo_entries,
        execution_rate,
        true,
        {},
        start
    );
    if (err) return {nullptr, err};
    const auto slaves = slaves_of(pdo_entries, this->image.slaves);
    auto output = std::make_shared<Output>(Output{
        .entries = std::move(pdo_entries),
        .rate = execution_rate,
        .offsets = std::move(joined.offsets),
        .stop_values = std::move(stop_values),
        .unapplied = std::move(joined.unapplied),
        .slaves = std::vector<Output::Slave>(slaves.size()),
        .changed = std::move(changed),
    });
    {
        std::scoped_lock lock(this->outputs_mu, this->outage_mu);
        for (size_t i = 0; i < slaves.size(); ++i) {
            auto &slave = output->slaves[i];
            std::tie(slave.position, slave.name) = slaves[i];
            slave.out = std::ranges::binary_search(
                this->outage.positions,
                slave.position
            );
            slave.silent = std::ranges::binary_search(
                this->outage.silent,
                slave.position
            );
        }
        this->outputs.push_back(output);
    }
    this->publish_schedule_locked(joined.mhz);
    return {std::make_unique<Writer>(*this, std::move(output)), x::errors::NIL};
}

x::telem::Rate Engine::cycle_rate() const {
    return x::telem::Rate(
        static_cast<double>(this->cycle_mhz.load(std::memory_order_acquire)) / 1000
    );
}

std::pair<std::vector<slave::DiscoveryResult>, x::errors::Error> Engine::discover() {
    std::lock_guard lock(this->mu);
    return this->discover_locked();
}

std::pair<std::vector<slave::DiscoveryResult>, x::errors::Error>
Engine::discover(const Start &start) {
    std::unique_lock lock(this->mu, std::defer_lock);
    if (auto err = this->lock_for(lock, start)) return {{}, err};
    if (!this->running())
        report(
            start,
            {.message = "Scanning " + this->network_name, .limit = SCAN_STEP_LIMIT}
        );
    return this->discover_locked();
}

std::pair<std::vector<slave::DiscoveryResult>, x::errors::Error>
Engine::discover_locked() {
    if (this->running()) {
        this->track_restarts_locked({});
        return {this->slaves, x::errors::NIL};
    }
    auto [slaves, err] = this->scan_locked();
    if (err) return {{}, err};
    const auto [change, change_err] = this->master->changed();
    if (change_err) return {{}, change_err};
    if (change.stale) {
        LOG(INFO) << "[ethercat] the slaves on " << this->iface
                  << " changed since the last discovery, rediscovering";
        return this->rescan_locked(change.restarted);
    }
    this->slaves = std::move(slaves);
    this->track_restarts_locked({});
    return {this->slaves, x::errors::NIL};
}

std::pair<std::vector<slave::DiscoveryResult>, x::errors::Error> Engine::rediscover() {
    std::lock_guard lock(this->mu);
    if (this->running()) {
        this->track_restarts_locked({});
        return {this->slaves, x::errors::NIL};
    }
    return this->rediscover_locked();
}

x::errors::Error
Engine::lock_for(std::unique_lock<std::mutex> &lock, const Start &start) const {
    auto polled = this->config.host_clock.steady();
    // A wait of one poll is a short hold, such as a notify of the cycle thread.
    for (int polls = 0; !lock.try_lock(); polls++) {
        if (polls == 1)
            report(
                start,
                {
                    .message = "Waiting for another task",
                    .description = "Another task is using " + this->network_name + ".",
                    .limit = WAIT_STEP_LIMIT,
                }
            );
        start.blocked = true;
        if (!start.cancel.running()) return start_canceled_error(this->network_name);
        const auto now = this->config.host_clock.steady();
        if (this->activating.load(std::memory_order_acquire))
            start.deadline += now - polled;
        polled = now;
        if (now >= start.deadline) return start_blocked_error(this->network_name);
        start.cancel.wait_for(LOCK_POLL_INTERVAL);
    }
    return x::errors::NIL;
}

std::pair<std::unique_lock<std::mutex>, x::errors::Error> Engine::lock_stopped() {
    std::unique_lock lock(this->mu, std::defer_lock);
    // An open counts itself before it waits on mu, so this never waits behind a start.
    while (this->opens.load(std::memory_order_acquire) == 0 && !lock.try_lock())
        std::this_thread::sleep_for(LOCK_POLL_INTERVAL.chrono());
    if (!lock.owns_lock())
        return {
            std::move(lock),
            x::errors::Error(
                errors::SLAVE_CONFIG_ERROR,
                "A task is starting on " + this->network_name +
                    ". Try again when the start ends, or stop the task."
            )
        };
    if (this->running())
        return {
            std::move(lock),
            x::errors::Error(
                errors::SLAVE_CONFIG_ERROR,
                "A task uses " + this->network_name +
                    ". Stop the tasks that use it first."
            )
        };
    return {std::move(lock), x::errors::NIL};
}

x::errors::Error
Engine::configure(const slave::Properties &slave, const Parameters &parameters) {
    const auto [lock, lock_err] = this->lock_stopped();
    if (lock_err) return lock_err;
    const auto [slaves, err] = this->rediscover_locked();
    if (err) return err;
    const slave::Identity identity{
        .position = slave.position,
        .vendor_id = slave.vendor_id,
        .product_code = slave.product_code,
        .serial = slave.serial,
        .name = slave.name,
    };
    const auto position = slave::locate({identity}, slaves).front();
    if (!position)
        return x::errors::Error(
            errors::SLAVE_CONFIG_ERROR,
            slave.name + " is not on " + this->network_name +
                ". Check its power and cabling."
        );
    const auto found = std::ranges::find(
        slaves,
        *position,
        [](const slave::DiscoveryResult &s) { return s.properties.position; }
    );
    const auto [params, params_err] = parameters(*found);
    if (params_err) return params_err;
    if (params.empty()) return x::errors::NIL;
    if (auto write_err = this->master->write(found->properties.position, params))
        return write_err;
    return this->rediscover_locked().second;
}

std::pair<uint32_t, x::errors::Error>
Engine::read(const uint16_t position, const uint16_t index, const uint8_t sub_index) {
    std::lock_guard lock(this->mu);
    return this->master->read(position, index, sub_index);
}

std::pair<std::vector<slave::DiscoveryResult>, x::errors::Error>
Engine::rediscover_locked() {
    const auto [change, err] = this->changed_locked();
    if (err) return {{}, err};
    return this->rescan_locked(change.restarted);
}

std::pair<std::vector<slave::DiscoveryResult>, x::errors::Error> Engine::scan_locked() {
    auto result = this->master->scan();
    // A filter that blocks the replies leaves the slaves running.
    if (const auto &err = result.second;
        err.matches(errors::NO_SLAVES_FOUND) && !err.matches(errors::REPLIES_BLOCKED)) {
        this->slaves.clear();
        this->track_restarts_locked({});
    }
    return result;
}

std::pair<std::vector<slave::DiscoveryResult>, x::errors::Error>
Engine::rescan_locked(const std::set<uint16_t> &restarted) {
    this->master->deactivate();
    auto [slaves, err] = this->scan_locked();
    if (err) return {{}, err};
    this->slaves = std::move(slaves);
    this->track_restarts_locked(restarted);
    return {this->slaves, x::errors::NIL};
}

std::pair<master::Change, x::errors::Error> Engine::changed_locked() {
    if (!this->master->restarts_visible()) return {{}, x::errors::NIL};
    auto [change, err] = this->master->changed();
    // A master that has no scan has nothing to compare.
    if (err.matches(errors::MASTER_INIT_ERROR)) return {{}, x::errors::NIL};
    return {std::move(change), err};
}

Engine::RestartKey Engine::restart_key(const slave::Properties &props) {
    return {
        props.vendor_id,
        props.product_code,
        props.serial,
        props.serial == 0 ? props.position : uint16_t{0},
    };
}

void Engine::track_restarts_locked(const std::set<uint16_t> &restarted) {
    if (!this->master->restarts_visible()) return;
    const auto now = this->config.host_clock.now();
    std::lock_guard lock(this->restarts_mu);
    decltype(this->restarts) found;
    for (auto &slave: this->slaves) {
        const auto &props = slave.properties;
        auto key = restart_key(props);
        auto it = this->restarts.find(key);
        const bool unread = slave.serial_unread();
        // An unread serial number differs from the read one, so a slave with one, or
        // with one at its last discovery, is the slave of its model at its position.
        if (unread || it == this->restarts.end())
            if (const auto earlier = std::ranges::find_if(
                    this->restarts,
                    [&props, unread](const auto &entry) {
                        return (unread || entry.second.unread) &&
                               std::get<0>(entry.first) == props.vendor_id &&
                               std::get<1>(entry.first) == props.product_code &&
                               entry.second.position == props.position;
                    }
                );
                earlier != this->restarts.end()) {
                if (unread) key = earlier->first;
                it = earlier;
            }
        const auto at = it == this->restarts.end() || restarted.contains(props.position)
                          ? now
                          : it->second.at;
        found[key] = {.at = at, .position = props.position, .unread = unread};
        slave.restarted_at = at;
    }
    this->restarts = std::move(found);
}

std::pair<std::vector<pdo::Offset>, x::errors::Error>
Engine::offsets_locked(const std::vector<pdo::Entry> &entries) const {
    std::vector<pdo::Offset> offsets;
    offsets.reserve(entries.size());
    for (const auto &entry: entries) {
        const auto pos = entry.slave_position;
        if (const auto it = this->left_out.find(pos); it != this->left_out.end())
            return {{}, left_out_error(pos, it->second.name, it->second.refusal)};
        const auto [offset, err] = this->image.offset(entry);
        if (err) return {{}, err};
        offsets.push_back(offset);
    }
    return {offsets, x::errors::NIL};
}

std::vector<slave::Health> Engine::health() const {
    std::lock_guard lock(this->health_mu);
    return this->slave_health;
}
}
