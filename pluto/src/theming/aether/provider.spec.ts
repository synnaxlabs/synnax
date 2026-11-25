// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { theme } from "@synnaxlabs/lyra/theme";
import { sleep, TimeSpan } from "@synnaxlabs/x";
import { afterEach, describe, expect, it, onTestFinished, vi } from "vitest";

import { aetherTest } from "@/aether/testutil";
import { renderAether } from "@/testutil/internal";

const { Leaf } = aetherTest;

const FONT_URLS = [{ name: "Inter Four", url: "inter-400.woff2" }];

// jsdom has no document.fonts.
const stubFontSet = (): void => {
  Object.defineProperty(document, "fonts", {
    configurable: true,
    value: { add: vi.fn() },
  });
  onTestFinished(() => {
    Reflect.deleteProperty(document, "fonts");
  });
};

describe("theming.Provider", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("reports no error when the host has no font registry", async () => {
    vi.stubGlobal("FontFace", undefined);
    const h = renderAether(Leaf, {
      state: {},
      theming: { theme: theme.SYNNAX_LIGHT, fontURLs: FONT_URLS },
    });
    await sleep.sleep(TimeSpan.milliseconds(20));
    expect(h.providers.status?.state.statuses).toHaveLength(0);
  });

  describe("in a worker with no font set", () => {
    const stubWorker = (): void => {
      vi.stubGlobal("FontFace", class {});
      vi.stubGlobal("FontFaceSet", undefined);
      vi.stubGlobal("document", undefined);
    };

    it("reports one error when it has fonts to add", async () => {
      stubWorker();
      const h = renderAether(Leaf, {
        state: {},
        theming: { theme: theme.SYNNAX_LIGHT, fontURLs: FONT_URLS },
      });
      await expect
        .poll(() => h.providers.status?.state.statuses)
        .toEqual([
          expect.objectContaining({
            variant: "error",
            message: "Failed to load theme fonts",
          }),
        ]);
    });

    it("reports no error when it has no fonts to add", async () => {
      stubWorker();
      const h = renderAether(Leaf, {
        state: {},
        theming: { theme: theme.SYNNAX_LIGHT, fontURLs: [] },
      });
      await sleep.sleep(TimeSpan.milliseconds(20));
      expect(h.providers.status?.state.statuses).toHaveLength(0);
    });
  });

  it("loads each font with common ligatures off", async () => {
    const FontFace = vi.fn(
      class {
        load = async () => {};
      },
    );
    vi.stubGlobal("FontFace", FontFace);
    stubFontSet();
    renderAether(Leaf, {
      state: {},
      theming: { theme: theme.SYNNAX_LIGHT, fontURLs: FONT_URLS },
    });
    await expect
      .poll(() => FontFace)
      .toHaveBeenCalledWith("Inter Four", "url(inter-400.woff2)", {
        featureSettings: '"liga" 0',
      });
  });

  it("loads a tabular font with tabular digits at every weight", async () => {
    const FontFace = vi.fn(
      class {
        load = async () => {};
      },
    );
    vi.stubGlobal("FontFace", FontFace);
    stubFontSet();
    renderAether(Leaf, {
      state: {},
      theming: {
        theme: theme.SYNNAX_LIGHT,
        fontURLs: [{ name: "Inter Tabular", url: "inter.woff2", tabular: true }],
      },
    });
    await expect
      .poll(() => FontFace)
      .toHaveBeenCalledWith("Inter Tabular", "url(inter.woff2)", {
        featureSettings: '"liga" 0, "tnum" 1',
        weight: "100 900",
      });
  });
});
