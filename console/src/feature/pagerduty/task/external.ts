// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ALERT_TYPE } from "@/feature/pagerduty/task/types";
import { Panel } from "@/platform/panel";
import { type Task } from "@/platform/task";

export * from "@/feature/pagerduty/task/commands";
export * from "@/feature/pagerduty/task/create";
export * from "@/feature/pagerduty/task/integration";
export * from "@/feature/pagerduty/task/types";

export const Alert = Panel.lazyComponent(
  () => import("@/feature/pagerduty/task/Alert"),
  "Alert",
);

export const FORMS: Task.Forms = { [ALERT_TYPE]: Alert };
