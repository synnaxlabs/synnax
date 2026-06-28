// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger } from "@synnaxlabs/client";
import { Access, Icon, Ranger } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Palette } from "@/palette";
import { Range } from "@/range";
import { useCreateModal } from "@/range/Create";

export const CreateCommand: Palette.Command = (listProps) => {
  const open = useCreateModal();
  const handleSelect = useCallback(() => open({}), [open]);
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Create a range"
      icon={<Ranger.CreateIcon />}
      onSelect={handleSelect}
    />
  );
};
CreateCommand.key = "define-range";
CreateCommand.commandName = "Create a range";
CreateCommand.useVisible = () => Access.useCreateGranted(ranger.TYPE_ONTOLOGY_ID);

export const OpenExplorerCommand = Palette.createSimpleCommand({
  key: "open-explorer",
  name: "Open the Range Explorer",
  icon: <Icon.Explore />,
  layout: Range.EXPLORER_LAYOUT,
  useVisible: () => Access.useRetrieveGranted(ranger.TYPE_ONTOLOGY_ID),
});

export const COMMANDS = [CreateCommand, OpenExplorerCommand];
