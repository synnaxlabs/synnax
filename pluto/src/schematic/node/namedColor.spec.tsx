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
import { ReactFlowProvider } from "@xyflow/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { beforeAll, beforeEach, describe, expect, it } from "vitest";

import { Primitive } from "@/schematic/node/common/primitive";
import { Gauge } from "@/schematic/node/general/gauge";
import { Input } from "@/schematic/node/general/input";
import { Scale } from "@/schematic/node/general/scale";
import { Setpoint } from "@/schematic/node/general/setpoint";
import { StringDisplay } from "@/schematic/node/general/stringDisplay";
import { Value } from "@/schematic/node/general/value";
import { Tank } from "@/schematic/node/vessels/tank";
import { telemTest } from "@/telem/aether/testutil";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";
import { gauge } from "@/vis/gauge/aether";
import { input } from "@/vis/input/aether";
import { scale } from "@/vis/scale/aether";
import { setpoint } from "@/vis/setpoint/aether";
import { stalenessTest } from "@/vis/staleness/aether/testutil";
import { stringValue } from "@/vis/stringValue/aether";
import { value } from "@/vis/value/aether";

const client = createTestClient();

// The last color each stubbed Aether component received, by field.
const drawn = new Map<string, unknown>();

const REGISTRY = {
  ...input.REGISTRY,
  ...setpoint.REGISTRY,
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

const symbolColor = (root: HTMLElement): string | undefined =>
  root
    .querySelector<HTMLElement>(`[style*="${Primitive.SYMBOL_COLOR}"]`)
    ?.style.getPropertyValue(Primitive.SYMBOL_COLOR);

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
  /** Reads the color the primitive paints, from the DOM or the drawn Aether state. */
  painted: (root: HTMLElement) => unknown;
}

const ENTRIES: Array<[string, Entry]> = [
  [
    "gauge stroke",
    {
      render: (strokeColor) => (
        <Gauge.spec.Node
          {...NODE}
          config={schematic.gaugeNodeConfigZ.parse({ variant: "gauge", strokeColor })}
        />
      ),
      painted: () => drawn.get("gauge"),
    },
  ],
  [
    "input fill",
    {
      render: (fillColor) => (
        <Input.spec.Node
          {...NODE}
          config={schematic.inputNodeConfigZ.parse({ variant: "input", fillColor })}
        />
      ),
      painted: symbolColor,
    },
  ],
  [
    "setpoint fill",
    {
      render: (fillColor) => (
        <Setpoint.spec.Node
          {...NODE}
          config={schematic.setpointNodeConfigZ.parse({
            variant: "setpoint",
            fillColor,
          })}
        />
      ),
      painted: symbolColor,
    },
  ],
  [
    "string display text",
    {
      render: (textColor) => (
        <StringDisplay.spec.Node
          {...NODE}
          config={schematic.stringDisplayNodeConfigZ.parse({
            variant: "string_display",
            textColor,
          })}
        />
      ),
      painted: () => drawn.get("string value"),
    },
  ],
  [
    "string display stroke",
    {
      render: (strokeColor) => (
        <StringDisplay.spec.Node
          {...NODE}
          config={schematic.stringDisplayNodeConfigZ.parse({
            variant: "string_display",
            strokeColor,
          })}
        />
      ),
      painted: symbolColor,
    },
  ],
  [
    "value text",
    {
      render: (textColor) => (
        <Value.spec.Node
          {...NODE}
          config={schematic.valueNodeConfigZ.parse({ variant: "value", textColor })}
        />
      ),
      painted: () => drawn.get("value text"),
    },
  ],
  [
    "value stroke",
    {
      render: (strokeColor) => (
        <Value.spec.Node
          {...NODE}
          config={schematic.valueNodeConfigZ.parse({ variant: "value", strokeColor })}
        />
      ),
      painted: symbolColor,
    },
  ],
  [
    "value fill",
    {
      render: (fillColor) => (
        <Value.spec.Node
          {...NODE}
          config={schematic.valueNodeConfigZ.parse({ variant: "value", fillColor })}
        />
      ),
      painted: () => drawn.get("value fill"),
    },
  ],
  ...(["level", "axis", "text"] as const).map((field): [string, Entry] => [
    `scale ${field}`,
    {
      render: (ref) => (
        <Scale.spec.Node
          {...NODE}
          config={schematic.scaleNodeConfigZ.parse({
            variant: "scale",
            [`${field}Color`]: ref,
          })}
        />
      ),
      painted: () => drawn.get(`scale ${field}`),
    },
  ]),
  [
    "gauge preview stroke",
    {
      render: (strokeColor) => (
        <Gauge.spec.Preview
          {...schematic.gaugeNodeConfigZ.parse({ variant: "gauge", strokeColor })}
        />
      ),
      painted: symbolColor,
    },
  ],
  [
    "scale preview level",
    {
      render: (levelColor) => (
        <Scale.spec.Preview
          {...schematic.scaleNodeConfigZ.parse({ variant: "scale", levelColor })}
        />
      ),
      painted: symbolColor,
    },
  ],
  [
    "scale preview axis",
    {
      render: (axisColor) => (
        <Scale.spec.Preview
          {...schematic.scaleNodeConfigZ.parse({ variant: "scale", axisColor })}
        />
      ),
      // The axis paints a theme variable until the color resolves.
      painted: (root) => root.querySelector('[stroke^="#"]')?.getAttribute("stroke"),
    },
  ],
  [
    "tank preview stroke",
    {
      render: (strokeColor) => (
        <Tank.spec.Preview
          {...schematic.tankNodeConfigZ.parse({ variant: "tank", strokeColor })}
        />
      ),
      painted: symbolColor,
    },
  ],
  [
    "tank preview fill",
    {
      render: (fillColor) => (
        <Tank.spec.Preview
          {...schematic.tankNodeConfigZ.parse({ variant: "tank", fillColor })}
        />
      ),
      painted: (root) =>
        root.querySelector<HTMLElement>(".pluto-tank")?.style.backgroundColor,
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
