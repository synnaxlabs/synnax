// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Alert, alertIngester, AlertSelectable } from "@/feature/pagerduty/task/Alert";
import { ALERT_TYPE } from "@/feature/pagerduty/task/types";
import { type Import } from "@/platform/import";
import { type Selector } from "@/platform/selector";
import { type Task } from "@/platform/task";

export * from "@/feature/pagerduty/task/Alert";
export * from "@/feature/pagerduty/task/commands";
export * from "@/feature/pagerduty/task/types";

export const FILE_INGESTERS: Import.FileIngesters = {
  [ALERT_TYPE]: alertIngester,
};

export const FORMS: Task.Forms = { [ALERT_TYPE]: Alert };

export const SELECTABLES: Selector.Selectable[] = [AlertSelectable];
