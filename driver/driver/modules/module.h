// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//
// Created by Emiliano Bonilla on 1/5/24.
//

#include "synnax/synnax.h"
#include <memory>


namespace module {
class Module {
private:
    synnax::Module internal;
public:
    Module(const synnax::Module &module) : internal(module) {}
};

class Factory {
public:
    virtual std::pair<std::unique_ptr<Module>, freighter::Error> configure(
            const std::shared_ptr<synnax::Synnax> &client, const synnax::Module &module) = 0;
};
}