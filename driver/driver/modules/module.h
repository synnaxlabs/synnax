// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "synnax/synnax.h"
#include <memory>
#include <utility>
#include "nlohmann/json.hpp"

using json = nlohmann::json;


namespace module {
class Module {
private:
    synnax::Module internal;
public:
    Module(synnax::Module module) : internal(std::move(module)) {}

    virtual void stop() = 0;

    virtual ~Module() = default;
};

class Factory {
public:
    virtual std::unique_ptr<Module> configure(
            const std::shared_ptr<synnax::Synnax> &client, const synnax::Module &module, bool &valid_config, const json &config_err) = 0;

    virtual ~Factory() = default;
};
}