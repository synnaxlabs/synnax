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

import { READ_TYPE, WRITE_TYPE } from "@/feature/ethercat/task/types";
import { Command } from "@/platform/command";
import { Task } from "@/platform/task";

const useVisible = () => Access.useCreateGranted(task.TYPE_ONTOLOGY_ID);

const CreateReadCommand = Command.create({
  key: "ethercat_create_read_task",
  name: "Create an EtherCAT Read Task",
  icon: <Icon.Logo.EtherCAT />,
  useOnSelect: Task.createOpenTab(READ_TYPE),
  useVisible,
});

const CreateWriteCommand = Command.create({
  key: "ethercat_create_write_task",
  name: "Create an EtherCAT Write Task",
  icon: <Icon.Logo.EtherCAT />,
  useOnSelect: Task.createOpenTab(WRITE_TYPE),
  useVisible,
});

export const COMMANDS = [CreateReadCommand, CreateWriteCommand];
