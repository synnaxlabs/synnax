// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Theming } from "@synnaxlabs/pluto";
import { MdDarkMode } from "react-icons/md";

import { Palette } from "@/palette";

export const ToggleCommand: Palette.Command = (listProps) => {
  const { toggleTheme } = Theming.useContext();
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Toggle color theme"
      icon={<MdDarkMode />}
      onSelect={toggleTheme}
    />
  );
};
ToggleCommand.key = "toggle-theme";
ToggleCommand.commandName = "Toggle color theme";

export const COMMANDS = [ToggleCommand];
