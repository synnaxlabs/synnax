// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, schematic } from "@synnaxlabs/client";
import { createTestClient, waitForStreamLive } from "@synnaxlabs/client/testutil";
import { Haul } from "@synnaxlabs/lyra/haul";
import { color as xcolor, uuid } from "@synnaxlabs/x";
import { render, waitFor } from "@testing-library/react";
import { type Node as FlowNode, ReactFlowProvider } from "@xyflow/react";
import type { FC, PropsWithChildren, ReactElement } from "react";
import { beforeAll, beforeEach, describe, expect, it } from "vitest";

import { Jumps } from "@/schematic/edge/common/jumps";
import { Pipe } from "@/schematic/edge/pipe";
import { Primitive } from "@/schematic/node/common/primitive";
import { Box } from "@/schematic/node/general/box";
import { Button } from "@/schematic/node/general/button";
import { Circle } from "@/schematic/node/general/circle";
import { Gauge } from "@/schematic/node/general/gauge";
import { Input } from "@/schematic/node/general/input";
import { Line } from "@/schematic/node/general/line";
import { OffPageReference } from "@/schematic/node/general/offPageReference";
import { Polygon } from "@/schematic/node/general/polygon";
import { Scale } from "@/schematic/node/general/scale";
import { Select } from "@/schematic/node/general/select";
import { Setpoint } from "@/schematic/node/general/setpoint";
import { StateIndicator } from "@/schematic/node/general/stateIndicator";
import { StringDisplay } from "@/schematic/node/general/stringDisplay";
import { TextBox } from "@/schematic/node/general/textBox";
import { Value } from "@/schematic/node/general/value";
import type { NodeProps } from "@/schematic/node/spec";
import { Cylinder } from "@/schematic/node/vessels/cylinder";
import { Tank } from "@/schematic/node/vessels/tank";
import { telemTest } from "@/telem/aether/testutil";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";
import { button } from "@/vis/button/aether";
import { diagram } from "@/vis/diagram/aether";
import { gauge } from "@/vis/gauge/aether";
import { input } from "@/vis/input/aether";
import { scale } from "@/vis/scale/aether";
import { setpoint } from "@/vis/setpoint/aether";
import { stalenessTest } from "@/vis/staleness/aether/testutil";
import { stateIndicator } from "@/vis/stateIndicator/aether";
import { stringValue } from "@/vis/stringValue/aether";
import { value } from "@/vis/value/aether";

const client = createTestClient();

// The last color each stubbed Aether component received, by field.
const drawn = new Map<string, unknown>();

const REGISTRY = {
  ...button.REGISTRY,
  ...input.REGISTRY,
  ...setpoint.REGISTRY,
  ...stateIndicator.REGISTRY,
  ...stalenessTest.createRegistry(gauge.Gauge.TYPE, gauge.Gauge.z, (s) =>
    drawn.set("gauge", s.color),
  ),
  ...stalenessTest.createRegistry(
    stringValue.StringValue.TYPE,
    stringValue.StringValue.z,
    (s) => drawn.set("string value", s.color),
  ),
  ...stalenessTest.createRegistry(value.Value.TYPE, value.Value.z, (s) => {
    drawn.set("value text", s.color);
    drawn.set("value fill", s.backgroundTelem.props);
  }),
  ...stalenessTest.createRegistry(scale.Scale.TYPE, scale.Scale.z, (s) => {
    drawn.set("scale level", s.color);
    drawn.set("scale axis", s.axisColor);
    drawn.set("scale text", s.textColor);
  }),
};

const NODE = { nodeKey: "n1", selected: false, onConfigChange: () => {} };

/** Reads the color the primitive paints, from the DOM or the drawn Aether state. */
type Painted = (root: HTMLElement) => unknown;

const symbolColor: Painted = (root) =>
  root
    .querySelector<HTMLElement>(`[style*="${Primitive.SYMBOL_COLOR}"]`)
    ?.style.getPropertyValue(Primitive.SYMBOL_COLOR);

const drew =
  (field: string): Painted =>
  () =>
    drawn.get(field);

const tankFill: Painted = (root) =>
  root.querySelector<HTMLElement>(".pluto-tank")?.style.backgroundColor;

// A fill that has not resolved is a theme variable, not a hex color.
const svgFill: Painted = (root) =>
  root.querySelector('[fill^="#"]')?.getAttribute("fill");

// The axis paints a theme variable until the color resolves.
const axisStroke: Painted = (root) =>
  root.querySelector('[stroke^="#"]')?.getAttribute("stroke");

const textColor: Painted = (root) =>
  Array.from(root.querySelectorAll<HTMLElement>("*")).find(
    (el) => el.style.color !== "",
  )?.style.color;

// The red, green, and blue of a painted color: a crude color, a symbol color variable,
// or a CSS rgb() string.
const rgb = (painted: unknown): string | undefined => {
  if (painted == null || painted === "") return undefined;
  if (typeof painted === "string" && !painted.startsWith("#"))
    return painted
      .replace(/^rgba?\(|\)$/g, "")
      .split(", ")
      .slice(0, 3)
      .join(", ");
  return xcolor.rgbString(xcolor.crudeZ.parse(painted));
};

interface Entry {
  render: (ref: color.Ref) => ReactElement;
  painted: Painted;
}

/**
 * A color field of a symbol: its label, its config key, and how to read its color on
 * the node and on the preview. A null reader skips that render.
 */
type Field = [
  label: string,
  key: string,
  node: Painted | null,
  preview: Painted | null,
];

interface SymbolSpec<C extends object> {
  key: string;
  Node: FC<NodeProps<C>>;
  Preview: FC<C>;
}

/** @returns an entry for each field of the symbol, on its node and on its preview. */
const symbolEntries = <C extends object>(
  spec: SymbolSpec<C>,
  configZ: { parse: (crude: unknown) => C },
  fields: Field[],
): Array<[string, Entry]> => {
  const name = spec.key.replaceAll("_", " ");
  const config = (key: string, ref: color.Ref): C =>
    configZ.parse({ variant: spec.key, [key]: ref });
  return fields.flatMap(([label, key, node, preview]) => {
    const entries: Array<[string, Entry]> = [];
    if (node != null)
      entries.push([
        `${name} ${label}`,
        {
          render: (ref) => <spec.Node {...NODE} config={config(key, ref)} />,
          painted: node,
        },
      ]);
    if (preview != null)
      entries.push([
        `${name} preview ${label}`,
        { render: (ref) => <spec.Preview {...config(key, ref)} />, painted: preview },
      ]);
    return entries;
  });
};

const EDGE_SIZE = { width: 40, height: 40 };

const EDGE_NODES: FlowNode[] = [
  { id: "a", position: { x: 0, y: 0 }, measured: EDGE_SIZE, data: {} },
  { id: "b", position: { x: -200, y: 100 }, measured: EDGE_SIZE, data: {} },
];

const ENTRIES: Array<[string, Entry]> = [
  ...symbolEntries(Gauge.spec, schematic.gaugeNodeConfigZ, [
    ["stroke", "strokeColor", drew("gauge"), symbolColor],
  ]),
  ...symbolEntries(Input.spec, schematic.inputNodeConfigZ, [
    ["fill", "fillColor", symbolColor, symbolColor],
  ]),
  ...symbolEntries(Setpoint.spec, schematic.setpointNodeConfigZ, [
    ["fill", "fillColor", symbolColor, symbolColor],
  ]),
  ...symbolEntries(StringDisplay.spec, schematic.stringDisplayNodeConfigZ, [
    ["text", "textColor", drew("string value"), textColor],
    ["stroke", "strokeColor", symbolColor, symbolColor],
  ]),
  ...symbolEntries(Value.spec, schematic.valueNodeConfigZ, [
    ["text", "textColor", drew("value text"), textColor],
    ["stroke", "strokeColor", symbolColor, symbolColor],
    ["fill", "fillColor", drew("value fill"), null],
  ]),
  ...symbolEntries(Scale.spec, schematic.scaleNodeConfigZ, [
    ["level", "levelColor", drew("scale level"), symbolColor],
    ["axis", "axisColor", drew("scale axis"), axisStroke],
    ["text", "textColor", drew("scale text"), null],
  ]),
  ...symbolEntries(Tank.spec, schematic.tankNodeConfigZ, [
    ["stroke", "strokeColor", symbolColor, symbolColor],
    ["fill", "fillColor", tankFill, tankFill],
  ]),
  ...symbolEntries(Box.spec, schematic.boxNodeConfigZ, [
    ["stroke", "strokeColor", symbolColor, symbolColor],
    ["fill", "fillColor", tankFill, tankFill],
  ]),
  ...symbolEntries(Cylinder.spec, schematic.cylinderNodeConfigZ, [
    ["stroke", "strokeColor", symbolColor, symbolColor],
    ["fill", "fillColor", svgFill, svgFill],
  ]),
  ...symbolEntries(Polygon.spec, schematic.polygonNodeConfigZ, [
    ["stroke", "strokeColor", symbolColor, symbolColor],
    ["fill", "fillColor", svgFill, svgFill],
  ]),
  // The preview renders the same symbol as the node.
  ...symbolEntries(Circle.spec, schematic.circleNodeConfigZ, [
    ["stroke", "strokeColor", symbolColor, null],
    ["fill", "fillColor", svgFill, null],
  ]),
  ...symbolEntries(Line.spec, schematic.lineNodeConfigZ, [
    ["stroke", "strokeColor", symbolColor, symbolColor],
  ]),
  ...symbolEntries(Button.spec, schematic.buttonNodeConfigZ, [
    ["fill", "fillColor", symbolColor, symbolColor],
  ]),
  ...symbolEntries(Select.spec, schematic.selectNodeConfigZ, [
    ["fill", "fillColor", symbolColor, symbolColor],
  ]),
  ...symbolEntries(StateIndicator.spec, schematic.stateIndicatorNodeConfigZ, [
    ["stroke", "strokeColor", symbolColor, symbolColor],
  ]),
  ...symbolEntries(TextBox.spec, schematic.textBoxNodeConfigZ, [
    ["text", "textColor", symbolColor, symbolColor],
  ]),
  ...symbolEntries(OffPageReference.spec, schematic.offPageReferenceNodeConfigZ, [
    ["fill", "fillColor", symbolColor, symbolColor],
  ]),
  [
    "pipe edge stroke",
    {
      render: (strokeColor) => (
        <ReactFlowProvider defaultNodes={EDGE_NODES}>
          <Jumps.Context value={Jumps.createStore()}>
            <svg>
              <Pipe.spec.Edge
                edgeKey="e1"
                source={diagram.createEndpoint(40, 20, "right")}
                target={diagram.createEndpoint(-160, 120, "right")}
                sourceNode="a"
                targetNode="b"
                selected={false}
                config={{ ...Pipe.spec.defaultConfig(), strokeColor }}
                onChange={() => {}}
              />
            </svg>
          </Jumps.Context>
        </ReactFlowProvider>
      ),
      painted: symbolColor,
    },
  ],
];

describe("Schematic named colors", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({
      client,
      additionalRegistry: REGISTRY,
      telemFactories: [new telemTest.TestFactory()],
    });
    await waitForStreamLive(client.connection);
  });

  beforeEach(() => drawn.clear());

  it.each(ENTRIES)("should paint and repaint a named %s", async (_, entry) => {
    const named = await client.colors.create({
      name: `named-${uuid.create()}`,
      light: "#123456",
    });
    const { container } = render(
      <ReactFlowProvider>
        <Haul.Provider>
          <div data-id={NODE.nodeKey}>
            {entry.render({ type: "named", key: named.key })}
          </div>
        </Haul.Provider>
      </ReactFlowProvider>,
      { wrapper },
    );
    await waitFor(() => expect(rgb(entry.painted(container))).toBe("18, 52, 86"));
    await client.colors.dispatch(
      named.key,
      color.set({ path: ["light"], value: xcolor.construct("#abcdef") }),
    );
    await waitFor(() => expect(rgb(entry.painted(container))).toBe("171, 205, 239"));
  });
});
