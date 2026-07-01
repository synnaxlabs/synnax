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

import { EXPLORER_LAYOUT } from "@/feature/arc/Explorer";
import { useCreate } from "@/platform/arc/useCreate";
import { Palette } from "@/platform/palette";

const useCreateVisible = () => Access.useCreateGranted(arc.TYPE_ONTOLOGY_ID);
const useViewVisible = () => Access.useRetrieveGranted(arc.TYPE_ONTOLOGY_ID);

const COMMAND_NAME = "Create an Arc automation";

export const CreateCommand: Palette.Command = ({ rename, ...listProps }) => {
  const create = useCreate();
  return (
    <Palette.CommandListItem
      {...listProps}
      name={COMMAND_NAME}
      icon={<Icon.Arc />}
      onSelect={create}
    />
  );
};
CreateCommand.key = "create_arc";
CreateCommand.commandName = COMMAND_NAME;
CreateCommand.useVisible = useCreateVisible;

export const OpenExplorerCommand = Palette.createSimpleCommand({
  key: "open_arc_explorer",
  name: "Open the Arc Explorer",
  icon: <Icon.Explore />,
  layout: EXPLORER_LAYOUT,
  useVisible: useViewVisible,
});

export const COMMANDS = [CreateCommand, OpenExplorerCommand];
