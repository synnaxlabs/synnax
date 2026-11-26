// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { type ReactElement, type Ref, useState } from "react";

import { Color } from "@/color";
import { CSS } from "@/css";
import { Control } from "@/schematic/node/common/control";
import { Custom } from "@/schematic/node/common/custom";
import { Grid } from "@/schematic/node/common/grid";
import { Indicator } from "@/schematic/node/common/indicator";
import { Label } from "@/schematic/node/common/label";
import { Primitive } from "@/schematic/node/common/primitive";
import { Toggle } from "@/schematic/node/common/toggle";
import type { NodeProps, PreviewProps } from "@/schematic/node/spec";
import { bind } from "@/schematic/node/symbol/bind";
import { Prompt, type Request, type Sendable } from "@/schematic/node/symbol/Prompt";
import { type Resolved, useResolved } from "@/schematic/node/symbol/resolve";
import { Warning } from "@/schematic/node/symbol/Warning";
import { Tooltip } from "@/schematic/tooltip";
import { SymbolNode } from "@/vis/symbolNode";

type Config = schematic.SymbolNodeConfig;

const DEFAULT_CONTROL = Control.stateConfigZ.parse({});

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
  buttonRef?: Ref<HTMLButtonElement>;
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
  buttonRef,
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
      ref={buttonRef}
      orientation={orientation}
      className={className}
      onClick={onClick}
      onClickDelay={onClickDelay}
    >
      {drawing}
    </Toggle.Button>
  );
};

interface Reads {
  /** value holds the value of each state that has one, by its key. */
  value: ReadonlyMap<string, number>;
  /** missing holds the key of each state whose channel has no data. */
  missing: ReadonlySet<string>;
  /** commanded holds the newest value on the command channel of each state. */
  commanded: ReadonlyMap<string, number>;
}

interface ClickActions {
  send: (s: Sendable) => void;
  ask: (r: Request) => void;
}

// resolveClick returns what a click on the whole drawing does, where spec is symbol
// with the node's bindings. It is undefined when a click does nothing, such as a
// command with no target.
const resolveClick = (
  symbol: schematic.symbol.Spec | undefined,
  spec: schematic.symbol.Spec | undefined,
  { disabled, states, commands }: Pick<Config, "disabled" | "states" | "commands">,
  { value, missing, commanded }: Reads,
  { send, ask }: ClickActions,
): (() => void) | undefined => {
  const interaction = spec?.interactions.find(({ part }) => part == null);
  if (symbol == null || spec == null || interaction == null || disabled)
    return undefined;
  if (interaction.type === "fire") {
    const command = spec.commands.find(({ key }) => key === interaction.command);
    const target = command == null ? undefined : bind.target(command, states, commands);
    if (command == null || target == null) return undefined;
    if (command.value.type === "fixed") {
      const s = { command, value: command.value.value, target };
      return () => send(s);
    }
    const state = spec.states.find(({ key }) => key === command.state);
    if (state == null) return undefined;
    return () => ask({ type: "ask", command, target, state });
  }
  const [a, b] = interaction.commands.map((key): Sendable | undefined => {
    const command = spec.commands.find((c) => c.key === key);
    if (command?.value.type !== "fixed") return undefined;
    const target = bind.target(command, states, commands);
    return target == null ? undefined : { command, value: command.value.value, target };
  });
  const state = spec.states.find(({ key }) => key === a?.command.state);
  const own = symbol.states.find(({ key }) => key === a?.command.state);
  if (a == null || b == null || state == null || own == null) return undefined;
  return () => {
    // Command values, also the newest one on the command channel, pass through the
    // symbol's match, and the state value through the node's.
    const current = Custom.matched(state, value);
    const sent = commanded.get(state.key);
    const known =
      !missing.has(state.key) &&
      (sent == null || Custom.matched(own, sent) === current);
    if (known && current === Custom.matched(own, a.value)) send(b);
    else if (known && current === Custom.matched(own, b.value)) send(a);
    else ask({ type: "choose", options: [a, b] });
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
  const commandChannels =
    bound.spec == null ? [] : bind.commandChannels(bound.spec, states, config.commands);
  const { readings, set } = SymbolNode.use({
    aetherKey: nodeKey,
    sources: bind.sources(bind.channels(states)),
    sinks: bind.sinks(commandChannels),
    stalenessTimeout,
  });
  const value = bind.values(states, readings);
  const missing = bind.missing(states, readings);
  const [anchor, setAnchor] = useState<HTMLButtonElement | null>(null);
  const [request, setRequest] = useState<Request>();
  const [promptVisible, setPromptVisible] = useState(false);
  Tooltip.useHidden(promptVisible);
  const send = ({ target, value: v }: Sendable): void => {
    if (target.type === "channel") set(target.channel.toString(), v);
    else
      onConfigChange({
        states: states.map((s) =>
          s.key === target.state ? { ...s, source: { type: "local", value: v } } : s,
        ),
      });
  };
  const onClick = resolveClick(
    resolved.spec,
    bound.spec,
    config,
    { value, missing, commanded: bind.commanded(states, readings) },
    {
      send,
      ask: (r) => {
        setRequest(r);
        setPromptVisible(true);
      },
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
      <Control.State
        config={config.control ?? DEFAULT_CONTROL}
        channels={commandChannels}
        onChange={onConfigChange}
      />
      <Indicator.Slot
        stale={bind.stale(states, readings)}
        unknown={missing.size > 0 || unmatched}
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
        buttonRef={setAnchor}
      />
      <Dialog.Frame
        visible={promptVisible}
        onVisibleChange={setPromptVisible}
        anchor={anchor}
      >
        {/* The prompt mounts for each open, so a typed value never outlives it. */}
        {promptVisible && request != null && (
          <Prompt
            request={request}
            onSend={(s) => {
              setPromptVisible(false);
              send(s);
            }}
          />
        )}
      </Dialog.Frame>
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
