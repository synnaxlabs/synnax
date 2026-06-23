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

import { useCreate } from "@/layered/service/schematic/useCreate";
import { Palette } from "@/palette";

const CreateCommand: Palette.Command = ({ placeLayout, ...listProps }) => {
  const create = useCreate({});
  const handleSelect = useCallback(() => create(), [create]);
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Create a schematic"
      icon={<PSchematic.CreateIcon />}
      onSelect={handleSelect}
    />
  );
};
CreateCommand.key = "create-schematic";
CreateCommand.commandName = "Create a schematic";
CreateCommand.useVisible = () => Access.useCreateGranted(schematic.TYPE_ONTOLOGY_ID);

export const COMMANDS = [CreateCommand];
