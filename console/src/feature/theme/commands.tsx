// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

import { useModal } from "@/feature/theme/Modal";
import { Command } from "@/platform/command";

export const SelectCommand = Command.create({
  key: "select_color_theme",
  name: "Change color theme",
  useOnSelect: () => useModal(),
  icon: <Icon.DarkMode />,
  sortOrder: 0,
});
