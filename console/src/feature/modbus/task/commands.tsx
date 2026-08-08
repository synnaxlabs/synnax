// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { Access, Icon } from "@synnaxlabs/pluto";

import { useCreateRead } from "@/feature/modbus/task/Read";
import { useCreateWrite } from "@/feature/modbus/task/Write";
import { Command } from "@/platform/command";

const useVisible = () => Access.useCreateGranted(task.TYPE_ONTOLOGY_ID);

const CreateReadCommand = Command.create({
  key: "modbus_create_read_task",
  name: "Create a Modbus Read Task",
  icon: <Icon.Logo.Modbus />,
  useOnSelect: useCreateRead,
  useVisible,
});

const CreateWriteCommand = Command.create({
  key: "modbus_create_write_task",
  name: "Create a Modbus Write Task",
  icon: <Icon.Logo.Modbus />,
  useOnSelect: useCreateWrite,
  useVisible,
});

export const COMMANDS = [CreateReadCommand, CreateWriteCommand];
