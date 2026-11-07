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
    const std::string &interface_name
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
                    "a write task on {} runs at {} Hz, above the maximum bus rate of "
                    "{} Hz",
                    interface_name,
                    static_cast<double>(floor) / 1000,
                    MAX_CYCLE_RATE.hz()
                )
            )
        };
    uint64_t base = 1;
    for (const auto &rate: reads) {
        const auto mhz = millihertz(rate);
        // Allow the rounding error of the float a Rate holds.
        const double error = std::abs(rate.hz() * 1000 - static_cast<double>(mhz));
        if (mhz == 0 || error > static_cast<double>(mhz) * 1e-6)
            return {
                0,
                x::errors::Error(
                    errors::RATE_MISMATCH,
                    std::format(
                        "a read task on {} samples at {} Hz, but sample rates must be "
                        "a positive multiple of 0.001 Hz",
                        interface_name,
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
                "no bus rate at or below {} Hz on {} is a multiple of every read task "
                "sample rate ({}). Choose sample rates with a smaller common multiple.",
                MAX_CYCLE_RATE.hz(),
                interface_name,
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

/// @brief looks up the offset of each entry in a process image of the given size.
std::pair<std::vector<pdo::Offset>, x::errors::Error> resolve_offsets(
    const master::Master &master,
    const std::vector<pdo::Entry> &entries,
    const size_t image_size
) {
    std::vector<pdo::Offset> offsets;
    offsets.reserve(entries.size());
    for (const auto &entry: entries) {
        const auto [offset, err] = master.pdo_offset(entry);
        if (err) return {{}, err};
        if (offset.byte + telem::pdo_required_bytes(offset.bit, entry.bit_length) >
            image_size)
            return {
                {},
                x::errors::Error(
                    errors::PDO_MAPPING_ERROR,
                    std::format(
                        "PDO {} on slave {} lies outside the process image",
                        telem::format_index_sub_index(entry.index, entry.sub_index),
                        entry.slave_position
                    )
                )
            };
        offsets.push_back(offset);
    }
    return {offsets, x::errors::NIL};
}
}

Engine::Input::Input(
    std::vector<pdo::Entry> entries,
    std::vector<pdo::Offset> offsets,
    const x::telem::Rate rate,
    const size_t samples_per_read
):
    entries(std::move(entries)),
    rate(rate),
    samples_per_read(samples_per_read),
    offsets(std::move(offsets)) {
    for (size_t i = 0; i < this->entries.size(); ++i) {
        const auto length = telem::pdo_required_bytes(
            this->offsets[i].bit,
            this->entries[i].bit_length
        );
        this->sample_offsets.push_back(this->sample_size);
        this->lengths.push_back(length);
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
    for (size_t i = 0; i < this->entries.size(); ++i)
        std::memcpy(
            sample + this->sample_offsets[i],
            image + this->offsets[i].byte,
            this->lengths[i]
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

void Engine::apply_rt(const x::telem::TimeSpan period) const {
    if (this->config.rt_handle) {
        this->config.rt_handle->apply();
        return;
    }
    x::thread::rt::Config rt_cfg = this->config.rt;
    if (rt_cfg.enabled && !rt_cfg.has_timing()) {
        rt_cfg.period = period;
        rt_cfg.computation = period * 0.2;
        rt_cfg.deadline = period * 0.8;
    }
    x::thread::rt::apply_config(rt_cfg);
}

void Engine::run() {
    const auto iface = this->master->interface_name();
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
        sent_at = x::telem::TimeStamp::now();
        if (auto err = this->master->send(); err) {
            if (send_failures++ == 0)
                LOG(WARNING) << "[ethercat] send failed on " << iface << ": "
                             << err.message();
        } else if (send_failures > 0) {
            LOG(INFO) << "[ethercat] send recovered on " << iface << " after "
                      << send_failures << " failures";
            send_failures = 0;
        }
    };

    this->write_outputs();
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
                LOG(INFO) << "[ethercat] bus on " << iface << " cycles at "
                          << sched->rate;
                this->apply_rt(sched->rate.period());
                timer = x::loop::Timer(sched->rate.period());
                std::vector<PhaseLock::Sync0> syncs;
                for (const auto &regs: this->master->dc_sync())
                    if (regs.marks_data())
                        syncs.push_back({
                            .cycle = x::telem::TimeSpan(regs.sync0_cycle),
                            .event = master::DC_EPOCH +
                                     x::telem::TimeSpan(regs.sync0_shift),
                        });
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
        auto [elapsed, on_time] = timer.wait();
        if (!this->bus.running()) break;
        if (!on_time && this->config.max_overrun.nanoseconds() > 0)
            VLOG(1) << "[ethercat] cycle overrun: " << elapsed;

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
                LOG(WARNING) << "[ethercat] cycles on " << iface
                             << " run late: a frame passed the reference clock near "
                                "a SYNC0 event";
        }
        if (!err) {
            if (dc) latched_at = clock.map(*dc, sent_at);
            if (receive_failures > 0) {
                LOG(INFO) << "[ethercat] cycles recovered on " << iface << " after "
                          << receive_failures << " failures";
                receive_failures = 0;
            }
        } else if (!shortfall(err) && receive_failures++ == 0) {
            LOG(WARNING) << "[ethercat] cycle failed on " << iface << ": "
                         << err.message();
            std::lock_guard lock(this->cycle_err_mu);
            this->cycle_err = err;
        }
        const uint8_t *image = this->master->input_data().data();
        for (const auto &[input, decimation]: sched->inputs) {
            if (input->countdown > 0) {
                input->countdown--;
                continue;
            }
            input->countdown = decimation - 1;
            if (err)
                input->failed_samples.fetch_add(1, std::memory_order_relaxed);
            else
                input->push(image, latched_at);
        }

        this->write_outputs();
        send();
    }
    // Sends the zeroed outputs of the last writer to close before the slaves leave OP.
    this->write_outputs();
    send();
    if (const auto err = this->master->receive(); err && !shortfall(err))
        LOG(WARNING) << "[ethercat] last cycle failed on " << iface << ": "
                     << err.message();
    if (late_frames > 0)
        LOG(WARNING) << "[ethercat] " << late_frames << " frames on " << iface
                     << " passed the reference clock near a SYNC0 event";
    LOG(INFO) << "[ethercat] bus stopped on " << iface;
}

void Engine::supervise() {
    const auto iface = this->master->interface_name();
    std::vector<slave::Health> last;
    while (this->bus.running()) {
        auto health = this->master->check(this->bus);
        for (const auto &h: health) {
            const auto prev = std::ranges::find(
                last,
                h.position,
                &slave::Health::position
            );
            const bool was_operational = prev == last.end() || prev->operational();
            if (was_operational && !h.operational())
                LOG(WARNING) << "[ethercat] slave " << h.position << " on " << iface
                             << " left OP: " << h.to_string();
            else if (!was_operational && h.operational())
                LOG(INFO) << "[ethercat] slave " << h.position << " on " << iface
                          << " returned to OP";
            else if (!h.operational() && *prev != h)
                LOG(WARNING) << "[ethercat] slave " << h.position << " on " << iface
                             << ": " << h.to_string();
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
    this->master->replace_slave_configs(std::move(configs));
    const auto start_cycle = [this, mhz] {
        {
            std::lock_guard lock(this->write_mu);
            const auto size = this->master->output_data().size();
            this->write_staging.assign(size, 0);
            this->write_active.assign(size, 0);
        }
        this->publish_schedule_locked(mhz);
        this->starting.store(true, std::memory_order_release);
        this->bus.start();
        this->cycle_thread = std::thread(&Engine::run, this);
    };
    x::breaker::Breaker retry(this->config.start_retry);
    retry.start();
    while (true) {
        auto err = this->master->initialize();
        if (!err) {
            err = this->master->activate(
                x::telem::Rate(static_cast<double>(mhz) / 1000).period(),
                start_cycle
            );
            if (!err) break;
            if (this->bus.running())
                this->stop_locked();
            else
                this->master->deactivate();
        }
        // A retry cannot clear a refused state or a slave config that cannot run.
        if (err.matches(errors::STATE_CHANGE_REFUSED) ||
            err.matches(errors::SLAVE_CONFIG_ERROR) || !retry.wait(err)) {
            retry.stop();
            return err;
        }
    }
    retry.stop();
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
    const auto iface = this->master->interface_name();
    auto [mhz, err] = bus_rate(reads, writes, iface);
    if (err || !this->running() || this->master->dc_sync().empty()) return {mhz, err};
    // The SYNC0 and SYNC1 cycles came from the bus period at activation.
    const auto fixed = this->cycle_mhz.load(std::memory_order_acquire);
    if (serves(fixed, reads, writes)) return {fixed, x::errors::NIL};
    return {
        0,
        x::errors::Error(
            errors::RATE_MISMATCH,
            std::format(
                "the bus on {0} cycles at {1} Hz while a slave runs DC sync, and the "
                "rate cannot change until every task on the port stops. Use a sample "
                "rate that {1} Hz is a multiple of, or an execution rate of at most "
                "{1} Hz.",
                iface,
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

void Engine::write_outputs() {
    if (std::unique_lock lock(this->write_mu, std::try_to_lock); lock.owns_lock())
        this->write_active = this->write_staging;
    const auto outputs = this->master->output_data();
    std::memcpy(
        outputs.data(),
        this->write_active.data(),
        std::min(outputs.size(), this->write_active.size())
    );
}

void Engine::rebuild_outputs_locked() {
    struct Applied {
        const pdo::Entry *entry;
        pdo::Offset offset;
        const Command *command;
    };
    std::vector<Applied> applied;
    for (const auto &output: this->outputs)
        for (size_t i = 0; i < output->entries.size(); ++i)
            if (const auto &command = output->commands[i]; command.has_value())
                applied.push_back({&output->entries[i], output->offsets[i], &*command});
    // Two writers can command the same output, so the latest command must land last.
    std::ranges::sort(applied, {}, [](const Applied &a) { return a.command->seq; });
    std::ranges::fill(this->write_staging, 0);
    for (const auto &[entry, offset, command]: applied)
        telem::write_pdo_from_value(
            this->write_staging.data() + offset.byte,
            offset.bit,
            entry->bit_length,
            entry->data_type,
            command->value
        );
}

x::errors::Error Engine::cycle_error() const {
    std::lock_guard lock(this->cycle_err_mu);
    return this->cycle_err;
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
        this->rebuild_outputs_locked();
    }
    this->reschedule_locked();
}

Engine::Engine(std::shared_ptr<master::Master> master, Config config):
    config(std::move(config)), master(std::move(master)) {}

Engine::Engine(std::shared_ptr<master::Master> master):
    Engine(std::move(master), Config{}) {}

Engine::~Engine() {
    std::lock_guard lock(this->mu);
    this->stop_locked();
}

std::pair<std::unique_ptr<Engine::Reader>, x::errors::Error> Engine::open_reader(
    const std::vector<pdo::Entry> &entries,
    const x::telem::Rate sample_rate,
    const size_t samples_per_read
) {
    std::unique_lock lock(this->mu);
    auto [configs, configs_err] = this->read_slave_configs(lock);
    auto reads = this->read_rates_locked();
    reads.push_back(sample_rate);
    const auto [mhz, rate_err] = this->rate_locked(reads, this->write_rates_locked());
    if (rate_err) return {nullptr, rate_err};
    const bool started = !this->running();
    if (started) {
        if (configs_err) return {nullptr, configs_err};
        if (auto err = this->start_locked(mhz, std::move(configs)); err)
            return {nullptr, err};
    }
    auto [offsets, err] = resolve_offsets(
        *this->master,
        entries,
        this->master->input_data().size()
    );
    if (err) {
        if (started) this->stop_locked();
        return {nullptr, err};
    }
    auto input = std::make_shared<Input>(
        entries,
        std::move(offsets),
        sample_rate,
        samples_per_read
    );
    this->inputs.push_back(input);
    this->publish_schedule_locked(mhz);
    return {std::make_unique<Reader>(*this, std::move(input)), x::errors::NIL};
}

std::pair<std::unique_ptr<Engine::Writer>, x::errors::Error> Engine::open_writer(
    const std::vector<pdo::Entry> &entries,
    const x::telem::Rate execution_rate
) {
    std::unique_lock lock(this->mu);
    auto [configs, configs_err] = this->read_slave_configs(lock);
    auto writes = this->write_rates_locked();
    writes.push_back(execution_rate);
    const auto [mhz, rate_err] = this->rate_locked(this->read_rates_locked(), writes);
    if (rate_err) return {nullptr, rate_err};
    const bool started = !this->running();
    if (started) {
        if (configs_err) return {nullptr, configs_err};
        if (auto err = this->start_locked(mhz, std::move(configs)); err)
            return {nullptr, err};
    }
    auto [offsets, err] = resolve_offsets(
        *this->master,
        entries,
        this->master->output_data().size()
    );
    if (err) {
        if (started) this->stop_locked();
        return {nullptr, err};
    }
    auto output = std::make_shared<Output>(Output{
        .entries = entries,
        .rate = execution_rate,
        .offsets = std::move(offsets),
        .commands = std::vector<std::optional<Command>>(entries.size()),
    });
    this->outputs.push_back(output);
    this->publish_schedule_locked(mhz);
    return {std::make_unique<Writer>(*this, std::move(output)), x::errors::NIL};
}

x::telem::Rate Engine::cycle_rate() const {
    return x::telem::Rate(
        static_cast<double>(this->cycle_mhz.load(std::memory_order_acquire)) / 1000
    );
}

x::errors::Error Engine::ensure_initialized() {
    std::lock_guard lock(this->mu);
    if (this->running()) return x::errors::NIL;
    if (auto err = this->master->initialize()) return err;
    const auto [count, err] = this->master->count_slaves();
    if (err) return err;
    const auto discovered = this->master->slaves().size();
    if (count == discovered) return x::errors::NIL;
    LOG(INFO) << "[ethercat] " << count << " slaves answer on "
              << this->master->interface_name() << " where " << discovered
              << " were discovered, rediscovering";
    this->master->deactivate();
    return this->master->initialize();
}

x::errors::Error Engine::rediscover() {
    std::lock_guard lock(this->mu);
    if (this->running()) return x::errors::NIL;
    this->master->deactivate();
    return this->master->initialize();
}

std::vector<slave::DiscoveryResult> Engine::slaves() const {
    std::lock_guard lock(this->mu);
    return this->master->slaves();
}

std::vector<slave::Health> Engine::health() const {
    std::lock_guard lock(this->health_mu);
    return this->slave_health;
}

std::string Engine::interface_name() const {
    return this->master->interface_name();
}
}
