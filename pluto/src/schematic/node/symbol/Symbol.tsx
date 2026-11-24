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
import type { MouseEvent, ReactElement } from "react";

import { Color } from "@/color";
import { CSS } from "@/css";
import { BUILT_INS } from "@/schematic/node/builtIns";
import { Control } from "@/schematic/node/common/control";
import { Custom } from "@/schematic/node/common/custom";
import { Grid } from "@/schematic/node/common/grid";
import { Indicator } from "@/schematic/node/common/indicator";
import { Label } from "@/schematic/node/common/label";
import { Primitive } from "@/schematic/node/common/primitive";
import { Toggle } from "@/schematic/node/common/toggle";
import type { NodeProps, PreviewProps } from "@/schematic/node/spec";
import { bind } from "@/schematic/node/symbol/bind";
import {
  clickable,
  commandChannels,
  fixedValue,
  interactionAt,
  pick,
  sinks,
  target,
} from "@/schematic/node/symbol/command";
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

interface Bound extends Resolved {
  /** symbol is the spec before the bindings of the node apply. */
  symbol?: schematic.symbol.Spec;
}

// useBound resolves the symbol of a node, with the bindings of its states applied.
const useBound = ({ spec: ref, states }: Pick<Config, "spec" | "states">): Bound => {
  const { spec, ...rest } = useResolved(ref);
  return {
    ...rest,
    symbol: spec,
    spec: spec == null ? undefined : bind.spec(spec, states),
  };
};

interface DrawnProps
  extends
    Resolved,
    Pick<Config, "color" | "orientation" | "scale">,
    Partial<Pick<Config, "onClickDelay">> {
  value: Custom.StateValues;
  /** onClick makes the drawing a button. */
  onClick?: (e: MouseEvent<HTMLElement>) => void;
}

const Drawn = ({
  spec,
  builtIn,
  missing,
  color,
  orientation,
  scale,
  value,
  onClick,
  onClickDelay,
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
  const drawing = spec != null && (
    <Custom.Drawing
      spec={spec}
      builtIn={builtIn}
      orientation={orientation}
      scale={scale}
      paints={paints}
    />
  );
  if (onClick == null)
    return (
      <Primitive.Div orientation={orientation} className={CSS.B("symbol")}>
        {drawing}
      </Primitive.Div>
    );
  return (
    <Toggle.Button
      className={CSS.B("symbol")}
      orientation={orientation}
      onClick={onClick}
      onClickDelay={onClickDelay}
    >
      {drawing}
    </Toggle.Button>
  );
};

const DEFAULT_CONTROL = Control.stateConfigZ.parse({});

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
    color,
    states,
    stalenessTimeout,
    disabled,
    onClickDelay,
    control = DEFAULT_CONTROL,
  } = config;
  const bound = useBound(config);
  const { symbol, spec } = bound;
  const channels = commandChannels(symbol, config);
  const { readings, set } = SymbolNode.use({
    aetherKey: nodeKey,
    sources: bind.sources(bind.stateChannels(states)),
    sinks: sinks(channels),
    stalenessTimeout,
  });
  const value = bind.values(states, readings);
  const unmatched = Custom.useUnmatched(spec, value);
  const scaleResize = Grid.useScaleResize(config, onConfigChange);
  const interactions = disabled || symbol == null ? [] : clickable(symbol, config);
  const send = (command: schematic.symbol.Command, v: number): void => {
    const t = target(command, config);
    if (t == null) throw new Error(`command ${command.key} has no target`);
    if (t.type === "channel") return set(String(t.channel), v);
    onConfigChange({
      states: states.map((b) =>
        b.key === t.state ? { ...b, source: { type: "local", value: v } } : b,
      ),
    });
  };
  const handleClick = ({ target: el }: MouseEvent<HTMLElement>): void => {
    if (symbol == null || spec == null || !(el instanceof Element)) return;
    const root = el.closest(`.${CSS.B("symbol")}`);
    if (root == null) return;
    const interaction = interactionAt(symbol, interactions, root, el);
    if (interaction == null) return;
    const command = pick(symbol, spec, interaction, value);
    if (command != null) send(command, fixedValue(command));
  };
  const badges = {
    stale: bind.stale(states, readings),
    unknown: bind.dataMissing(states, readings) || unmatched,
  };
  return (
    <Grid.Grid
      editable={selected}
      nodeKey={nodeKey}
      orientation={orientation}
      onRotate={onConfigChange}
      {...scaleResize}
    >
      <Label.Label config={label} onChange={onConfigChange} />
      {channels.length === 1 ? (
        <Control.State config={control} channel={channels[0]} onChange={onConfigChange}>
          <Indicator.Badges {...badges} />
        </Control.State>
      ) : (
        <Indicator.Slot {...badges} />
      )}
      <Drawn
        {...bound}
        color={color}
        orientation={orientation}
        scale={scale}
        value={value}
        onClick={interactions.length > 0 ? handleClick : undefined}
        onClickDelay={onClickDelay}
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
