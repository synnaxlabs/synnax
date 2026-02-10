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

import { createLayout } from "@/hardware/task/sequence/Sequence";
import { Palette } from "@/palette";

const useVisible = () => Access.useUpdateGranted(task.TYPE_ONTOLOGY_ID);

export const CreateCommand: Palette.Command = ({
  placeLayout,
  rename,
  handleError,
  ...listProps
}) => {
  const handleSelect = useCallback(
    () =>
      handleError(async () => {
        const layout = await createLayout({ rename });
        if (layout != null) placeLayout(layout);
      }, "Failed to create a control sequence"),
    [placeLayout, rename, handleError],
  );
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Create a control sequence"
      icon={<Icon.Control />}
      onSelect={handleSelect}
    />
  );
};
CreateCommand.key = "create-control-sequence";
CreateCommand.commandName = "Create a control sequence";
CreateCommand.useVisible = useVisible;

export const COMMANDS = [CreateCommand];
