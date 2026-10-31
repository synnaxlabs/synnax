// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { SyntheticCore } from "@synnaxlabs/client/testutil";
import { Haul } from "@synnaxlabs/lyra/haul";
import { TimeStamp } from "@synnaxlabs/x";
import { screen, waitFor } from "@testing-library/react";
import { ReactFlowProvider } from "@xyflow/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Setpoint } from "@/schematic/node/general/setpoint";
import { telem } from "@/telem/aether";
import { telemTest } from "@/telem/aether/testutil";
import { render } from "@/testutil/render";
import { setpoint } from "@/vis/setpoint/aether";
import { value } from "@/vis/value/aether";

const NODE_KEY = "s1";

describe("Setpoint.Symbol", () => {
  const closers: Array<() => Promise<void>> = [];
  afterEach(async () => {
    for (const close of closers.splice(0)) await close();
  });

  it("should mark the problem of a failed readback", async () => {
    const core = new SyntheticCore({
      rate: 10,
      start: TimeStamp.seconds(100),
      now: TimeStamp.seconds(110),
    });
    const client = telemTest.createClient(core);
    closers.push(async () => await client.feed.close());
    render(
      <ReactFlowProvider>
        <Haul.Provider>
          <div data-id={NODE_KEY}>
            <Setpoint.spec.Node
              nodeKey={NODE_KEY}
              selected={false}
              onConfigChange={vi.fn()}
              config={schematic.setpointNodeConfigZ.parse({
                variant: "setpoint",
                stateChannel: 999,
              })}
            />
          </div>
        </Haul.Provider>
      </ReactFlowProvider>,
      {
        render: true,
        registry: { ...value.REGISTRY, ...setpoint.REGISTRY },
        telem: { factories: [telem.createFactory(client)] },
      },
    );
    await waitFor(() =>
      expect(
        screen.getByRole("img", { name: "Failed to stream channel value" }),
      ).toBeTruthy(),
    );
  });
});
