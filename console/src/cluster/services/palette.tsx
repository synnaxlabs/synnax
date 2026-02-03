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
import { AiFillApi } from "react-icons/ai";

import { Cluster } from "@/cluster";
import { logout } from "@/cluster/services/logout";
import { Palette } from "@/palette";
import { Runtime } from "@/runtime";

const useConnectVisible = () => Runtime.ENGINE === "tauri";

export const ConnectCommand = Palette.createSimpleCommand({
  key: "connect-cluster",
  name: "Add a Core",
  icon: <AiFillApi />,
  layout: Cluster.CONNECT_LAYOUT,
  useVisible: useConnectVisible,
});

export const LogoutCommand: Palette.Command = ({ store, ...listProps }) => {
  const handleSelect = useCallback(() => logout(store.dispatch), [store]);
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Log Out"
      icon={<Icon.Logout />}
      onSelect={handleSelect}
    />
  );
};
LogoutCommand.key = "logout";
LogoutCommand.commandName = "Log Out";

export const COMMANDS = [ConnectCommand, LogoutCommand];
