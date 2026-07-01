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

import { Palette } from "@/component/palette";
import { READ_LAYOUT } from "@/service/ethercat/task/Read";
import { WRITE_LAYOUT } from "@/service/ethercat/task/Write";

const useVisible = () => Access.useCreateGranted(task.TYPE_ONTOLOGY_ID);

const CreateReadCommand = Palette.createSimpleCommand({
  key: "ethercat_create_read_task",
  name: "Create an EtherCAT Read Task",
  icon: <Icon.Logo.EtherCAT />,
  layout: READ_LAYOUT,
  useVisible,
});

const CreateWriteCommand = Palette.createSimpleCommand({
  key: "ethercat_create_write_task",
  name: "Create an EtherCAT Write Task",
  icon: <Icon.Logo.EtherCAT />,
  layout: WRITE_LAYOUT,
  useVisible,
});

export const COMMANDS = [CreateReadCommand, CreateWriteCommand];
