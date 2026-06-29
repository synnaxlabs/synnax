// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { logout } from "@/cluster/services/logout";
import { Palette } from "@/palette";

const COMMAND_NAME = "Log out";

export const LogoutCommand: Palette.Command = ({ store, ...listProps }) => {
  const handleSelect = useCallback(() => logout(store.dispatch), [store]);
  return (
    <Palette.CommandListItem
      {...listProps}
      name={COMMAND_NAME}
      icon={<Icon.Logout />}
      onSelect={handleSelect}
    />
  );
};
LogoutCommand.key = "logout";
LogoutCommand.commandName = COMMAND_NAME;

export const COMMANDS = [LogoutCommand];
