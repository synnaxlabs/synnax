// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, table } from "@synnaxlabs/client";
import { createTestClient, waitForStreamLive } from "@synnaxlabs/client/testutil";
import { box, color as xcolor, id, TimeSpan } from "@synnaxlabs/x";
import { render, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Aether } from "@/aether";
import { aether } from "@/aether/aether";
import { Cell } from "@/table/cells";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const valueState = z.object({ stalenessTimeout: z.number().optional() }).loose();

// Stands in for the value worker component, so the spec reads the state that the cell
// sends to it.
class CapturingValue extends aether.Leaf<typeof valueState> {
  static readonly TYPE = "value";
  static readonly states: Array<z.infer<typeof valueState>> = [];
  override schema = valueState;

  override afterUpdate(): void {
    CapturingValue.states.push(this.state);
  }
}

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

  it("should send and repaint its named text, fill, and staleness colors", async () => {
    const client = createTestClient();
    const wrapper = await createAsyncSynnaxWrapper({
      client,
      additionalRegistry: { [CapturingValue.TYPE]: CapturingValue },
    });
    await waitForStreamLive(client.connection);
    const [text, fill, stale] = await Promise.all(
      ["#111111", "#222222", "#333333"].map(
        async (light) =>
          await client.colors.create({ name: `value-${id.create()}`, light }),
      ),
    );
    render(
      <table>
        <tbody>
          <tr>
            <Cell.Value
              {...table.valueCellConfigZ.parse({
                variant: "value",
                textColor: { type: "named", key: text.key },
                fillColor: { type: "named", key: fill.key },
                stalenessColor: { type: "named", key: stale.key },
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
      </table>,
      { wrapper },
    );
    const hex = (c: unknown): string | undefined => {
      const parsed = xcolor.crudeZ.safeParse(c);
      return parsed.success ? xcolor.hex(parsed.data) : undefined;
    };
    const hexes = (cs: string[]): string[] => cs.map((c) => xcolor.hex(c));
    const sent = (): Array<string | undefined> => {
      const state = CapturingValue.states.at(-1);
      const background = z
        .object({ props: z.unknown() })
        .safeParse(state?.backgroundTelem);
      return [
        hex(state?.color),
        hex(background.success ? background.data.props : undefined),
        hex(state?.stalenessColor),
      ];
    };
    await waitFor(() =>
      expect(sent()).toEqual(hexes(["#111111", "#222222", "#333333"])),
    );
    await Promise.all(
      [text, fill, stale].map(
        async ({ key }, i) =>
          await client.colors.dispatch(
            key,
            color.set({
              path: ["light"],
              value: xcolor.construct(["#444444", "#555555", "#666666"][i]),
            }),
          ),
      ),
    );
    await waitFor(() =>
      expect(sent()).toEqual(hexes(["#444444", "#555555", "#666666"])),
    );
  });
});

describe("Cells.Text", () => {
  it("should fill with its named color and repaint when the color changes", async () => {
    const client = createTestClient();
    const wrapper = await createAsyncSynnaxWrapper({ client });
    await waitForStreamLive(client.connection);
    const named = await client.colors.create({
      name: `fill-${id.create()}`,
      light: "#123456",
    });
    const { container } = render(
      <table>
        <tbody>
          <tr>
            <Cell.Text
              {...table.textCellConfigZ.parse({
                variant: "text",
                value: "A",
                fillColor: { type: "named", key: named.key },
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
      </table>,
      { wrapper },
    );
    const fill = (): string | undefined =>
      container.querySelector<HTMLElement>("#c1")?.style.backgroundColor;
    await waitFor(() => expect(fill()).toBe("rgb(18, 52, 86)"));
    await client.colors.dispatch(
      named.key,
      color.set({ path: ["light"], value: xcolor.construct("#654321") }),
    );
    await waitFor(() => expect(fill()).toBe("rgb(101, 67, 33)"));
  });
});
