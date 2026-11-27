// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { renderThemed, treatment } from "@synnaxlabs/lyra/styletest";
import { describe, expect, it } from "vitest";

import { StringDisplay } from "@/schematic/node/general/stringDisplay";

describe("StringDisplay preview", () => {
  it("should draw the sample text with tabular digits", async () => {
    const config = schematic.stringDisplayNodeConfigZ.parse({
      variant: "string_display",
    });
    const { getByText } = await renderThemed(
      <StringDisplay.spec.Preview {...config} />,
    );
    expect(treatment(getByText("Hello world!").element())).toBe("numeric");
  });
});
