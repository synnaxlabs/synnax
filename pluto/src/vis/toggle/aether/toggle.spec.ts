// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, it } from "vitest";

import { aetherTest } from "@/aether/test";
import { telem } from "@/telem/aether";
import { testTelem } from "@/telem/test";
import { toggle } from "@/vis/toggle/aether";

describe("toggle", () => {
  it("should be a component", () => {
    const initialState: toggle.ToggleState = {
      enabled: false,
      triggered: false,
      source: testTelem.booleanSourceSpec,
      sink: testTelem.booleanSinkSpec,
    };

    const prov = new testTelem.Provider();
    const t = aetherTest.render(toggle.Toggle, initialState, (ctx) =>
      telem.setProvider(ctx, prov),
    );
  });
});
