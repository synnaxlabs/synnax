// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { user } from "@synnaxlabs/client";
import { Access, User as PUser } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Palette } from "@/palette";
import { useRegisterModal } from "@/user/Register";

const COMMAND_NAME = "Register a user";

const RegisterCommand: Palette.Command = (listProps) => {
  const open = useRegisterModal();
  const handleSelect = useCallback(() => open(), [open]);
  return (
    <Palette.CommandListItem
      {...listProps}
      name={COMMAND_NAME}
      icon={<PUser.CreateIcon />}
      onSelect={handleSelect}
    />
  );
};
RegisterCommand.key = "register_user";
RegisterCommand.commandName = COMMAND_NAME;
RegisterCommand.useVisible = () => Access.useCreateGranted(user.TYPE_ONTOLOGY_ID);

export const COMMANDS = [RegisterCommand];
