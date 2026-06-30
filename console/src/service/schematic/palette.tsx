// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { Access, Schematic as PSchematic } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { useCreate } from "@/service/schematic/useCreate";
import { Palette } from "@/palette";

const COMMAND_NAME = "Create a schematic";

const CreateCommand: Palette.Command = ({ placeLayout, ...listProps }) => {
  const create = useCreate({});
  const handleSelect = useCallback(() => create(), [create]);
  return (
    <Palette.CommandListItem
      {...listProps}
      name={COMMAND_NAME}
      icon={<PSchematic.CreateIcon />}
      onSelect={handleSelect}
    />
  );
};
CreateCommand.key = "create_schematic";
CreateCommand.commandName = COMMAND_NAME;
CreateCommand.useVisible = () => Access.useCreateGranted(schematic.TYPE_ONTOLOGY_ID);

export const COMMANDS = [CreateCommand];
