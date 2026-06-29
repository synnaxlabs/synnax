// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import { Access, Icon, Status } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Palette } from "@/palette";
import { useCreateModal } from "@/status/Create";
import { EXPLORER_LAYOUT } from "@/status/Explorer";

const CreateCommand: Palette.Command = (listProps) => {
  const open = useCreateModal();
  const handleSelect = useCallback(() => open(), [open]);
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Create a status"
      icon={<Status.CreateIcon />}
      onSelect={handleSelect}
    />
  );
};
CreateCommand.key = "create_status";
CreateCommand.commandName = "Create a status";
CreateCommand.useVisible = () => Access.useCreateGranted(status.TYPE_ONTOLOGY_ID);

const OpenExplorerCommand = Palette.createSimpleCommand({
  key: "open_status_explorer",
  name: "Open the Status Explorer",
  icon: <Icon.Explore />,
  layout: EXPLORER_LAYOUT,
  useVisible: () => Access.useRetrieveGranted(status.TYPE_ONTOLOGY_ID),
});

export const COMMANDS = [CreateCommand, OpenExplorerCommand];
