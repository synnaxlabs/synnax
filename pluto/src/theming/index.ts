// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { synnaxDark, synnaxLight } from "./theme";
import {
  ThemeProvider,
  ThemeSwitch,
  useThemeContext,
  useThemeProvider,
} from "./ThemeContext";
export * from "./hooks";
export type { Theme } from "./theme";
export type { ThemeProviderProps } from "./ThemeContext";

export const Theming = {
  Provider: ThemeProvider,
  Switch: ThemeSwitch,
  useContext: useThemeContext,
  useProvider: useThemeProvider,
  themes: { synnaxDark, synnaxLight },
};
