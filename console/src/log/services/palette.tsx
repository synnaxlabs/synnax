// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log } from "@synnaxlabs/client";
import { Access } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Log } from "@/log";
import { CreateIcon } from "@/log/services/Icon";
import { Palette } from "@/palette";

const useVisible = () => Access.useUpdateGranted(log.TYPE_ONTOLOGY_ID);

export const CreateCommand: Palette.Command = ({ placeLayout, ...listProps }) => {
  const handleSelect = useCallback(() => placeLayout(Log.create()), [placeLayout]);
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Create a log"
      icon={<CreateIcon />}
      onSelect={handleSelect}
    />
  );
};
CreateCommand.key = "create-log";
CreateCommand.commandName = "Create a log";
CreateCommand.useVisible = useVisible;

export const COMMANDS = [CreateCommand];
