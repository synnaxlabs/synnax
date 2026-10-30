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

/// @brief returns the slowest bus rate that is a multiple of every read rate, at least
/// every write rate, and at least MIN_CYCLE_RATE.
std::pair<x::telem::Rate, x::errors::Error> bus_rate(
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
            x::telem::Rate(0),
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
                x::telem::Rate(0),
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
    if (rate <= max) return {x::telem::Rate(static_cast<double>(rate) / 1000), {}};
    std::string rates;
    for (const auto &r: reads)
        rates += (rates.empty() ? "" : ", ") + std::format("{} Hz", r.hz());
    return {
        x::telem::Rate(0),
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
    uint64_t receive_failures = 0;
    uint64_t send_failures = 0;
    x::telem::TimeStamp sent_at{};
    Clock clock;

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
                LOG(INFO) << "[ethercat] bus on " << iface << " cycles at "
                          << sched->rate;
                this->apply_rt(sched->rate.period());
                timer = x::loop::Timer(sched->rate.period());
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
        if (err) {
            if (receive_failures++ == 0) {
                LOG(WARNING) << "[ethercat] cycle failed on " << iface << ": "
                             << err.message();
                std::lock_guard lock(this->cycle_err_mu);
                this->cycle_err = err;
            }
        } else {
            if (const auto dc = this->master->dc_time())
                latched_at = clock.map(*dc, sent_at);
            if (receive_failures > 0) {
                LOG(INFO) << "[ethercat] cycles recovered on " << iface << " after "
                          << receive_failures << " failures";
                receive_failures = 0;
            }
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

        this->write_outputs(false);
        send();
    }
    // Sends the stop values of the last writer to close before the slaves leave OP.
    this->write_outputs(true);
    send();
    if (const auto err = this->master->receive())
        LOG(WARNING) << "[ethercat] last cycle failed on " << iface << ": "
                     << err.message();
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

x::errors::Error Engine::start_locked(const x::telem::Rate rate) {
    x::breaker::Breaker retry(this->config.start_retry);
    retry.start();
    while (true) {
        auto err = this->master->initialize();
        if (!err) {
            err = this->master->activate();
            if (!err) break;
            this->master->deactivate();
        }
        if (!retry.wait(err)) {
            retry.stop();
            return err;
        }
    }
    retry.stop();
    {
        std::lock_guard lock(this->write_mu);
        const auto size = this->master->output_data().size();
        this->write_staging.assign(size, 0);
        this->write_active.assign(size, 0);
    }
    this->publish_schedule_locked(rate);
    this->bus.start();
    this->cycle_thread = std::thread(&Engine::run, this);
    this->supervisor_thread = std::thread(&Engine::supervise, this);
    return x::errors::NIL;
}

void Engine::stop_locked() {
    if (!this->bus.stop()) return;
    this->cycle_thread.join();
    this->supervisor_thread.join();
    this->master->deactivate();
    this->cycle_hz.store(0, std::memory_order_release);
    {
        std::lock_guard lock(this->schedule_mu);
        this->schedule.reset();
        this->retired.reset();
    }
    std::lock_guard lock(this->health_mu);
    this->slave_health.clear();
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

void Engine::publish_schedule_locked(const x::telem::Rate rate) {
    auto next = std::make_shared<Schedule>();
    next->rate = rate;
    for (const auto &input: this->inputs)
        next->inputs.emplace_back(input, millihertz(rate) / millihertz(input->rate));
    std::shared_ptr<const Schedule> retired;
    {
        std::lock_guard lock(this->schedule_mu);
        retired = std::move(this->retired);
        this->schedule = std::move(next);
        this->schedule_version.fetch_add(1, std::memory_order_release);
    }
    this->cycle_hz.store(rate.hz(), std::memory_order_release);
}

void Engine::reschedule_locked() {
    if (this->inputs.empty() && this->outputs.empty()) {
        this->stop_locked();
        return;
    }
    // Rates that had a bus rate keep one when a task leaves.
    const auto rate = bus_rate(
        this->read_rates_locked(),
        this->write_rates_locked(),
        this->master->interface_name()
    );
    this->publish_schedule_locked(rate.first);
}

void Engine::write_outputs(const bool wait) {
    std::unique_lock lock(this->write_mu, std::try_to_lock);
    if (!lock.owns_lock() && wait) lock.lock();
    if (lock.owns_lock()) this->write_active = this->write_staging;
    const auto outputs = this->master->output_data();
    std::memcpy(
        outputs.data(),
        this->write_active.data(),
        std::min(outputs.size(), this->write_active.size())
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
    std::lock_guard lock(this->mu);
    auto reads = this->read_rates_locked();
    reads.push_back(sample_rate);
    const auto [rate, rate_err] = bus_rate(
        reads,
        this->write_rates_locked(),
        this->master->interface_name()
    );
    if (rate_err) return {nullptr, rate_err};
    const bool started = !this->running();
    if (started)
        if (auto err = this->start_locked(rate); err) return {nullptr, err};
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
    this->publish_schedule_locked(rate);
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
    std::lock_guard lock(this->mu);
    auto writes = this->write_rates_locked();
    writes.push_back(execution_rate);
    const auto [rate, rate_err] = bus_rate(
        this->read_rates_locked(),
        writes,
        this->master->interface_name()
    );
    if (rate_err) return {nullptr, rate_err};
    const bool started = !this->running();
    if (started)
        if (auto err = this->start_locked(rate); err) return {nullptr, err};
    auto [offsets, err] = resolve_offsets(
        *this->master,
        pdo_entries,
        this->master->output_data().size()
    );
    if (err) {
        if (started) this->stop_locked();
        return {nullptr, err};
    }
    auto output = std::make_shared<Output>(Output{
        .entries = std::move(pdo_entries),
        .rate = execution_rate,
        .offsets = std::move(offsets),
        .stop_values = std::move(stop_values),
    });
    this->outputs.push_back(output);
    this->publish_schedule_locked(rate);
    return {std::make_unique<Writer>(*this, std::move(output)), x::errors::NIL};
}

x::telem::Rate Engine::cycle_rate() const {
    return x::telem::Rate(this->cycle_hz.load(std::memory_order_acquire));
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

void Engine::set_slave_enabled(const uint16_t position, const bool enabled) {
    this->master->set_slave_enabled(position, enabled);
}
}
