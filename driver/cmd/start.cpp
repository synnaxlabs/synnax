// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// external
#include "glog/logging.h"

/// module
#include "x/cpp/xshutdown/xshutdown.h"
#include "x/cpp/xlog/xlog.h"

/// internal
#include "driver/cmd/cmd.h"
#include "driver/rack/rack.h"

int cmd::sub::start(int argc, char *argv[]) {
    LOG(INFO) << xlog::BLUE << "[driver] starting synnax driver " << cmd::version() << xlog::RESET;
    rack::Rack r;
    r.start(argc, argv);
    xshutdown::Listen::listen();
    LOG(INFO) << xlog::BLUE << "[driver] received shutdown signal. stopping driver" << xlog::RESET;
    if (auto err = r.stop())
        LOG(FATAL) << "[driver] stopped with error: " << err;
    else LOG(INFO) << xlog::BLUE << "[driver] stopped" << xlog::RESET;
    return 0;
}
