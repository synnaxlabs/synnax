// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

import { READ_TYPE, WRITE_TYPE } from "@/feature/labjack/task/types";
import { Task } from "@/platform/task";

const CreateReadCommand = Task.createCommand({
  key: "labjack_create_read_task",
  name: "Create a LabJack Read Task",
  icon: <Icon.Logo.LabJack />,
  type: READ_TYPE,
});

const CreateWriteCommand = Task.createCommand({
  key: "labjack_create_write_task",
  name: "Create a LabJack Write Task",
  icon: <Icon.Logo.LabJack />,
  type: WRITE_TYPE,
});

export const COMMANDS = [CreateReadCommand, CreateWriteCommand];
