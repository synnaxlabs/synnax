// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, Status, Theming } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Command } from "@/platform/command";
import { Session } from "@/session";

export const ToggleCommand = Command.create({
  key: "toggle_theme",
  name: "Toggle color theme",
  useOnSelect: () => {
    const { toggleTheme } = Theming.useContext();
    return toggleTheme;
  },
  icon: <Icon.DarkMode />,
  sortOrder: 0,
});

const base = " theme sync with system";

export const ToggleSyncWithSystemCommand: Command.Command = (listProps) => {
  const enabled = Session.Theme.useSelectSyncWithSystem();
  const toggle = Session.Theme.useToggleSyncWithSystem();
  const addStatus = Status.useAdder();
  const onSelect = useCallback(() => {
    toggle();
    addStatus({
      variant: "success",
      message: enabled ? `Disabled${base}` : `Enabled${base}`,
    });
  }, [toggle, addStatus, enabled]);
  return (
    <Command.ListItem
      {...listProps}
      name={enabled ? `Disable${base}` : `Enable${base}`}
      icon={<Icon.Sync />}
      onSelect={onSelect}
    />
  );
};
ToggleSyncWithSystemCommand.key = "enable_disable_theme_sync_with_system";
ToggleSyncWithSystemCommand.commandName = `Enable Disable${base}`;
ToggleSyncWithSystemCommand.sortOrder = 1;
