// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

import { useCreateRead } from "@/feature/ethercat/task/Read";
import { useCreateWrite } from "@/feature/ethercat/task/Write";
import { Task } from "@/platform/task";

const CreateReadCommand = Task.createCommand({
  key: "ethercat_create_read_task",
  name: "Create an EtherCAT Read Task",
  icon: <Icon.Logo.EtherCAT />,
  useOnSelect: useCreateRead,
});

const CreateWriteCommand = Task.createCommand({
  key: "ethercat_create_write_task",
  name: "Create an EtherCAT Write Task",
  icon: <Icon.Logo.EtherCAT />,
  useOnSelect: useCreateWrite,
});

export const COMMANDS = [CreateReadCommand, CreateWriteCommand];
