// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { Haul } from "@synnaxlabs/lyra/haul";
import { type text } from "@synnaxlabs/x";
import { act, render, type RenderResult } from "@testing-library/react";
import { ReactFlowProvider } from "@xyflow/react";
import { describe, expect, it, vi } from "vitest";

import { CSS } from "@/css";
import { Setpoint } from "@/schematic/node/general/setpoint";
import { telemTest } from "@/telem/aether/testutil";
import { createSynnaxWrapper } from "@/testutil/Synnax";
import { canvasTest } from "@/vis/render/testutil";
import { setpoint } from "@/vis/setpoint/aether";
import { value } from "@/vis/value/aether";

const NODE_KEY = "s1";

const Wrapper = createSynnaxWrapper({
  client: null,
  additionalRegistry: { ...value.REGISTRY, ...setpoint.REGISTRY },
  telemFactories: [new telemTest.TestFactory()],
  renderContext: canvasTest.record(),
});

const readoutTag = async (level: text.Level): Promise<string | undefined> => {
  let result: RenderResult | undefined;
  await act(async () => {
    result = render(
      <Wrapper>
        <ReactFlowProvider>
          <Haul.Provider>
            <Setpoint.spec.Node
              nodeKey={NODE_KEY}
              selected={false}
              onConfigChange={vi.fn()}
              config={schematic.setpointNodeConfigZ.parse({
                variant: "setpoint",
                stateChannel: 7,
                level,
              })}
            />
          </Haul.Provider>
        </ReactFlowProvider>
      </Wrapper>,
    );
  });
  return result?.container.querySelector(`.${CSS.BE("symbol-field", "readout-value")}`)
    ?.tagName;
};

describe("Setpoint.Symbol", () => {
  it("should size the readout from the field's text level", async () => {
    expect(await readoutTag("h4")).toBe("P");
    expect(await readoutTag("small")).toBe("SMALL");
  });
});
