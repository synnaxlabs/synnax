// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { render } from "@testing-library/react";
import { ReactFlowProvider } from "@xyflow/react";
import { assert, describe, expect, it } from "vitest";

import { Box } from "@/schematic/node/general/box";

const cornerRadius = (config: schematic.BoxNodeConfig): string => {
  const { container } = render(
    <ReactFlowProvider>
      <Box.spec.Preview {...config} />
    </ReactFlowProvider>,
  );
  const shape = container.firstElementChild;
  assert(shape instanceof HTMLElement);
  return shape.style.borderRadius;
};

describe("Box.spec.Preview", () => {
  it("should draw 3 px corners when the border radius is absent", () => {
    expect(cornerRadius(schematic.boxNodeConfigZ.parse({ variant: "box" }))).toBe(
      "3px",
    );
  });

  it("should draw the border radius the config sets", () => {
    expect(
      cornerRadius(schematic.boxNodeConfigZ.parse({ variant: "box", borderRadius: 8 })),
    ).toBe("8px");
  });
});
