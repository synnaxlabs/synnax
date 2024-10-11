// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { aether } from "@/aether/aether";
import { type Theme, themeZ } from "@/theming/core/theme";

const CONTEXT_KEY = "pluto-theming-context";

export const fontSpecZ = z.object({
  name: z.string(),
  url: z.string(),
});

const providerStateZ = z.object({
  theme: themeZ,
  fontURLs: z.array(fontSpecZ),
  // For some reason the generate type is too deep, so we need to cast it to ZodTypeAny
});

export class Provider extends aether.Composite<typeof providerStateZ> {
  static readonly TYPE: string = "theming.Provider";
  static readonly z = providerStateZ;
  schema = Provider.z;

  async afterUpdate(): Promise<void> {
    this.ctx.set(CONTEXT_KEY, this.state.theme);
    await this.loadFonts();
  }

  private async loadFonts(): Promise<void> {
    await Promise.all(
      this.state.fontURLs.map(async ({ name, url }) => {
        const face = new FontFace(name, `url(${url})`);
        await face.load();
        self.fonts.add(face);
      }),
    );
  }
}

export const use = (ctx: aether.Context): Theme => ctx.get<Theme>(CONTEXT_KEY);

export const REGISTRY: aether.ComponentRegistry = {
  [Provider.TYPE]: Provider,
};
