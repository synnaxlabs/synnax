// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Export } from "@/export";
import { Common } from "@/hardware/common";
import { Alert, ALERT_LAYOUT } from "@/hardware/pagerduty/task/Alert";
import { ALERT_SCHEMAS, ALERT_TYPE } from "@/hardware/pagerduty/task/types";
import { type Import } from "@/import";
import { type Layout } from "@/layout";

export * from "@/hardware/pagerduty/task/Alert";
export * from "@/hardware/pagerduty/task/palette";
export * from "@/hardware/pagerduty/task/types";

export const EXTRACTORS: Export.Extractors = { [ALERT_TYPE]: Common.Task.extract };

export const FILE_INGESTERS: Import.FileIngesters = {
  [ALERT_TYPE]: Common.Task.createIngester(ALERT_SCHEMAS.config, ALERT_LAYOUT),
};

export const LAYOUTS: Record<string, Layout.Renderer> = { [ALERT_TYPE]: Alert };

export const ZERO_LAYOUTS: Record<string, Common.Task.Layout> = {
  [ALERT_TYPE]: ALERT_LAYOUT,
};
