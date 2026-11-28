// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderThemed, treatment } from "@synnaxlabs/lyra/styletest";
import { describe, expect, it } from "vitest";

import type { Options } from "@/schematic/node/common/options";
import { Primitive } from "@/schematic/node/general/stateIndicator/primitive";

const OPTIONS: Options.State[] = [{ key: "1", name: "Stage 2", value: 1 }];

describe("StateIndicator", () => {
  it("should draw the state with tabular digits", async () => {
    const { getByText } = await renderThemed(
      <Primitive.StateIndicator matchedOptionKey="1" options={OPTIONS} />,
    );
    expect(treatment(getByText("Stage 2").element())).toBe("numeric");
  });
});
