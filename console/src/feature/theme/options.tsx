// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import { type Theme } from "@/session/theme";

export interface Option {
  key: Theme.Mode;
  name: string;
  icon: Icon.ReactElement;
}

export const OPTIONS: Option[] = [
  { key: "light", name: "Light", icon: <Icon.LightMode /> },
  { key: "dark", name: "Dark", icon: <Icon.DarkMode /> },
  { key: "system", name: "System", icon: <Icon.Sync /> },
];
