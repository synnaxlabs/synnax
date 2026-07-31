// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

import { ALERT_TYPE } from "@/feature/pagerduty/task/types";
import { Task } from "@/platform/task";

const CreateAlertCommand = Task.createCommand({
  key: "pagerduty_create_alert_task",
  name: "Create a PagerDuty Alert Task",
  icon: <Icon.Logo.PagerDuty />,
  type: ALERT_TYPE,
});

export const COMMANDS = [CreateAlertCommand];
