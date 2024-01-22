// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "daqmx.h"
#include "synnax/synnax.h"
#include <string>
#include <vector>

namespace ni {
class Reader {
private:
    TaskHandle task;

    Reader();
public:
    std::pair<synnax::Frame, freighter::Error> read();

    freighter::Error configure(synnax::Module config);

    freighter::Error stop();
};

}
class Factory {
};