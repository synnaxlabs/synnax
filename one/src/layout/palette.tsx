// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { MdDarkMode } from "react-icons/md";

import { setActiveTheme } from "@/layout/slice";
import { type Command, type CommandSelectionContext } from "@/palette/Palette";

export const toggleThemeCommand: Command = {
  icon: <MdDarkMode />,
  name: "Toggle Color Theme",
  key: "toggle-theme",
  onSelect: (ctx: CommandSelectionContext) => {
    ctx.store.dispatch(setActiveTheme());
  },
};

export const COMMANDS = [toggleThemeCommand];
