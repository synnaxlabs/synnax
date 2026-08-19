// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

import { useCreateAlert } from "@/feature/pagerduty/task/Alert";
import { Task } from "@/platform/task";

const CreateAlertCommand = Task.createCommand({
  key: "pagerduty_create_alert_task",
  name: "Create PagerDuty alert task",
  icon: <Icon.Logo.PagerDuty />,
  useOnSelect: useCreateAlert,
});

export const COMMANDS = [CreateAlertCommand];
