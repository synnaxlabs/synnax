// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Note } from "@synnaxlabs/lyra/note";
import { Text } from "@synnaxlabs/lyra/text";
import type { ReactElement } from "react";

import { Color } from "@/color";
import { CSS } from "@/css";
import { BUILT_INS } from "@/schematic/node/builtIns";
import { Custom } from "@/schematic/node/common/custom";
import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import { Primitive } from "@/schematic/node/common/primitive";
import { Stale } from "@/schematic/node/common/stale";
import type { NodeProps, PreviewProps } from "@/schematic/node/spec";
import { bind } from "@/schematic/node/symbol/bind";
import { Symbol } from "@/schematic/symbol";
import { SymbolNode } from "@/vis/symbolNode";

type Config = schematic.SymbolNodeConfig;

interface Resolved {
  spec?: schematic.symbol.Spec;
  builtIn: boolean;
  missing: boolean;
  invalid: boolean;
}

const useSpec = (ref: schematic.symbol.Ref): Resolved => {
  const custom = Symbol.useResolved(ref.type === "custom" ? ref.key : null);
  switch (ref.type) {
    case "builtin": {
      const spec = BUILT_INS.get(ref.name);
      return { spec, builtIn: true, missing: spec == null, invalid: false };
    }
    case "custom": {
      const spec = custom.symbol?.data;
      // The check parses the markup, so it runs here, where it is cached on the spec.
      const invalid = spec != null && !Custom.isValidSVG(spec.drawing.markup);
      return { spec, builtIn: false, missing: custom.missing, invalid };
    }
    case "missing":
      return { builtIn: false, missing: true, invalid: false };
  }
};

interface DrawnProps extends Pick<
  Config,
  "spec" | "color" | "states" | "orientation" | "scale" | "motionDisabled"
> {
  value: Custom.StateValues;
  readings?: Record<string, Custom.Reading>;
}

const Drawn = ({
  spec: ref,
  color,
  states,
  orientation,
  scale,
  motionDisabled,
  value,
  readings,
}: DrawnProps): ReactElement => {
  const { spec, builtIn, missing, invalid } = useSpec(ref);
  const bound = spec == null ? undefined : bind.spec(spec, states);
  const [nodeColor] = Color.useValues([color]);
  const paints = Custom.usePaints(bound, value, { nodeColor, builtIn });
  if (missing || invalid)
    return (
      <Note.Note variant="warning">
        <Text.Text level="p" status="warning">
          <Icon.Warning />
          {missing ? "Missing symbol" : "Invalid symbol"}
        </Text.Text>
      </Note.Note>
    );
  return (
    <Primitive.Div
      orientation={orientation}
      className={CSS.cls(CSS.B("symbol"), !builtIn && CSS.BM("symbol", "custom"))}
    >
      {bound != null && (
        <Custom.Drawing
          spec={bound}
          builtIn={builtIn}
          orientation={orientation}
          scale={scale}
          paints={paints}
          // A custom symbol draws the node color only where its looks use a node paint.
          strokeColor={builtIn ? nodeColor : undefined}
          readings={readings}
          motionDisabled={motionDisabled}
        />
      )}
    </Primitive.Div>
  );
};

export const Node = ({
  nodeKey,
  config,
  selected,
  onConfigChange,
}: NodeProps<Config>): ReactElement => {
  const {
    label,
    orientation,
    scale,
    spec,
    color,
    states,
    stalenessTimeout,
    motionDisabled,
  } = config;
  const { readings } = SymbolNode.use({
    aetherKey: nodeKey,
    sources: bind.sources(bind.stateChannels(states)),
    sinks: {},
    stalenessTimeout,
  });
  const scaleResize = Grid.useScaleResize(config, onConfigChange);
  return (
    <Grid.Grid
      editable={selected}
      nodeKey={nodeKey}
      orientation={orientation}
      onRotate={onConfigChange}
      {...scaleResize}
    >
      <Label.Label config={label} onChange={onConfigChange} />
      {bind.stale(states, readings) && <Stale.Indicator />}
      <Drawn
        spec={spec}
        color={color}
        states={states}
        orientation={orientation}
        scale={scale}
        motionDisabled={motionDisabled}
        value={bind.values(states, readings)}
        readings={bind.stateReadings(states, readings)}
      />
    </Grid.Grid>
  );
};

const NO_VALUES: Custom.StateValues = new Map();

export const Preview = ({
  spec,
  color,
  states,
  orientation,
  scale,
  motionDisabled,
}: PreviewProps<Config>): ReactElement => (
  <Drawn
    spec={spec}
    color={color}
    states={states}
    orientation={orientation}
    scale={scale}
    motionDisabled={motionDisabled}
    value={NO_VALUES}
  />
);
