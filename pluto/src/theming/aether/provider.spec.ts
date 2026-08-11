// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sleep, TimeSpan } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { aetherTest } from "@/aether/test";
import { renderAether } from "@/testutil/renderAether";
import { SYNNAX_LIGHT } from "@/theming/base/theme";

const { Leaf } = aetherTest;

const FONT_URLS = [{ name: "Inter Four", url: "inter-400.woff2" }];

describe("theming.Provider", () => {
  it("reports no error when the host has no font registry", async () => {
    vi.stubGlobal("FontFace", undefined);
    const h = renderAether(Leaf, {
      state: {},
      theming: { theme: SYNNAX_LIGHT, fontURLs: FONT_URLS },
    });
    await sleep.sleep(TimeSpan.milliseconds(20));
    expect(h.providers.status?.state.statuses).toHaveLength(0);
  });
});
