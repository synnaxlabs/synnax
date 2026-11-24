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
import { Indicator } from "@/schematic/node/common/indicator";
import { Label } from "@/schematic/node/common/label";
import { Primitive } from "@/schematic/node/common/primitive";
import type { NodeProps, PreviewProps } from "@/schematic/node/spec";
import { bind } from "@/schematic/node/symbol/bind";
import { Symbol } from "@/schematic/symbol";
import { SymbolNode } from "@/vis/symbolNode";

type Config = schematic.SymbolNodeConfig;

interface Resolved {
  spec?: schematic.symbol.Spec;
  builtIn: boolean;
  missing: boolean;
}

const useResolved = (ref: schematic.symbol.Ref): Resolved => {
  const custom = Symbol.useResolved(ref.type === "custom" ? ref.key : null);
  switch (ref.type) {
    case "builtin": {
      const spec = BUILT_INS.get(ref.name);
      return { spec, builtIn: true, missing: spec == null };
    }
    case "custom":
      return { spec: custom.symbol?.data, builtIn: false, missing: custom.missing };
    case "missing":
      return { builtIn: false, missing: true };
  }
};

// useBound resolves the symbol of a node, with the bindings of its states applied.
const useBound = ({ spec: ref, states }: Pick<Config, "spec" | "states">): Resolved => {
  const { spec, ...rest } = useResolved(ref);
  return { ...rest, spec: spec == null ? undefined : bind.spec(spec, states) };
};

interface DrawnProps extends Resolved, Pick<Config, "color" | "orientation" | "scale"> {
  value: Custom.StateValues;
}

const Drawn = ({
  spec,
  builtIn,
  missing,
  color,
  orientation,
  scale,
  value,
}: DrawnProps): ReactElement => {
  const [nodeColor] = Color.useValues([color]);
  const paints = Custom.usePaints(spec, value, { nodeColor, builtIn });
  const invalid = spec != null && !builtIn && !Custom.isValidSVG(spec.drawing.markup);
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
    <Primitive.Div orientation={orientation} className={CSS.B("symbol")}>
      {spec != null && (
        <Custom.Drawing
          spec={spec}
          builtIn={builtIn}
          orientation={orientation}
          scale={scale}
          paints={paints}
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
  const { label, orientation, scale, color, states, stalenessTimeout } = config;
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
        value={value}
      />
    </Grid.Grid>
  );
};

const NO_VALUES: Custom.StateValues = new Map();

export const Preview = ({
  color,
  orientation,
  scale,
  ...rest
}: PreviewProps<Config>): ReactElement => {
  const bound = useBound(rest);
  return (
    <Drawn
      {...bound}
      color={color}
      orientation={orientation}
      scale={scale}
      value={NO_VALUES}
    />
  );
};
