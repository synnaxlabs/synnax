// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, table } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { box, id, TimeSpan } from "@synnaxlabs/x";
import { render, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Aether } from "@/aether";
import { aether } from "@/aether/aether";
import { Cell } from "@/table/cells";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";
import { canvasTest } from "@/vis/render/testutil";
import { value } from "@/vis/value/aether";

const valueState = z.object({ stalenessTimeout: z.number().optional() }).loose();

// Stands in for the value worker component, so the spec reads the state that the cell
// sends to it.
class CapturingValue extends aether.Leaf<typeof valueState> {
  static readonly TYPE = "value";
  static readonly states: Array<z.infer<typeof valueState>> = [];
  schema = valueState;

  afterUpdate(): void {
    CapturingValue.states.push(this.state);
  }
}

const client = createTestClient();

const renderLiveValue = async (channel: number) => {
  const Providers = await createAsyncSynnaxWrapper({
    client,
    additionalRegistry: value.REGISTRY,
    telemFactories: [],
    renderContext: canvasTest.record(),
  });
  return render(
    <Providers>
      <table>
        <tbody>
          <tr>
            <Cell.Value
              {...table.valueCellConfigZ.parse({ variant: "value", channel })}
              cellKey={id.create()}
              box={box.construct(0, 0, 100, 20)}
              selected={false}
              editable={false}
              onSelect={vi.fn()}
              onChange={vi.fn()}
            />
          </tr>
        </tbody>
      </table>
    </Providers>,
  );
};

describe("Cells.Value deleted channel", () => {
  it("should say Deleted channel for a channel the Core deleted", async () => {
    const ch = await client.channels.create({
      name: `cell_${id.create()}`,
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    await client.channels.delete(ch.key);
    const { findByText } = await renderLiveValue(ch.key);
    expect(await findByText("Deleted channel")).toBeTruthy();
  });

  it("should say nothing while its channel exists", async () => {
    const ch = await client.channels.create({
      name: `cell_${id.create()}`,
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    const { container, queryByText } = await renderLiveValue(ch.key);
    await waitFor(() => expect(container.querySelector("td")).not.toBeNull());
    expect(queryByText("Deleted channel")).toBeNull();
  });
});

describe("Cells.Value", () => {
  it("should send its staleness timeout to the worker in nanoseconds", async () => {
    const [workerSide, mainSide] = aether.createMockPair();
    aether.render({
      worker: workerSide,
      registry: { [CapturingValue.TYPE]: CapturingValue },
    });
    render(
      <Aether.Provider worker={mainSide}>
        <table>
          <tbody>
            <tr>
              <Cell.Value
                {...table.valueCellConfigZ.parse({
                  variant: "value",
                  stalenessTimeout: 2,
                })}
                cellKey="c1"
                box={box.construct(0, 0, 100, 20)}
                selected={false}
                editable={false}
                onSelect={vi.fn()}
                onChange={vi.fn()}
              />
            </tr>
          </tbody>
        </table>
      </Aether.Provider>,
    );
    await waitFor(() =>
      expect(CapturingValue.states.at(-1)?.stalenessTimeout).toBe(
        Number(TimeSpan.seconds(2).valueOf()),
      ),
    );
  });
});
