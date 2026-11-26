// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import inter from "@fontsource-variable/inter/files/inter-latin-standard-normal.woff2";
import { theme } from "@synnaxlabs/lyra/theme";
import { describe, expect, it } from "vitest";

import { aetherTest } from "@/aether/testutil";
import { renderAether } from "@/testutil/internal";

describe("theming.Provider", () => {
  it("adds each font to the document when it runs on the main thread", async () => {
    const family = "Main Thread Inter";
    renderAether(aetherTest.Leaf, {
      state: {},
      theming: {
        theme: theme.SYNNAX_LIGHT,
        fontURLs: [{ name: family, url: inter, tabular: true }],
      },
    });
    await expect
      .poll(() =>
        Array.from(document.fonts).some(
          (f) => f.family.includes(family) && f.status === "loaded",
        ),
      )
      .toBe(true);
  });
});
