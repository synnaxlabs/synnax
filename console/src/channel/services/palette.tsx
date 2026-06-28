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

import { useOpenCalculated } from "@/channel/Calculated";
import { useOpenCreate } from "@/channel/Create";
import { Palette } from "@/palette";

const useVisible = () => Access.useCreateGranted(channel.TYPE_ONTOLOGY_ID);

const CreateCommand: Palette.Command = ({ placeModal, ...listProps }) => {
  const handleSelect = useCallback(() => placeModal.open(useOpenCreate), [placeModal]);
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Create a channel"
      icon={<PChannel.CreateIcon />}
      onSelect={handleSelect}
    />
  );
};
CreateCommand.key = "create-channel";
CreateCommand.commandName = "Create a channel";
CreateCommand.useVisible = useVisible;

const CreateCalculatedCommand: Palette.Command = ({ placeModal, ...listProps }) => {
  const handleSelect = useCallback(
    () => placeModal.open(useOpenCalculated),
    [placeModal],
  );
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Create a calculated channel"
      icon={<PChannel.CreateCalculatedIcon />}
      onSelect={handleSelect}
    />
  );
};
CreateCalculatedCommand.key = "create-calculated-channel";
CreateCalculatedCommand.commandName = "Create a calculated channel";
CreateCalculatedCommand.useVisible = useVisible;

export const COMMANDS = [CreateCommand, CreateCalculatedCommand];
