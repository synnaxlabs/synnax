// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//
// Created by Synnax on 3/24/2024.
//

#ifndef DRIVER_NI_SCANNER_H
#define DRIVER_NI_SCANNER_H

#endif //DRIVER_NI_SCANNER_H

#include "nlohmann/json.hpp"

using json = nlohmann::json;

namespace ni {
    class NiScanner {
    public:
        NiScanner();
        ~NiScanner();
        json getDevices();
    private:

    };
}

