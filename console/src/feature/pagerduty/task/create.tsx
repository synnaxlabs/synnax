// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import {
  ALERT_SCHEMAS,
  ALERT_TYPE,
  type AlertSchemas,
} from "@/feature/pagerduty/task/types";
import { type Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const getAlertInitialValues: Task.GetInitialValues<AlertSchemas> = ({ config }) => {
  const parsed = ALERT_SCHEMAS.config.safeParse(config ?? {});
  return {
    name: "PagerDuty alert task",
    type: ALERT_TYPE,
    config: parsed.success ? parsed.data : ALERT_SCHEMAS.config.parse({}),
  };
};

export const useCreateAlert = Task.createUseCreate({
  getInitialValues: getAlertInitialValues,
});

export const AlertSelectable: Selector.Selectable = {
  type: ALERT_TYPE,
  title: "PagerDuty alert task",
  label: "Alert",
  icon: <Icon.Logo.PagerDuty />,
  useOnSelect: useCreateAlert,
};
