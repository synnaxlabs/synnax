// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot } from "@synnaxlabs/client";
import { Access, LinePlot } from "@synnaxlabs/pluto";

import { useCreate } from "@/service/lineplot/useCreate";
import { Palette } from "@/palette";

const COMMAND_NAME = "Create a line plot";

const CreateCommand: Palette.Command = ({ placeLayout, ...listProps }) => {
  const create = useCreate({});
  return (
    <Palette.CommandListItem
      {...listProps}
      name={COMMAND_NAME}
      icon={<LinePlot.CreateIcon />}
      onSelect={create}
    />
  );
};
CreateCommand.key = "create_line_plot";
CreateCommand.commandName = COMMAND_NAME;
CreateCommand.useVisible = () => Access.useCreateGranted(lineplot.TYPE_ONTOLOGY_ID);

export const COMMANDS = [CreateCommand];
