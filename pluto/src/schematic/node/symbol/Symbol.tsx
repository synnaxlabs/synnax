// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import type { ReactElement } from "react";

import { Color } from "@/color";
import { CSS } from "@/css";
import { BUILT_INS } from "@/schematic/node/builtIns";
import { Custom } from "@/schematic/node/common/custom";
import { Grid } from "@/schematic/node/common/grid";
import { Indicator } from "@/schematic/node/common/indicator";
import { Label } from "@/schematic/node/common/label";
import { Primitive } from "@/schematic/node/common/primitive";
import type { NodeProps, PreviewProps } from "@/schematic/node/spec";
import { bind } from "@/schematic/node/symbol/bind";
import { Warning } from "@/schematic/node/symbol/Warning";
import { Symbol } from "@/schematic/symbol";
import { SymbolNode } from "@/vis/symbolNode";

type Config = schematic.SymbolNodeConfig;

interface Resolved {
  spec?: schematic.symbol.Spec;
  builtIn: boolean;
  missing: boolean;
  failed: boolean;
  invalid: boolean;
}

const useResolved = (ref: schematic.symbol.Ref): Resolved => {
  const custom = Symbol.useResolved(ref.type === "custom" ? ref.key : null);
  switch (ref.type) {
    case "builtin": {
      const spec = BUILT_INS.get(ref.name);
      return {
        spec,
        builtIn: true,
        missing: spec == null,
        failed: false,
        invalid: false,
      };
    }
    case "custom": {
      const spec = custom.symbol?.data;
      // The check parses the markup, so it runs here, where it is cached on the spec.
      const invalid = spec != null && !Custom.isValidSVG(spec.drawing.markup);
      const { missing, failed } = custom;
      return { spec, builtIn: false, missing, failed, invalid };
    }
    case "missing":
      return { builtIn: false, missing: true, failed: false, invalid: false };
  }
};

// useBound resolves the symbol of a node, with the bindings of its states applied.
const useBound = ({ spec: ref, states }: Pick<Config, "spec" | "states">): Resolved => {
  const { spec, ...rest } = useResolved(ref);
  return { ...rest, spec: spec == null ? undefined : bind.spec(spec, states) };
};

interface DrawnProps
  extends Resolved, Pick<Config, "color" | "orientation" | "scale" | "motionDisabled"> {
  value: Custom.StateValues;
  readings?: Record<string, Custom.Reading>;
}

const Drawn = ({
  spec,
  builtIn,
  missing,
  failed,
  invalid,
  color,
  orientation,
  scale,
  motionDisabled,
  value,
  readings,
}: DrawnProps): ReactElement => {
  const [nodeColor] = Color.useValues([color]);
  const paints = Custom.usePaints(spec, value, { nodeColor, builtIn });
  if (missing || failed || invalid)
    return <Warning cause={missing ? "missing" : failed ? "failed" : "invalid"} />;
  return (
    <Primitive.Div
      orientation={orientation}
      className={CSS.cls(CSS.B("symbol"), !builtIn && CSS.BM("symbol", "custom"))}
    >
      {spec != null && (
        <Custom.Drawing
          spec={spec}
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
  const { label, orientation, scale, color, states, stalenessTimeout, motionDisabled } =
    config;
  const { readings } = SymbolNode.use({
    aetherKey: nodeKey,
    sources: bind.sources(bind.stateChannels(states)),
    sinks: {},
    stalenessTimeout,
  });
  const bound = useBound(config);
  const value = bind.values(states, readings);
  const unmatched = Custom.useUnmatched(bound.spec, value);
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
      <Indicator.Slot
        stale={bind.stale(states, readings)}
        unknown={bind.dataMissing(states, readings) || unmatched}
      />
      <Drawn
        {...bound}
        color={color}
        orientation={orientation}
        scale={scale}
        motionDisabled={motionDisabled}
        value={value}
        readings={bind.stateReadings(states, readings)}
      />
    </Grid.Grid>
  );
};

const NO_VALUES: Custom.StateValues = new Map();

export const Preview = ({
  color,
  orientation,
  scale,
  motionDisabled,
  ...rest
}: PreviewProps<Config>): ReactElement => {
  const bound = useBound(rest);
  return (
    <Drawn
      {...bound}
      color={color}
      orientation={orientation}
      scale={scale}
      motionDisabled={motionDisabled}
      value={NO_VALUES}
    />
  );
};
