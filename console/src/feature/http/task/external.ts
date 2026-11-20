// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { READ_TYPE, WRITE_TYPE } from "@/feature/http/task/types";
import { Panel } from "@/platform/panel";
import type { Task } from "@/platform/task";

export * from "@/feature/http/task/commands";
export * from "@/feature/http/task/create";
export * from "@/feature/http/task/integration";
export * from "@/feature/http/task/types";

export const FORMS: Task.Forms = {
  [READ_TYPE]: Panel.lazyComponent(() => import("@/feature/http/task/Read"), "Read"),
  [WRITE_TYPE]: Panel.lazyComponent(() => import("@/feature/http/task/Write"), "Write"),
};
