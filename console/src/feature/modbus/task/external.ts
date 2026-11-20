// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { READ_TYPE, WRITE_TYPE } from "@/feature/modbus/task/types";
import { Panel } from "@/platform/panel";
import { type Task } from "@/platform/task";

export * from "@/feature/modbus/task/commands";
export * from "@/feature/modbus/task/create";
export * from "@/feature/modbus/task/integration";
export * from "@/feature/modbus/task/types";

export const FORMS: Task.Forms = {
  [READ_TYPE]: Panel.lazyComponent(() => import("@/feature/modbus/task/Read"), "Read"),
  [WRITE_TYPE]: Panel.lazyComponent(
    () => import("@/feature/modbus/task/Write"),
    "Write",
  ),
};
