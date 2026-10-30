// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack, type Synnax, type task } from "@synnaxlabs/client";

import { SCAN_SCHEMAS, SCAN_TYPE } from "@/feature/mqtt/task/types";

export const retrieveScanTask = async (
  client: Synnax,
  rack: rack.Key,
): Promise<task.Task<typeof SCAN_SCHEMAS>> =>
  await client.tasks.retrieve({ type: SCAN_TYPE, rack, schemas: SCAN_SCHEMAS });
