// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

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

        virtual void stop() {};

        virtual ~Module() = default;
    };

    class Factory {
    public:
        virtual std::unique_ptr<Module> createModule(const std::shared_ptr<synnax::Synnax> &client,
                                                    const json &config,
                                                    bool &valid_config,
                                                    json &config_err) = 0;

        virtual ~Factory() = default;
    };
}



//class NiDigitalReader : module{
//    public:
//};
//
//class NiDigitalWriter : module{
//    public:
//};