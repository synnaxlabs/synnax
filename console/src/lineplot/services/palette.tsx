// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot } from "@synnaxlabs/client";
import { Access } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { LinePlot } from "@/lineplot";
import { CreateIcon } from "@/lineplot/services/Icon";
import { Palette } from "@/palette";

const useVisible = () => Access.useUpdateGranted(lineplot.TYPE_ONTOLOGY_ID);

export const CreateCommand: Palette.Command = ({ placeLayout, ...listProps }) => {
  const handleSelect = useCallback(() => placeLayout(LinePlot.create()), [placeLayout]);
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Create a Line Plot"
      icon={<CreateIcon />}
      onSelect={handleSelect}
    />
  );
};
CreateCommand.key = "create-line-plot";
CreateCommand.commandName = "Create a Line Plot";
CreateCommand.useVisible = useVisible;

export const COMMANDS = [CreateCommand];
