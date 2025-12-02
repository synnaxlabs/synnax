// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { Access, Icon } from "@synnaxlabs/pluto";

import { importRead, importWrite } from "@/hardware/labjack/task/import";
import { READ_LAYOUT } from "@/hardware/labjack/task/Read";
import { WRITE_LAYOUT } from "@/hardware/labjack/task/Write";
import { type Palette } from "@/palette";

const visibleFilter = ({ store, client }: Palette.CommandVisibleContext) =>
  Access.editGranted({ id: task.ontologyID(""), store, client });

const CREATE_READ_COMMAND: Palette.Command = {
  key: "labjack-create-read-task",
  name: "Create a LabJack Read Task",
  icon: <Icon.Logo.LabJack />,
  onSelect: ({ placeLayout }) => placeLayout(READ_LAYOUT),
  visible: visibleFilter,
};

const CREATE_WRITE_COMMAND: Palette.Command = {
  key: "labjack-create-write-task",
  name: "Create a LabJack Write Task",
  icon: <Icon.Logo.LabJack />,
  onSelect: ({ placeLayout }) => placeLayout(WRITE_LAYOUT),
  visible: visibleFilter,
};

const IMPORT_READ_COMMAND: Palette.Command = {
  key: "labjack-import-read-task",
  name: "Import LabJack Read Task(s)",
  sortOrder: -1,
  icon: <Icon.Logo.LabJack />,
  onSelect: importRead,
  visible: visibleFilter,
};

const IMPORT_WRITE_COMMAND: Palette.Command = {
  key: "labjack-import-write-task",
  name: "Import LabJack Write Task(s)",
  sortOrder: -1,
  icon: <Icon.Logo.LabJack />,
  onSelect: importWrite,
  visible: visibleFilter,
};

export const COMMANDS = [
  CREATE_READ_COMMAND,
  CREATE_WRITE_COMMAND,
  IMPORT_READ_COMMAND,
  IMPORT_WRITE_COMMAND,
];
