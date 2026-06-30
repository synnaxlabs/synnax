// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label } from "@synnaxlabs/client";
import { Access, Icon } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { useEditModal } from "@/label/Edit";
import { Palette } from "@/palette";

const useVisible = () => Access.useUpdateGranted(label.TYPE_ONTOLOGY_ID);

const COMMAND_NAME = "Edit labels";

export const EditCommand: Palette.Command = (listProps) => {
  const open = useEditModal();
  const handleSelect = useCallback(() => open(), [open]);
  return (
    <Palette.CommandListItem
      {...listProps}
      name={COMMAND_NAME}
      icon={<Icon.Label />}
      onSelect={handleSelect}
    />
  );
};
EditCommand.key = "edit_labels";
EditCommand.commandName = COMMAND_NAME;
EditCommand.useVisible = useVisible;

export const COMMANDS = [EditCommand];
