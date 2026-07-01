// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Layout } from "@/component/layout";
import { type Selector } from "@/component/selector";
import { type Export } from "@/service/export";
import { type Import } from "@/service/import";
import { Alert, ALERT_LAYOUT, AlertSelectable } from "@/service/pagerduty/task/Alert";
import { ALERT_SCHEMAS, ALERT_TYPE } from "@/service/pagerduty/task/types";
import { createIngester } from "@/service/task/createIngester";
import { extract } from "@/service/task/export";
import { type Layout as TaskLayout } from "@/service/task/Form";

export * from "@/service/pagerduty/task/Alert";
export * from "@/service/pagerduty/task/palette";
export * from "@/service/pagerduty/task/types";

export const EXTRACTORS: Export.Extractors = { [ALERT_TYPE]: extract };

export const FILE_INGESTERS: Import.FileIngesters = {
  [ALERT_TYPE]: createIngester(ALERT_SCHEMAS.config, ALERT_LAYOUT),
};

export const LAYOUTS: Record<string, Layout.Renderer> = { [ALERT_TYPE]: Alert };

export const SELECTABLES: Selector.Selectable[] = [AlertSelectable];

export const ZERO_LAYOUTS: Record<string, TaskLayout> = {
  [ALERT_TYPE]: ALERT_LAYOUT,
};
