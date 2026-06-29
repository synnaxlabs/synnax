// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel } from "@synnaxlabs/client";
import { Access, Channel as PChannel } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { useCalculatedModal } from "@/channel/Calculated";
import { useCreateModal } from "@/channel/Create";
import { Palette } from "@/palette";

const useVisible = () => Access.useCreateGranted(channel.TYPE_ONTOLOGY_ID);

const CREATE_COMMAND_NAME = "Create a channel";

const CreateCommand: Palette.Command = (listProps) => {
  const open = useCreateModal();
  const handleSelect = useCallback(() => open(), [open]);
  return (
    <Palette.CommandListItem
      {...listProps}
      name={CREATE_COMMAND_NAME}
      icon={<PChannel.CreateIcon />}
      onSelect={handleSelect}
    />
  );
};
CreateCommand.key = "create_channel";
CreateCommand.commandName = CREATE_COMMAND_NAME;
CreateCommand.useVisible = useVisible;

const CREATE_CALCULATED_COMMAND_NAME = "Create a calculated channel";

const CreateCalculatedCommand: Palette.Command = (listProps) => {
  const open = useCalculatedModal();
  const handleSelect = useCallback(() => open(), [open]);
  return (
    <Palette.CommandListItem
      {...listProps}
      name={CREATE_CALCULATED_COMMAND_NAME}
      icon={<PChannel.CreateCalculatedIcon />}
      onSelect={handleSelect}
    />
  );
};
CreateCalculatedCommand.key = "create_calculated_channel";
CreateCalculatedCommand.commandName = CREATE_CALCULATED_COMMAND_NAME;
CreateCalculatedCommand.useVisible = useVisible;

export const COMMANDS = [CreateCommand, CreateCalculatedCommand];
