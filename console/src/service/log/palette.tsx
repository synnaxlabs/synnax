// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log } from "@synnaxlabs/client";
import { Access, Log } from "@synnaxlabs/pluto";

import { useCreate } from "@/component/log/useCreate";
import { Palette } from "@/component/palette";

const COMMAND_NAME = "Create a log";

const CreateCommand: Palette.Command = (listProps) => {
  const create = useCreate({});
  return (
    <Palette.CommandListItem
      {...listProps}
      name={COMMAND_NAME}
      icon={<Log.CreateIcon />}
      onSelect={create}
    />
  );
};
CreateCommand.key = "create_log";
CreateCommand.commandName = COMMAND_NAME;
CreateCommand.useVisible = () => Access.useCreateGranted(log.TYPE_ONTOLOGY_ID);

export const COMMANDS = [CreateCommand];
