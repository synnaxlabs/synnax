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
import { useCallback } from "react";

import { importRead, importWrite } from "@/hardware/labjack/task/import";
import { READ_LAYOUT } from "@/hardware/labjack/task/Read";
import { WRITE_LAYOUT } from "@/hardware/labjack/task/Write";
import { Palette } from "@/palette";

const useVisible = () => Access.useUpdateGranted(task.TYPE_ONTOLOGY_ID);

export const CreateReadCommand = Palette.createSimpleCommand({
  key: "labjack-create-read-task",
  name: "Create a LabJack Read Task",
  icon: <Icon.Logo.LabJack />,
  layout: READ_LAYOUT,
  useVisible,
});

export const CreateWriteCommand = Palette.createSimpleCommand({
  key: "labjack-create-write-task",
  name: "Create a LabJack Write Task",
  icon: <Icon.Logo.LabJack />,
  layout: WRITE_LAYOUT,
  useVisible,
});

export const ImportReadCommand: Palette.Command = ({
  placeLayout,
  handleError,
  store,
  client,
  fluxStore,
  ...listProps
}) => {
  const handleSelect = useCallback(
    () => importRead({ placeLayout, handleError, store, client, fluxStore }),
    [placeLayout, handleError, store, client, fluxStore],
  );
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Import LabJack Read Task(s)"
      icon={<Icon.Logo.LabJack />}
      onSelect={handleSelect}
    />
  );
};
ImportReadCommand.key = "labjack-import-read-task";
ImportReadCommand.commandName = "Import LabJack Read Task(s)";
ImportReadCommand.sortOrder = -1;
ImportReadCommand.useVisible = useVisible;

export const ImportWriteCommand: Palette.Command = ({
  placeLayout,
  handleError,
  store,
  client,
  fluxStore,
  ...listProps
}) => {
  const handleSelect = useCallback(
    () => importWrite({ placeLayout, handleError, store, client, fluxStore }),
    [placeLayout, handleError, store, client, fluxStore],
  );
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Import LabJack Write Task(s)"
      icon={<Icon.Logo.LabJack />}
      onSelect={handleSelect}
    />
  );
};
ImportWriteCommand.key = "labjack-import-write-task";
ImportWriteCommand.commandName = "Import LabJack Write Task(s)";
ImportWriteCommand.sortOrder = -1;
ImportWriteCommand.useVisible = useVisible;

export const COMMANDS = [
  CreateReadCommand,
  CreateWriteCommand,
  ImportReadCommand,
  ImportWriteCommand,
];
