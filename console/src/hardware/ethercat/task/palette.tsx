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

import { READ_LAYOUT } from "@/hardware/ethercat/task/Read";
import { WRITE_LAYOUT } from "@/hardware/ethercat/task/Write";
import { Palette } from "@/palette";

const useVisible = () => Access.useUpdateGranted(task.TYPE_ONTOLOGY_ID);

export const CreateReadCommand = Palette.createSimpleCommand({
  key: "ethercat-create-read-task",
  name: "Create an EtherCAT Read Task",
  icon: <Icon.Logo.EtherCAT />,
  layout: READ_LAYOUT,
  useVisible,
});

export const CreateWriteCommand = Palette.createSimpleCommand({
  key: "ethercat-create-write-task",
  name: "Create an EtherCAT Write Task",
  icon: <Icon.Logo.EtherCAT />,
  layout: WRITE_LAYOUT,
  useVisible,
});

export const COMMANDS = [CreateReadCommand, CreateWriteCommand];
