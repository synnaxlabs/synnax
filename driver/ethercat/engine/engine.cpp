// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cmath>
#include <format>
#include <numeric>
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
/// @brief how often a call that needs the stopped bus tries mu again.
const auto LOCK_POLL_INTERVAL = x::telem::MILLISECOND * 10;
/// @brief the length of the sample buffer of each reader, in seconds of samples.
constexpr double BUFFER_SECONDS = 2;
constexpr size_t MIN_BUFFER_SAMPLES = 16;

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

/// @brief returns the error for an entry on a slave that the bus started without.
x::errors::Error left_out_error(
    const uint16_t position,
    const std::string &name,
    const slave::Refusal &refusal
) {
    auto msg = std::format(
        "Slave {} ({}) is not on the network: it {}. Fix the device, then stop every "
        "task on the network to try it again.",
        position,
        name,
        refusal.reason
    );
    if (!refusal.advice.empty()) msg.append(" ").append(refusal.advice);
    return x::errors::Error(errors::STATE_CHANGE_REFUSED, msg);
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
        err.data += " The Driver left out slave " + std::to_string(pos) +
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
    const x::telem::Rate rate,
    const size_t samples_per_read
):
    rate(rate), samples_per_read(samples_per_read) {
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

void Engine::run() {
    std::shared_ptr<const Schedule> sched;
    uint64_t version = 0;
    x::loop::Timer timer;
    std::optional<PhaseLock> phase;
    uint64_t late_frames = 0;
    uint64_t receive_failures = 0;
    uint64_t send_failures = 0;
    x::telem::TimeStamp sent_at{};
    Clock clock;
    // The working counter falls short until the slaves reach OP.
    const auto shortfall = [this](const x::errors::Error &err) {
        return this->starting.load(std::memory_order_acquire) &&
               err.matches(errors::WORKING_COUNTER_ERROR);
    };

    const auto send = [&] {
        sent_at = this->config.host_clock.now();
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
        timer.wait();
        if (!this->bus.running()) break;

        // The slaves latched these inputs when the previous cycle's frame passed.
        auto latched_at = sent_at;
        const auto err = this->master->receive();
        const bool starting = this->starting.load(std::memory_order_acquire);
        // After the slaves reach OP, a short frame can miss the reference clock, and
        // SOEM repeats the last DC time.
        const auto dc = (!err || shortfall(err)) ? this->master->dc_time()
                                                 : std::nullopt;
        if (dc && phase) {
            timer.shift(phase->update(*dc));
            if (!starting && phase->lost() && late_frames++ == 0)
                LOG(WARNING) << "[ethercat] cycles on " << this->iface
                             << " run late: a frame passed the reference clock near "
                                "a SYNC0 event";
        }
        if (!err) {
            if (dc) latched_at = clock.map(*dc, sent_at);
            if (receive_failures > 0) {
                LOG(INFO) << "[ethercat] cycles recovered on " << this->iface
                          << " after " << receive_failures << " failures";
                receive_failures = 0;
            }
        } else if (!shortfall(err) && receive_failures++ == 0) {
            LOG(WARNING) << "[ethercat] cycle failed on " << this->iface << ": "
                         << err.message();
            std::lock_guard lock(this->cycle_err_mu);
            this->cycle_err = err;
        }
        const uint8_t *inputs = this->image.inputs.data();
        for (const auto &[input, decimation]: sched->inputs) {
            if (input->countdown > 0) {
                input->countdown--;
                continue;
            }
            input->countdown = decimation - 1;
            if (err)
                input->failed_samples.fetch_add(1, std::memory_order_relaxed);
            else
                input->push(inputs, latched_at);
        }

        this->write_outputs(false);
        send();
    }
    // Sends the stop values of the last writer to close before the slaves leave OP.
    this->write_outputs(true);
    send();
    if (const auto err = this->master->receive(); err && !shortfall(err))
        LOG(WARNING) << "[ethercat] last cycle failed on " << this->iface << ": "
                     << err.message();
    if (late_frames > 0)
        LOG(WARNING) << "[ethercat] " << late_frames << " frames on " << this->iface
                     << " passed the reference clock near a SYNC0 event";
    LOG(INFO) << "[ethercat] bus stopped on " << this->iface;
}

void Engine::supervise() {
    std::vector<slave::Health> last;
    while (this->bus.running()) {
        auto health = this->master->check(this->bus);
        for (const auto &[pos, left]: this->left_out)
            health.push_back({.position = pos, .refusal = left.refusal});
        for (const auto &h: health) {
            if (h.refusal) continue;
            const auto prev = std::ranges::find(
                last,
                h.position,
                &slave::Health::position
            );
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
        {
            std::lock_guard health_lock(this->health_mu);
            this->slave_health = health;
        }
        last = std::move(health);
        this->bus.wait_for(SUPERVISOR_INTERVAL);
    }
}

x::errors::Error
Engine::start_locked(const uint64_t mhz, std::map<uint16_t, slave::Config> configs) {
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
        .slaves = std::move(configs),
    };
    this->left_out.clear();
    x::breaker::Breaker retry(this->config.start_retry);
    retry.start();
    const auto &clock = this->config.host_clock;
    const auto started = clock.steady();
    x::telem::TimeSpan longest_try(0);
    while (true) {
        const auto try_started = clock.steady();
        auto [slaves, err] = this->master->scan();
        std::map<uint16_t, slave::Refusal> refused;
        if (!err) {
            this->slaves = std::move(slaves);
            for (const auto &s: this->slaves)
                if (const auto it = plan.slaves.find(s.properties.position);
                    it != plan.slaves.end() && s.properties.shared())
                    it->second.startup_parameters.clear();
            auto activation = this->master->activate(plan, start_cycle);
            err = activation.err;
            if (!err) break;
            refused = std::move(activation.refusals);
            // A slave with no process data, such as a coupler, gives a task nothing to
            // run.
            const bool io_left = std::ranges::any_of(
                this->image.offsets,
                [&refused](const auto &offset) {
                    return !refused.contains(offset.first.slave_position);
                }
            );
            if (this->bus.running())
                this->stop_locked();
            else
                this->master->deactivate();
            if (!refused.empty() && io_left) {
                for (const auto &[pos, refusal]: refused) {
                    LOG(WARNING)
                        << "[ethercat] starting " << this->iface << " without slave "
                        << pos << ": it " << refusal.reason;
                    plan.slaves[pos].enabled = false;
                }
                for (const auto &s: this->slaves)
                    if (const auto it = refused.find(s.properties.position);
                        it != refused.end())
                        this->left_out[it->first] = {
                            .name = s.properties.name,
                            .refusal = it->second,
                        };
                continue;
            }
        }
        const auto now = clock.steady();
        longest_try = std::max(longest_try, x::telem::TimeSpan(now - try_started));
        const auto next_try_end = x::telem::TimeSpan(now - started) +
                                  this->config.start_retry.max_interval + longest_try;
        // A retry cannot clear a refused state or a slave config that cannot run. A try
        // that ends after start_timeout hides its error behind the task timeout.
        if (err.matches(errors::STATE_CHANGE_REFUSED) ||
            err.matches(errors::SLAVE_CONFIG_ERROR) ||
            next_try_end > this->config.start_timeout || !retry.wait(err)) {
            retry.stop();
            return start_error(err, refused, this->left_out);
        }
    }
    retry.stop();
    this->slaves = this->image.slaves;
    this->starting.store(false, std::memory_order_release);
    this->supervisor_thread = std::thread(&Engine::supervise, this);
    return x::errors::NIL;
}

void Engine::stop_locked() {
    if (!this->bus.stop()) return;
    this->stops++;
    this->cycle_thread.join();
    if (this->supervisor_thread.joinable()) this->supervisor_thread.join();
    this->master->deactivate();
    this->image = {};
    this->cycle_mhz.store(0, std::memory_order_release);
    {
        std::lock_guard lock(this->schedule_mu);
        this->schedule.reset();
        this->retired.reset();
    }
    std::lock_guard lock(this->health_mu);
    this->slave_health.clear();
}

std::pair<std::map<uint16_t, slave::Config>, x::errors::Error>
Engine::read_slave_configs(std::unique_lock<std::mutex> &lock) const {
    while (!this->running()) {
        const auto stops = this->stops;
        lock.unlock();
        auto result = this->config.slave_configs();
        lock.lock();
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
                "The bus on {0} cycles at {1} Hz while a slave runs DC sync, and the "
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
            ". Slave {} ({}) is not in OP: {}",
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
        std::erase(this->outputs, output);
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
                                "Another task sets {} of slave {} to {}. Stop it, or "
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
    const Settings &settings
) {
    auto [configs, configs_err] = this->read_slave_configs(lock);
    if (auto err = this->check_settings_locked(settings)) return {{}, err};
    auto reads = this->read_rates_locked();
    auto writes = this->write_rates_locked();
    (output ? writes : reads).push_back(rate);
    const auto [mhz, rate_err] = this->rate_locked(reads, writes);
    if (rate_err) return {{}, rate_err};
    const bool started = !this->running();
    if (started) {
        if (configs_err) return {{}, configs_err};
        for (const auto &[pos, params]: settings)
            configs[pos].add_startup_parameters(params);
        if (auto err = this->start_locked(mhz, std::move(configs)); err)
            return {{}, err};
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
    return {{.mhz = mhz, .offsets = std::move(offsets)}, x::errors::NIL};
}

std::pair<std::unique_ptr<Engine::Reader>, x::errors::Error> Engine::open_reader(
    const std::vector<pdo::Entry> &entries,
    const x::telem::Rate sample_rate,
    const size_t samples_per_read,
    const Settings &settings
) {
    this->opens.fetch_add(1, std::memory_order_acq_rel);
    x::defer::defer done([this] {
        this->opens.fetch_sub(1, std::memory_order_acq_rel);
    });
    std::unique_lock lock(this->mu);
    auto [joined, err] = this->join_locked(lock, entries, sample_rate, false, settings);
    if (err) return {nullptr, err};
    auto input = std::make_shared<Input>(
        entries,
        joined.offsets,
        sample_rate,
        samples_per_read
    );
    input->settings = settings;
    this->inputs.push_back(input);
    this->publish_schedule_locked(joined.mhz);
    return {std::make_unique<Reader>(*this, std::move(input)), x::errors::NIL};
}

std::pair<std::unique_ptr<Engine::Writer>, x::errors::Error> Engine::open_writer(
    const std::vector<Writer::Entry> &entries,
    const x::telem::Rate execution_rate
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
    std::unique_lock lock(this->mu);
    auto [joined, err] = this->join_locked(lock, pdo_entries, execution_rate, true, {});
    if (err) return {nullptr, err};
    auto output = std::make_shared<Output>(Output{
        .entries = std::move(pdo_entries),
        .rate = execution_rate,
        .offsets = std::move(joined.offsets),
        .stop_values = std::move(stop_values),
    });
    this->outputs.push_back(output);
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
    if (this->running()) return {this->slaves, x::errors::NIL};
    auto [slaves, err] = this->master->scan();
    if (err) return {{}, err};
    const auto [count, count_err] = this->master->count_slaves();
    if (count_err) return {{}, count_err};
    if (count != slaves.size()) {
        LOG(INFO) << "[ethercat] " << count << " slaves answer on " << this->iface
                  << " where " << slaves.size() << " were discovered, rediscovering";
        return this->rediscover_locked();
    }
    this->slaves = slaves;
    return {std::move(slaves), x::errors::NIL};
}

std::pair<std::vector<slave::DiscoveryResult>, x::errors::Error> Engine::rediscover() {
    std::lock_guard lock(this->mu);
    if (this->running()) return {this->slaves, x::errors::NIL};
    return this->rediscover_locked();
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
    const auto [slaves, err] = this->master->scan();
    if (err) return err;
    const auto found = std::ranges::find_if(slaves, [&slave](const auto &s) {
        const auto &p = s.properties;
        return p.vendor_id == slave.vendor_id && p.product_code == slave.product_code &&
               p.serial == slave.serial &&
               (slave.serial != 0 || p.position == slave.position);
    });
    if (found == slaves.end())
        return x::errors::Error(
            errors::SLAVE_CONFIG_ERROR,
            slave.name + " is not on " + this->network_name +
                ". Check its power and cabling."
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
    this->master->deactivate();
    auto [slaves, err] = this->master->scan();
    if (err) return {{}, err};
    this->slaves = slaves;
    return {std::move(slaves), x::errors::NIL};
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
