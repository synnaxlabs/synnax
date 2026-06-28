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
import { useOpenRegister } from "@/user/Register";

const RegisterCommand: Palette.Command = (listProps) => {
  const open = useOpenRegister();
  const handleSelect = useCallback(() => open(), [open]);
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Register a user"
      icon={<PUser.CreateIcon />}
      onSelect={handleSelect}
    />
  );
};
RegisterCommand.key = "register-user";
RegisterCommand.commandName = "Register a user";
RegisterCommand.useVisible = () => Access.useCreateGranted(user.TYPE_ONTOLOGY_ID);

export const COMMANDS = [RegisterCommand];
