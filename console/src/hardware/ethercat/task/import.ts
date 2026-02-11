// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Common } from "@/hardware/common";
import { READ_LAYOUT } from "@/hardware/ethercat/task/Read";
import { readConfigZ, writeConfigZ } from "@/hardware/ethercat/task/types";
import { WRITE_LAYOUT } from "@/hardware/ethercat/task/Write";

export const ingestRead = Common.Task.createIngester(readConfigZ, READ_LAYOUT);

export const ingestWrite = Common.Task.createIngester(writeConfigZ, WRITE_LAYOUT);
