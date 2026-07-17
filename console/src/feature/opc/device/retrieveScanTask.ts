// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax, type task } from "@synnaxlabs/client";

import { type SCAN_SCHEMAS, SCAN_TYPE } from "@/feature/opc/task/types";

export const retrieveScanTask = async (
  client: Synnax,
  rack: number,
): Promise<task.Task<typeof SCAN_SCHEMAS>> =>
  // The shared cache holds untyped tasks; the scan task's schemas are known.
  (await client.tasks.retrieve({
    type: SCAN_TYPE,
    rack,
  })) as unknown as task.Task<typeof SCAN_SCHEMAS>;
