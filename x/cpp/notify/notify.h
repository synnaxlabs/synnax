// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>

#include "x/cpp/telem/telem.h"

namespace notify {

struct Notifier {
    virtual ~Notifier() = default;

    virtual void signal() = 0;

    virtual bool wait(telem::TimeSpan timeout = telem::TimeSpan::MAX()) = 0;

    virtual bool poll() = 0;

    [[nodiscard]] virtual int fd() const = 0;
};

std::unique_ptr<Notifier> create();

}
