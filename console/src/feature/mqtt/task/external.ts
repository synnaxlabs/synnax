// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { EDGE_TYPE, READ_TYPE, WRITE_TYPE } from "@/feature/mqtt/task/types";
import { Panel } from "@/platform/panel";
import { type Task } from "@/platform/task";

export * from "@/feature/mqtt/task/commands";
export * from "@/feature/mqtt/task/create";
export * from "@/feature/mqtt/task/integration";
export * from "@/feature/mqtt/task/types";

export const FORMS: Task.Forms = {
  [READ_TYPE]: Panel.lazyComponent(() => import("@/feature/mqtt/task/Read"), "Read"),
  [WRITE_TYPE]: Panel.lazyComponent(() => import("@/feature/mqtt/task/Write"), "Write"),
  [EDGE_TYPE]: Panel.lazyComponent(() => import("@/feature/mqtt/task/Edge"), "Edge"),
};
