// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { READ_TYPE, WRITE_TYPE } from "@/feature/ethercat/task/types";
import { Panel } from "@/platform/panel";
import { type Task } from "@/platform/task";

export * from "@/feature/ethercat/task/commands";
export * from "@/feature/ethercat/task/create";
export * from "@/feature/ethercat/task/integration";
export * from "@/feature/ethercat/task/types";

export const Read = Panel.lazyComponent(
  () => import("@/feature/ethercat/task/Read"),
  "Read",
);

export const Write = Panel.lazyComponent(
  () => import("@/feature/ethercat/task/Write"),
  "Write",
);

export const FORMS: Task.Forms = {
  [READ_TYPE]: Read,
  [WRITE_TYPE]: Write,
};
