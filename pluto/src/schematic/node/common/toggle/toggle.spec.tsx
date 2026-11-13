// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";
import { render } from "@testing-library/react";
import { ReactFlowProvider } from "@xyflow/react";
import { describe, expect, it } from "vitest";

import { Node } from "@/schematic/node";
import { Primitive } from "@/schematic/node/common/primitive";
import { createSynnaxWrapper } from "@/testutil/Synnax";
import { toggle } from "@/vis/toggle/aether";

const SynnaxWrapper = createSynnaxWrapper({
  client: null,
  additionalRegistry: toggle.REGISTRY,
  telemFactories: [],
});

describe("createToggle", () => {
  it("should hold a valve for its stored delay in nanoseconds", () => {
    const Valve = Node.resolveSpec("valve").Node;
    const config = Node.createConfig({
      variant: "valve",
      onClickDelay: Number(TimeSpan.milliseconds(500).valueOf()),
    });
    const { container } = render(
      <SynnaxWrapper>
        <ReactFlowProvider>
          <Valve
            nodeKey="v1"
            selected={false}
            config={config}
            onConfigChange={() => {}}
          />
        </ReactFlowProvider>
      </SynnaxWrapper>,
    );
    const button = container.querySelector("button");
    expect(button?.style.getPropertyValue(Primitive.TOGGLE_DELAY)).toBe("0.5s");
  });
});
