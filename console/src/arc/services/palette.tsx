// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { Access, Icon } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Arc } from "@/arc";
import { Palette } from "@/palette";

const useUpdateVisible = () => Access.useUpdateGranted(arc.TYPE_ONTOLOGY_ID);
const useViewVisible = () => Access.useRetrieveGranted(arc.TYPE_ONTOLOGY_ID);

export const CreateCommand: Palette.Command = ({
  placeLayout,
  rename,
  handleError,
  ...listProps
}) => {
  const createArcModal = Arc.Editor.useCreate();
  const handleSelect = useCallback(
    () =>
      handleError(async () => {
        const result = await createArcModal({});
        if (result != null)
          placeLayout(Arc.Editor.create({ name: result.name, mode: result.mode }));
      }, "Failed to create Arc program"),
    [placeLayout, handleError, createArcModal],
  );
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Create an Arc Automation"
      icon={<Icon.Arc />}
      onSelect={handleSelect}
    />
  );
};
CreateCommand.key = "create_arc";
CreateCommand.commandName = "Create an Arc Automation";
CreateCommand.useVisible = useUpdateVisible;

export const OpenExplorerCommand = Palette.createSimpleCommand({
  key: "open_arc_explorer",
  name: "Open Arc Explorer",
  icon: <Icon.Explore />,
  layout: Arc.EXPLORER_LAYOUT,
  useVisible: useViewVisible,
});

export const COMMANDS = [CreateCommand, OpenExplorerCommand];
