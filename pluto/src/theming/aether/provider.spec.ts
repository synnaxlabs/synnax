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
import { afterEach, describe, expect, it, vi } from "vitest";

import { aetherTest } from "@/aether/testutil";
import { renderAether } from "@/testutil/renderAether";

const { Leaf } = aetherTest;

const FONT_URLS = [{ name: "Inter Four", url: "inter-400.woff2" }];

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

  it("loads each font with common ligatures off", async () => {
    const FontFace = vi.fn(
      class {
        load = async () => {};
      },
    );
    vi.stubGlobal("FontFace", FontFace);
    vi.stubGlobal("fonts", { add: vi.fn() });
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
});
