// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstddef>
#include <cstdint>
#include <utility>

#include "x/cpp/xerrors/errors.h"

#include "driver/ethercat/master/slave_info.h"

namespace ethercat {
/// Abstract interface for an EtherCAT domain.
///
/// A domain manages process data (PDO) exchange between the master and slaves.
/// PDO entries must be registered before the master is activated. After activation,
/// the domain's data buffer can be accessed for reading inputs and writing outputs.
///
/// Thread safety: Implementations must be thread-safe for concurrent calls to data()
/// and registered PDO offsets, but register_pdo() is only safe during configuration.
class Domain {
public:
    virtual ~Domain() = default;

    /// Registers a PDO entry for cyclic exchange in this domain.
    ///
    /// Must be called before master activation. Each registered entry reserves space
    /// in the domain's data buffer. The returned offset indicates where in the buffer
    /// this entry's data will be located after activation.
    ///
    /// @param entry PDO entry describing the slave position, CoE index, subindex,
    ///              bit length, and direction (input/output).
    /// @returns A pair containing:
    ///          - size_t: Byte offset into data() where this entry's data resides.
    ///          - xerrors::Error: PDO_MAPPING_ERROR if registration fails.
    [[nodiscard]] virtual std::pair<size_t, xerrors::Error>
    register_pdo(const PDOEntry &entry) = 0;

    /// Returns a pointer to the domain's process data buffer.
    ///
    /// The buffer layout is determined by the registered PDO entries. Input data
    /// (from slaves) is valid after receive()/process(). Output data should be
    /// written before queue()/send().
    ///
    /// @returns Pointer to the raw process data buffer, or nullptr if not activated.
    [[nodiscard]] virtual uint8_t *data() = 0;

    /// Returns the total size of the domain's process data buffer in bytes.
    [[nodiscard]] virtual size_t size() const = 0;

    /// Returns the size of input data (TxPDO, slave→master) in bytes.
    [[nodiscard]] virtual size_t input_size() const = 0;

    /// Returns the size of output data (RxPDO, master→slave) in bytes.
    [[nodiscard]] virtual size_t output_size() const = 0;
};
}
