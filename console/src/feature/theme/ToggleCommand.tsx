// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, Theming } from "@synnaxlabs/pluto";

import { Palette } from "@/platform/palette";

const COMMAND_NAME = "Toggle color theme";

export interface ToggleCommandProps extends Omit<
  Palette.ListItemProps,
  "name" | "onSelect" | "icon"
> {}

export const ToggleCommand = (props: ToggleCommandProps) => {
  const { toggleTheme } = Theming.useContext();
  return (
    <Palette.CommandListItem
      {...props}
      name={COMMAND_NAME}
      icon={<Icon.DarkMode />}
      onSelect={toggleTheme}
    />
  );
};
ToggleCommand.key = "toggle_theme";
ToggleCommand.commandName = COMMAND_NAME;
