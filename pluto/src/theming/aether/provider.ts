// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { aether } from "@/aether/aether";
import { status } from "@/status/aether";
import { type Theme, themeZ } from "@/theming/core/theme";

const CONTEXT_KEY = "pluto-theming-context";

export const fontSpecZ = z.object({
  name: z.string(),
  url: z.string(),
});

const providerStateZ = z.object({
  theme: themeZ,
  fontURLs: z.array(fontSpecZ),
});

export class Provider extends aether.Composite<typeof providerStateZ> {
  static readonly TYPE: string = "theming.Provider";
  static readonly z = providerStateZ;
  schema = Provider.z;

  afterUpdate(ctx: aether.Context): void {
    const v = ctx.getOptional<Theme>(CONTEXT_KEY);
    if (v != null && this.state.theme.key === this.prevState.theme.key) return;
    ctx.set(CONTEXT_KEY, this.state.theme);
    const runAsync = status.useErrorHandler(ctx);
    runAsync(async () => {
      await this.loadFonts();
    }, "failed to load theme fonts");
  }

  private async loadFonts(): Promise<void> {
    await Promise.all(
      this.state.fontURLs.map(async ({ name, url }) => {
        const face = new FontFace(name, `url(${url})`);
        try {
          await face.load();
          // @ts-expect-error - font loading
          self.fonts.add(face);
        } catch (e) {
          console.error(e);
        }
      }),
    );
  }
}

export const use = (ctx: aether.Context): Theme => ctx.get<Theme>(CONTEXT_KEY);

export const REGISTRY: aether.ComponentRegistry = {
  [Provider.TYPE]: Provider,
};
