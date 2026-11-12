// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <memory>
#include <optional>
#include <string>
#include <unordered_set>
#include <variant>
#include <vector>

#include "client/cpp/ethercat/types.gen.h"
#include "client/cpp/synnax.h"
#include "x/cpp/json/json.h"

#include "driver/ethercat/pdo/pdo.h"
#include "driver/ethercat/slave/slave.h"

namespace driver::ethercat::channel {
/// @brief the objects of a slave that hold the gain and offset of an entry with integer
/// samples. A physical value is raw / divisor * gain + offset, or (raw / divisor +
/// offset) * gain when the offset is raw.
struct Calibration {
    /// @brief the index of the object that holds the gain and the offset.
    uint16_t index = 0;
    /// @brief the sub-index that holds the gain, as a float32.
    uint8_t gain_sub_index = 0;
    /// @brief the sub-index that holds the offset, as a float32.
    uint8_t offset_sub_index = 0;
    /// @brief the number that raw samples are divided by before the gain.
    double divisor = 1;
    /// @brief whether the offset is in raw units, before the gain.
    bool raw_offset = false;

    /// @brief returns the slope and the offset of the line that turns raw samples into
    /// physical values, for the gain and offset that the slave holds.
    [[nodiscard]] std::pair<double, double>
    line(const float gain, const float offset) const {
        const double g = gain;
        return {g / this->divisor, this->raw_offset ? offset * g : offset};
    }
};

/// @brief an enabled channel that reads an input entry (TxPDO, slave->master).
struct Input {
    /// @brief the entry the channel reads.
    pdo::Entry entry;
    /// @brief the key of the Synnax channel to write data to.
    synnax::channel::Key synnax_key = 0;
    /// @brief the Synnax channel, set by the owning task.
    synnax::channel::Channel ch;
    /// @brief the scale applied to raw samples after acquisition.
    ::synnax::task::Scale scale = ::synnax::task::NoneScale{};
    /// @brief the calibration that the catalog gives the integer samples of the entry,
    /// applied before scale, or nullopt for none.
    std::optional<Calibration> calibration;

    /// @brief whether the channel scales its raw samples.
    [[nodiscard]] bool scaled() const {
        return !std::holds_alternative<::synnax::task::NoneScale>(this->scale);
    }
};

/// @brief an enabled channel that writes an output entry (RxPDO, master->slave).
struct Output {
    /// @brief the entry the channel writes.
    pdo::Entry entry;
    /// @brief the key of the Synnax channel to receive commands from.
    synnax::channel::Key command_key = 0;
    /// @brief the key of the Synnax channel to write state feedback to.
    synnax::channel::Key state_key = 0;
    /// @brief the state channel, set by the owning task when state_key is set.
    std::optional<synnax::channel::Channel> state;
    /// @brief the value the entry takes when the task stops.
    double stop_value = 0;
};

/// @brief returns the input channel that cfg configures on slave. A calibrated channel
/// on an entry with integer samples takes the calibration and the data type that the
/// catalog of slave gives the entry. Accumulates a field error on parser and returns
/// nullptr when cfg does not address one entry on slave, its data type does not fit the
/// entry, or the catalog gives such a channel no calibration that fits the entry.
std::unique_ptr<Input> make_input(
    const ::synnax::ethercat::ReadChannel &cfg,
    const slave::Properties &slave,
    const x::json::Parser &parser
);

/// @brief returns the output channel that cfg configures on slave. Accumulates a field
/// error under path and returns nullptr when cfg does not address one entry on slave,
/// or its data type does not fit the entry. Accumulates a field error when its stop
/// value does not fit the entry.
std::unique_ptr<Output> make_output(
    const ::synnax::ethercat::WriteChannel &cfg,
    const slave::Properties &slave,
    const x::json::Parser &parser,
    const std::string &path = ""
);

/// @brief tracks the entries a task's channels use, so that no two channels share one.
class Claims {
    std::unordered_set<pdo::Key, pdo::KeyHash> keys;

public:
    /// @brief claims entry, accumulating a field error under path when another channel
    /// holds it.
    void claim(
        const pdo::Entry &entry,
        const slave::Properties &slave,
        const x::json::Parser &parser,
        const std::string &path = ""
    );
};

/// @brief sorts channels by slave position, then by index.
template<typename ChannelPtr>
void sort_by_position(std::vector<ChannelPtr> &channels) {
    std::ranges::sort(channels, [](const auto &a, const auto &b) {
        if (a->entry.slave_position != b->entry.slave_position)
            return a->entry.slave_position < b->entry.slave_position;
        return a->entry.index < b->entry.index;
    });
}
}
