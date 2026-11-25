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
import { Toggle } from "@/schematic/node/common/toggle";
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

// bindStates applies the bindings of states to the symbol of resolved.
const bindStates = (
  { spec, ...rest }: Resolved,
  states: Config["states"],
): Resolved => ({
  ...rest,
  spec: spec == null ? undefined : bind.spec(spec, states),
});

interface DrawnProps
  extends Resolved, Pick<Config, "color" | "orientation" | "scale" | "motionDisabled"> {
  value: Custom.StateValues;
  readings?: Record<string, Custom.Reading>;
  onClick?: () => void;
  onClickDelay?: Config["onClickDelay"];
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
  onClick,
  onClickDelay,
}: DrawnProps): ReactElement => {
  const [nodeColor] = Color.useValues([color]);
  const paints = Custom.usePaints(spec, value, { nodeColor, builtIn });
  if (missing || failed || invalid)
    return <Warning cause={missing ? "missing" : failed ? "failed" : "invalid"} />;
  const className = CSS.cls(CSS.B("symbol"), !builtIn && CSS.BM("symbol", "custom"));
  const drawing = spec != null && (
    <Custom.Drawing
      spec={spec}
      builtIn={builtIn}
      orientation={orientation}
      scale={scale}
      paints={paints}
      readings={readings}
      motionDisabled={motionDisabled}
    />
  );
  if (onClick == null)
    return (
      <Primitive.Div orientation={orientation} className={className}>
        {drawing}
      </Primitive.Div>
    );
  return (
    <Toggle.Button
      orientation={orientation}
      className={className}
      onClick={onClick}
      onClickDelay={onClickDelay}
    >
      {drawing}
    </Toggle.Button>
  );
};

interface Sendable {
  command: schematic.symbol.Command;
  value: number;
  target: bind.Target;
}

// resolveClick returns what a click on the whole drawing sends, where spec is symbol
// with the node's bindings. It is undefined when a click sends nothing, such as a
// command with no target or one that asks for a value.
const resolveClick = (
  symbol: schematic.symbol.Spec | undefined,
  spec: schematic.symbol.Spec | undefined,
  { disabled, states, commands }: Pick<Config, "disabled" | "states" | "commands">,
  value: ReadonlyMap<string, number>,
  send: (s: Sendable) => void,
): (() => void) | undefined => {
  const interaction = spec?.interactions.find(({ part }) => part == null);
  if (symbol == null || spec == null || interaction == null || disabled)
    return undefined;
  const sendable = (key: string): Sendable | undefined => {
    const command = spec.commands.find((c) => c.key === key);
    if (command?.value.type !== "fixed") return undefined;
    const target = bind.target(command, states, commands);
    return target == null ? undefined : { command, value: command.value.value, target };
  };
  if (interaction.type === "fire") {
    const s = sendable(interaction.command);
    return s == null ? undefined : () => send(s);
  }
  const [a, b] = interaction.commands.map(sendable);
  if (a == null || b == null) return undefined;
  return () => {
    const state = spec.states.find(({ key }) => key === a.command.state);
    const own = symbol.states.find(({ key }) => key === a.command.state);
    // An unknown value sends nothing.
    if (state == null || own == null || !value.has(state.key)) return;
    const current = Custom.matched(state, value);
    // A command value passes through the symbol's match, and the state value through
    // the node's.
    if (current === Custom.matched(own, a.value)) send(b);
    else if (current === Custom.matched(own, b.value)) send(a);
  };
};

export const Node = ({
  nodeKey,
  config,
  selected,
  onConfigChange,
}: NodeProps<Config>): ReactElement => {
  const { label, orientation, scale, color, states, stalenessTimeout, motionDisabled } =
    config;
  const resolved = useResolved(config.spec);
  const bound = bindStates(resolved, states);
  const { readings, set } = SymbolNode.use({
    aetherKey: nodeKey,
    sources: bind.sources(bind.stateChannels(states)),
    sinks: bound.spec == null ? {} : bind.sinks(bound.spec, states, config.commands),
    stalenessTimeout,
  });
  const value = bind.values(states, readings);
  const onClick = resolveClick(
    resolved.spec,
    bound.spec,
    config,
    value,
    ({ target, value: v }) => {
      if (target.type === "channel") set(target.channel.toString(), v);
      else
        onConfigChange({
          states: states.map((s) =>
            s.key === target.state ? { ...s, source: { type: "local", value: v } } : s,
          ),
        });
    },
  );
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
        onClick={onClick}
        onClickDelay={config.onClickDelay}
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
  const bound = bindStates(useResolved(rest.spec), rest.states);
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
