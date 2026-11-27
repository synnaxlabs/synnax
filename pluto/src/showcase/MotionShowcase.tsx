// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Input } from "@synnaxlabs/lyra/input";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement, useEffect, useState } from "react";

import { Custom } from "@/schematic/node/common/custom";
import { SubcategorySection } from "@/showcase/SubcategorySection";

// The drawing drops the fill and stroke of its root, so a group carries them.
const svg = (viewBox: string, body: string): string =>
  `<svg xmlns="http://www.w3.org/2000/svg" viewBox="${viewBox}">` +
  `<g fill="none" stroke="#cfd1d4" stroke-width="2">${body}</g></svg>`;

const STATES: schematic.symbol.State[] = [
  { type: "number", key: "value", name: "Value", bounds: { lower: 0, upper: 100 } },
  {
    type: "enumeration",
    key: "alarm",
    name: "Alarm",
    match: { type: "within", bounds: { lower: 50, upper: 101 } },
    default: 0,
  },
];

const spec = (
  markup: string,
  parts: schematic.symbol.Part[],
  looks: schematic.symbol.Look[],
): schematic.symbol.Spec => ({
  drawing: { type: "svg", markup },
  parts,
  handles: [],
  states: STATES,
  commands: [],
  looks,
  interactions: [],
  scale: 2.5,
  strokeScaled: false,
});

const part = (key: string, pivot?: { x: number; y: number }) => ({
  key,
  name: key,
  selectors: [`.${key}`],
  pivot,
});

// The impeller's box is lopsided, so its pivot is the hub at the symbol's center.
const HUB = { x: 0.5, y: 0.5 };

const PUMP = spec(
  svg(
    "0 0 50 50",
    '<circle cx="25" cy="25" r="22"/>' +
      '<g class="impeller"><path d="M25 25L25 8M25 25L39.7 33.5M25 25L10.3 33.5"/>' +
      '<circle cx="25" cy="25" r="3" fill="#cfd1d4"/></g>',
  ),
  [part("impeller", HUB)],
  [
    {
      key: "spin",
      part: "impeller",
      motion: { type: "spin", rate: 1, state: "value" },
    },
  ],
);

const PIPE = spec(
  svg(
    "0 0 100 24",
    '<path d="M0 3H100M0 21H100"/>' +
      '<path class="fluid" d="M0 12H100" stroke="#3fb6c2" stroke-width="4" ' +
      'stroke-dasharray="8 6"/>',
  ),
  [part("fluid")],
  [{ key: "flow", part: "fluid", motion: { type: "flow", rate: 2, state: "value" } }],
);

const LAMP = spec(
  svg(
    "0 0 50 50",
    '<circle cx="25" cy="25" r="22"/>' +
      '<circle class="lamp" cx="25" cy="25" r="15" fill="#f5a524" stroke="none"/>',
  ),
  [part("lamp")],
  [
    {
      key: "blink",
      part: "lamp",
      when: { type: "equals", state: "alarm", values: [1] },
      blink: { period: 600_000_000 },
    },
  ],
);

const TANK = spec(
  svg(
    "0 0 50 90",
    '<rect class="liquid" x="5" y="5" width="40" height="80" rx="18" ' +
      'fill="#3fb6c2" stroke="none"/>' +
      '<rect x="5" y="5" width="40" height="80" rx="18"/>',
  ),
  [part("liquid")],
  [
    {
      key: "level",
      part: "liquid",
      motion: { type: "level", state: "value", side: "bottom" },
    },
  ],
);

const GAUGE = spec(
  svg(
    "0 0 60 40",
    '<path d="M6 36A24 24 0 0 1 54 36"/>' +
      '<path class="needle" d="M30 36L30 16" stroke="#f5a524" stroke-width="3"/>' +
      '<circle cx="30" cy="36" r="3" fill="#cfd1d4"/>',
  ),
  [part("needle", { x: 0.5, y: 0.9 })],
  [
    {
      key: "rotate",
      part: "needle",
      motion: { type: "rotate", state: "value", from: -90, to: 90 },
    },
  ],
);

const ACTUATOR = spec(
  svg(
    "0 0 100 30",
    '<rect x="2" y="5" width="50" height="20"/>' +
      '<g class="piston"><rect x="8" y="9" width="6" height="12" fill="#cfd1d4"/>' +
      '<path d="M14 15H60"/></g>',
  ),
  [part("piston")],
  [
    {
      key: "move",
      part: "piston",
      motion: {
        type: "move",
        state: "value",
        from: { x: 0, y: 0 },
        to: { x: 0.36, y: 0 },
      },
    },
  ],
);

const SPECIMENS = [
  { name: "Spin", note: "Turns at its rate times the value.", spec: PUMP },
  { name: "Flow", note: "Moves the dashes at its rate times the value.", spec: PIPE },
  { name: "Blink", note: "Flashes while the alarm reads 1, at 50 and up.", spec: LAMP },
  { name: "Level", note: "Fills from the bottom to the value.", spec: TANK },
  { name: "Rotate", note: "Turns from -90° to 90° with the value.", spec: GAUGE },
  { name: "Move", note: "Slides with the value.", spec: ACTUATOR },
];

const STRESS = spec(
  svg(
    "0 0 50 50",
    '<circle cx="25" cy="25" r="22"/>' +
      '<path class="impeller" d="M25 25L25 8M25 25L39.7 33.5M25 25L10.3 33.5"/>',
  ),
  [part("impeller", HUB)],
  [{ key: "spin", part: "impeller", motion: { type: "spin", rate: 0.5 } }],
);
const SMALL_STRESS = { ...STRESS, scale: 0.6 };

const useFrameRate = (): number => {
  const [fps, setFPS] = useState(0);
  useEffect(() => {
    const count = { handle: 0, frames: 0, start: performance.now() };
    const frame = (now: number) => {
      count.frames += 1;
      if (now - count.start >= 1000) {
        setFPS(Math.round((count.frames * 1000) / (now - count.start)));
        count.frames = 0;
        count.start = now;
      }
      count.handle = requestAnimationFrame(frame);
    };
    count.handle = requestAnimationFrame(frame);
    return () => cancelAnimationFrame(count.handle);
  }, []);
  return fps;
};

const Stress = (): ReactElement => {
  const [count, setCount] = useState(100);
  const fps = useFrameRate();
  return (
    <Flex.Box y gap="medium">
      <Flex.Box x gap="large" align="center">
        <Input.Item label="Moving parts" x>
          <Input.Numeric
            value={count}
            onChange={setCount}
            bounds={{ lower: 0, upper: 1000 }}
          />
        </Input.Item>
        <Text.Text level="p">{fps} frames a second</Text.Text>
      </Flex.Box>
      <Flex.Box x wrap gap="tiny">
        {Array.from({ length: count }, (_, i) => (
          <Custom.Drawing key={i} spec={SMALL_STRESS} />
        ))}
      </Flex.Box>
    </Flex.Box>
  );
};

export const MotionShowcase = (): ReactElement => {
  const [value, setValue] = useState(60);
  const [stale, setStale] = useState(false);
  const [unknown, setUnknown] = useState(false);
  const [motionDisabled, setMotionDisabled] = useState(false);
  const reading = { value: unknown ? NaN : value, stale };
  const readings = { value: reading, alarm: reading };
  return (
    <Flex.Box y gap="large">
      <SubcategorySection
        title="Looks that move"
        description="Each part moves by the reading of its state. Stale and unknown readings stop spin, flow, and blink. Motion disabled and the Reduce motion setting stop them too, and make level, rotate, and move jump."
      >
        <Flex.Box x gap="large" align="center" wrap>
          <Input.Item label="Value" x>
            <Input.Numeric
              value={value}
              onChange={setValue}
              bounds={{ lower: 0, upper: 100 }}
            />
          </Input.Item>
          <Input.Item label="Stale" x>
            <Input.Switch value={stale} onChange={setStale} />
          </Input.Item>
          <Input.Item label="Unknown" x>
            <Input.Switch value={unknown} onChange={setUnknown} />
          </Input.Item>
          <Input.Item label="Motion disabled" x>
            <Input.Switch value={motionDisabled} onChange={setMotionDisabled} />
          </Input.Item>
        </Flex.Box>
        <Flex.Box x gap="huge" wrap align="end">
          {SPECIMENS.map(({ name, note, spec }) => (
            <Flex.Box key={name} y gap="small" align="center" style={{ width: 160 }}>
              <Custom.Drawing
                spec={spec}
                readings={readings}
                motionDisabled={motionDisabled}
              />
              <Text.Text level="h5">{name}</Text.Text>
              <Text.Text level="small" color={9} style={{ textAlign: "center" }}>
                {note}
              </Text.Text>
            </Flex.Box>
          ))}
        </Flex.Box>
      </SubcategorySection>
      <SubcategorySection
        title="One loop for every part"
        description="Each symbol has one spinning part, and one animation loop draws them all."
      >
        <Stress />
      </SubcategorySection>
    </Flex.Box>
  );
};
