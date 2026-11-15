// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { StateIndicator } from "@/schematic/node/general/stateIndicator/Primitive";

const OPTIONS: schematic.StateMapping[] = [{ key: "1", name: "Active", value: 1 }];

describe("StateIndicator", () => {
  it("should size the text from the level, not the size", () => {
    const { getByText } = render(
      <StateIndicator matchedOptionKey="1" options={OPTIONS} size="tiny" level="h2" />,
    );
    expect(getByText("Active").tagName).toBe("H2");
  });
});
