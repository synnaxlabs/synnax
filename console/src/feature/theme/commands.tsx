// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, Theming } from "@synnaxlabs/pluto";

import { Command } from "@/platform/command";

export const ToggleCommand = Command.create({
  key: "toggle_theme",
  name: "Toggle color theme",
  useOnSelect: () => {
    const { toggleTheme } = Theming.useContext();
    return toggleTheme;
  },
  icon: <Icon.DarkMode />,
});
