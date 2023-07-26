// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { AetherContext, AetherComposite } from "../aether/worker";
import { CSS } from "../css";

import { Theme, themeZ } from "./theme";

const aetherThemeProviderState = z.object({
  theme: themeZ,
});

// eslint-disable-next-line @typescript-eslint/no-extraneous-class
export class ThemeContext {
  static readonly CONTEXT_KEY = CSS.B("theme");

  static create(ctx: AetherContext, theme: Theme): void {
    ctx.set(ThemeContext.CONTEXT_KEY, theme);
  }

  static use(ctx: AetherContext): Theme {
    return ctx.get<Theme>(ThemeContext.CONTEXT_KEY);
  }
}

export class AetherThemeProvider extends AetherComposite<
  typeof aetherThemeProviderState
> {
  static readonly TYPE: string = CSS.B("ThemeProvider");
  static readonly z = aetherThemeProviderState;
  schema = AetherThemeProvider.z;

  afterUpdate(): void {
    ThemeContext.create(this.ctx, this.state.theme);
  }
}
