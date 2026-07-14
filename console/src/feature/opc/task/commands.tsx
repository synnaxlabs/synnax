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

import { useCreateRead } from "@/feature/opc/task/Read";
import { useCreateWrite } from "@/feature/opc/task/Write";
import { Command } from "@/platform/command";

const useVisible = () => Access.useCreateGranted(task.TYPE_ONTOLOGY_ID);

const CreateReadCommand = Command.create({
  key: "opc_ua_create_read_task",
  name: "Create an OPC UA Read Task",
  icon: <Icon.Logo.OPC />,
  useOnSelect: useCreateRead,
  useVisible,
});

const CreateWriteCommand = Command.create({
  key: "opc_ua_create_write_task",
  name: "Create an OPC UA Write Task",
  icon: <Icon.Logo.OPC />,
  useOnSelect: useCreateWrite,
  useVisible,
});

export const COMMANDS = [CreateReadCommand, CreateWriteCommand];
