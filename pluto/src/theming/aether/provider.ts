// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { theme } from "@synnaxlabs/lyra/theme";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { status } from "@/status/aether";

const CONTEXT_KEY = "pluto-theming-context";

// A ligature in Geist Mono is one cell wide, so it breaks a fixed character grid.
const DESCRIPTORS: FontFaceDescriptors = { featureSettings: '"liga" 0' };
// The face loads the variable font, so it spans every weight.
const TABULAR_DESCRIPTORS: FontFaceDescriptors = {
  featureSettings: '"liga" 0, "tnum" 1',
  weight: "100 900",
};

export const fontSpecZ = z.object({
  name: z.string(),
  url: z.string(),
  /** If true, the face draws digits at one width. */
  tabular: z.boolean().default(false),
});
export interface FontSpec extends z.input<typeof fontSpecZ> {}

const providerStateZ = z.object({
  theme: theme.themeZ,
  fontURLs: z.array(fontSpecZ),
});

export class Provider extends aether.Composite<typeof providerStateZ> {
  static readonly TYPE: string = "theming.Provider";
  static readonly z = providerStateZ;
  override schema = Provider.z;

  override afterUpdate(ctx: aether.Context): void {
    const v = ctx.getOptional<theme.Theme>(CONTEXT_KEY);
    if (v != null && this.state.theme.key === this.prevState.theme.key) return;
    ctx.set(CONTEXT_KEY, this.state.theme);
    const runAsync = status.useErrorHandler(ctx);
    runAsync(async () => {
      await this.loadFonts(ctx);
    }, "Failed to load theme fonts");
  }

  private async loadFonts(ctx: aether.Context): Promise<void> {
    // Hosts without a font registry (jsdom, some worker runtimes) have no FontFace.
    // A throw reports an error that re-renders consumers, looping any that suspend.
    if (typeof FontFace === "undefined" || this.state.fontURLs.length === 0) return;
    const fonts = fontSet();
    await Promise.all(
      this.state.fontURLs.map(async ({ name, url, tabular }) => {
        const descriptors = tabular ? TABULAR_DESCRIPTORS : DESCRIPTORS;
        const face = new FontFace(name, `url(${url})`, descriptors);
        try {
          await face.load();
        } catch (e) {
          console.error(e);
          return;
        }
        fonts.add(face);
      }),
    );
    // Text drawn before the fonts loaded has fallback glyphs. A new theme value makes
    // its consumers draw again.
    ctx.set(CONTEXT_KEY, { ...this.state.theme });
    this.propagate();
  }
}

// The provider runs in a worker, or on the main thread with in-process comms. A canvas
// draws only with the faces in the font set of its own scope. Each supported browser
// has a font set in a worker, so a host without one is an error.
const fontSet = (): FontFaceSet => {
  if (typeof document !== "undefined") return document.fonts;
  if (
    typeof FontFaceSet !== "undefined" &&
    "fonts" in self &&
    self.fonts instanceof FontFaceSet
  )
    return self.fonts;
  throw new Error("This worker has no font set. Canvas text uses a fallback font.");
};

export const use = (ctx: aether.Context): theme.Theme =>
  ctx.get<theme.Theme>(CONTEXT_KEY);

export const REGISTRY: aether.ComponentRegistry = {
  [Provider.TYPE]: Provider,
};
