// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback } from "react";
import { MdDarkMode } from "react-icons/md";

import { setActiveTheme } from "@/layout/slice";
import { Palette } from "@/palette";

export const ToggleCommand: Palette.Command = ({ store, ...listProps }) => {
  const handleSelect = useCallback(() => store.dispatch(setActiveTheme()), [store]);
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Toggle Color Theme"
      icon={<MdDarkMode />}
      onSelect={handleSelect}
    />
  );
};
ToggleCommand.key = "toggle-theme";
ToggleCommand.commandName = "Toggle Color Theme";

export const COMMANDS = [ToggleCommand];
