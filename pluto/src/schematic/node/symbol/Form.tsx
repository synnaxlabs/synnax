// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type enumeration, query, schematic } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Form as Base } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Tabs } from "@synnaxlabs/lyra/tabs";
import { Text } from "@synnaxlabs/lyra/text";
import { bounds, deep, notation } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { Channel } from "@/channel";
import { CSS } from "@/css";
import { Enumeration } from "@/enumeration";
import { Notation } from "@/notation";
import { Form as NodeForm } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Orientation } from "@/schematic/node/common/orientation";
import { Telem } from "@/schematic/node/common/telem";
import type { FormProps } from "@/schematic/node/spec";
import { type Resolved, useResolved } from "@/schematic/node/symbol/resolve";
import { Synnax } from "@/synnax";
import { Properties } from "@/vis/properties";
import { Staleness } from "@/vis/staleness";

type Binding = schematic.StateBinding;

const path = deep.path<schematic.SymbolNodeConfig>();

const NONE = "none";

type SourceType = schematic.SourceType | typeof NONE;

const NEW_SOURCES: Record<schematic.SourceType, schematic.Source> = {
  channel: { type: "channel" },
  local: { type: "local", value: 0 },
};

const upsert = <T extends { key: string }>(entries: T[], next: T): T[] =>
  entries.some((e) => e.key === next.key)
    ? entries.map((e) => (e.key === next.key ? next : e))
    : [...entries, next];

interface BindingProps {
  binding: Binding;
  onChange: (patch: Partial<Binding>) => void;
}

const SourceFields = ({ binding, onChange }: BindingProps): ReactElement => {
  const { source } = binding;
  const setSource = (next?: schematic.Source): void => onChange({ source: next });
  return (
    <>
      <Input.Item label="Source" padHelpText={false}>
        <Select.Buttons
          value={source?.type ?? NONE}
          onChange={(t: SourceType) =>
            setSource(t === NONE ? undefined : NEW_SOURCES[t])
          }
        >
          <Select.Item itemKey={NONE}>None</Select.Item>
          <Select.Item itemKey="channel">Channel</Select.Item>
          <Select.Item itemKey="local">Local</Select.Item>
        </Select.Buttons>
      </Input.Item>
      {source?.type === "channel" && (
        <>
          <Input.Item label="Channel" padHelpText={false}>
            <Channel.SelectSingle
              allowNone
              value={source.key}
              onChange={(key: channel.Key | null) =>
                setSource({ ...source, key: key ?? undefined })
              }
            />
          </Input.Item>
          <Input.Item label="Command channel" padHelpText={false}>
            <Channel.SelectSingle
              allowNone
              value={source.command}
              onChange={(key: channel.Key | null) =>
                setSource({ ...source, command: key ?? undefined })
              }
            />
          </Input.Item>
        </>
      )}
      {source?.type === "local" && (
        <Input.Item label="Value" padHelpText={false}>
          <Input.Numeric
            value={source.value}
            onChange={(value) => setSource({ ...source, value })}
          />
        </Input.Item>
      )}
    </>
  );
};

const FormatFields = ({ binding, onChange }: BindingProps): ReactElement => (
  <>
    <Input.Item label="Notation" padHelpText={false}>
      <Notation.Select
        value={binding.notation ?? "standard"}
        onChange={(n: notation.Notation) => onChange({ notation: n })}
      />
    </Input.Item>
    <Input.Item label="Precision" padHelpText={false}>
      <Input.Numeric
        value={binding.precision ?? 2}
        onChange={(precision) => onChange({ precision })}
        integer
        bounds={notation.PRECISION_BOUNDS}
      />
    </Input.Item>
    <Input.Item label="Units" padHelpText={false}>
      <Input.Text
        value={binding.units ?? ""}
        onChange={(units) => onChange({ units: units === "" ? undefined : units })}
      />
    </Input.Item>
  </>
);

interface BoundsInputsProps {
  value: bounds.Bounds;
  onChange: (next: bounds.Bounds) => void;
}

interface Held {
  typed: bounds.Bounds;
  /** The stored bounds when the hold started. */
  stored: bounds.Bounds;
}

const INVALID_BOUNDS = "Minimum value must be less than maximum value";

/**
 * Edits a pair of bounds. The Core refuses bounds whose lower is not below its upper,
 * so the fields hold such a pair with an error and save nothing until it is valid. A
 * change to the stored bounds drops the held pair.
 */
const BoundsInputs = ({ value, onChange }: BoundsInputsProps): ReactElement => {
  const [held, setHeld] = useState<Held>();
  const stale = held != null && !bounds.equals(held.stored, value);
  if (stale) setHeld(undefined);
  const holding = held != null && !stale;
  const shown = holding ? held.typed : value;
  const handleChange = (key: keyof bounds.Bounds) => (v: number) => {
    const next = { ...shown, [key]: v };
    if (next.lower >= next.upper) return setHeld({ typed: next, stored: value });
    setHeld(undefined);
    onChange(next);
  };
  return (
    <>
      <Input.Item
        label="Min value"
        padHelpText={false}
        status={holding ? "error" : undefined}
        helpText={holding ? INVALID_BOUNDS : undefined}
      >
        <Input.Numeric value={shown.lower} onChange={handleChange("lower")} />
      </Input.Item>
      <Input.Item label="Max value" padHelpText={false}>
        <Input.Numeric value={shown.upper} onChange={handleChange("upper")} />
      </Input.Item>
    </>
  );
};

interface DefaultEntryProps {
  enumeration: enumeration.Key;
  value: number;
  onChange: (next: number) => void;
}

const DefaultEntry = ({
  enumeration: key,
  value,
  onChange,
}: DefaultEntryProps): ReactElement => {
  const { data } = Enumeration.useResult({ key });
  const unmatched = data != null && !data.entries.some((e) => e.value === value);
  return (
    <Input.Item
      label="Default"
      padHelpText={false}
      status={unmatched ? "warning" : undefined}
      helpText={unmatched ? `No entry has the value ${value}.` : undefined}
    >
      <Select.Simple<number> value={value} onChange={onChange} resourceName="entry">
        {data?.entries.map((e) => (
          <Select.Item<number> key={e.value} itemKey={e.value}>
            {e.name}
          </Select.Item>
        ))}
      </Select.Simple>
    </Input.Item>
  );
};

interface EnumerationOverridesProps extends BindingProps {
  state: schematic.symbol.EnumerationState;
}

const EnumerationOverrides = ({
  state,
  binding,
  onChange,
}: EnumerationOverridesProps): ReactElement => {
  const match = binding.match ?? state.match;
  const within =
    state.match.type === "within" ? state.match.bounds : Telem.DEFAULT_THRESHOLD;
  const setMatch = (next: schematic.symbol.Matcher): void => onChange({ match: next });
  const enumerationKey = binding.enumeration ?? state.enumeration;
  const value = binding.default ?? state.default;
  const setDefault = (next: number): void => onChange({ default: next });
  const client = Synnax.use();
  // A new enumeration starts with the entries that the state draws.
  const draft = (): Enumeration.CreatePromptParams => {
    const drawn =
      state.enumeration == null
        ? undefined
        : client?.enumerations.watch({ key: state.enumeration }).getCached();
    return { name: state.name, entries: query.isLive(drawn) ? drawn.entries : [] };
  };
  return (
    <>
      <Input.Item label="Match" padHelpText={false}>
        <Select.Buttons
          value={match.type}
          onChange={(type: schematic.symbol.MatcherType) =>
            setMatch(type === "exact" ? { type } : { type, bounds: within })
          }
        >
          <Select.Item itemKey="exact">Exact</Select.Item>
          <Select.Item itemKey="within">Within</Select.Item>
        </Select.Buttons>
      </Input.Item>
      {match.type === "within" && (
        <BoundsInputs
          value={match.bounds}
          onChange={(b) => setMatch({ ...match, bounds: b })}
        />
      )}
      <NodeForm.EnumerationField
        value={binding.enumeration}
        onChange={(next) => onChange({ enumeration: next })}
        draft={draft}
        fallback={
          state.enumeration == null ? (
            <Text.Text level="p">The symbol names no enumeration.</Text.Text>
          ) : (
            <Enumeration.Entries enumeration={state.enumeration} />
          )
        }
      />
      {enumerationKey == null ? (
        <Input.Item label="Default" padHelpText={false}>
          <Input.Numeric value={value} onChange={setDefault} />
        </Input.Item>
      ) : (
        <DefaultEntry
          enumeration={enumerationKey}
          value={value}
          onChange={setDefault}
        />
      )}
    </>
  );
};

const overridden = (b: Binding): boolean =>
  b.enumeration != null || b.match != null || b.bounds != null || b.default != null;

interface StateSectionProps {
  state: schematic.symbol.State;
}

// A state with no binding shows the defaults, and its first edit adds the binding.
const StateSection = ({ state }: StateSectionProps): ReactElement => {
  const { get, set } = Base.useContext();
  const find = (states: Binding[]): Binding =>
    states.find((b) => b.key === state.key) ??
    schematic.stateBindingZ.parse({ key: state.key });
  const binding = find(Base.useFieldValue(path.states));
  // A change reads the states when it applies, so the link of an enumeration that
  // resolves late keeps the edits made while it ran.
  const handleChange = (patch: Partial<Binding>): void => {
    const states = get<Binding[]>(path.states).value;
    set(path.states, upsert(states, { ...find(states), ...patch }));
  };
  return (
    <Base.Section
      title={state.name}
      className={state.type === "enumeration" ? CSS.B("states") : undefined}
    >
      <SourceFields binding={binding} onChange={handleChange} />
      {state.type === "number" ? (
        <>
          <FormatFields binding={binding} onChange={handleChange} />
          <BoundsInputs
            value={binding.bounds ?? state.bounds}
            onChange={(b) => handleChange({ bounds: b })}
          />
        </>
      ) : (
        <EnumerationOverrides state={state} binding={binding} onChange={handleChange} />
      )}
      {overridden(binding) && (
        <Button.Button
          variant="text"
          onClick={() =>
            handleChange({
              enumeration: undefined,
              match: undefined,
              bounds: undefined,
              default: undefined,
            })
          }
        >
          <Icon.Reset />
          Reset to symbol
        </Button.Button>
      )}
    </Base.Section>
  );
};

interface CommandFieldProps {
  command: schematic.symbol.Command;
}

const CommandField = ({ command }: CommandFieldProps): ReactElement => {
  const { value, onChange } = Base.useField(path.commands);
  const handleChange = (key: channel.Key | null): void =>
    onChange(
      key == null
        ? value.filter((b) => b.key !== command.key)
        : upsert(value, { key: command.key, channel: key }),
    );
  return (
    <Input.Item label={command.name} padHelpText={false}>
      <Channel.SelectSingle
        allowNone
        value={value.find((b) => b.key === command.key)?.channel}
        onChange={handleChange}
      />
    </Input.Item>
  );
};

const useSymbol = (): Resolved => useResolved(Base.useFieldValue(path.spec));

const MissingSection = (): ReactElement => (
  <Base.Section title="Symbol">
    <Text.Text level="p" status="warning">
      <Icon.Warning />
      The symbol of this node was not found.
    </Text.Text>
  </Base.Section>
);

const TelemetryForm = (): ReactElement => {
  const { spec, missing } = useSymbol();
  return (
    <Base.Sections x>
      {missing && <MissingSection />}
      {spec?.states.map((state) => (
        <StateSection key={state.key} state={state} />
      ))}
      <Base.Section title="Staleness">
        <Staleness.TimeoutField />
      </Base.Section>
    </Base.Sections>
  );
};

const ControlForm = (): ReactElement => {
  const { spec, missing } = useSymbol();
  const commands = spec?.commands.filter((c) => c.state == null) ?? [];
  return (
    <Base.Sections x>
      {missing && <MissingSection />}
      {commands.length > 0 && (
        <Base.Section title="Commands">
          {commands.map((command) => (
            <CommandField key={command.key} command={command} />
          ))}
        </Base.Section>
      )}
      <Base.Section title="Clicks">
        <NodeForm.NegatedSwitchField
          path={path.disabled}
          label="Clickable"
          padHelpText={false}
        />
        <NodeForm.ActivationDelayField />
        <NodeForm.ControlChipField />
      </Base.Section>
    </Base.Sections>
  );
};

const StyleForm = (): ReactElement => (
  <Base.Sections x>
    <Base.Section title="Label">
      <Label.Form path={path.label} />
    </Base.Section>
    <Base.Section title="Appearance">
      <NodeForm.ColorField path={path.color} label="Color" />
      <NodeForm.ScaleField path={path.scale} />
      <NodeForm.NegatedSwitchField
        path={path.motionDisabled}
        label="Motion"
        padHelpText={false}
      />
    </Base.Section>
    <Orientation.Section path={path} />
  </Base.Sections>
);

/**
 * Edits a symbol node: the source, format, and overrides of each state of its symbol,
 * the channel of each command with no state, its clicks, and its style.
 */
export const Form = ({ actions, tab, onTabChange }: FormProps): ReactElement => (
  <Properties.Tabs
    tabs={["telemetry", "control", "style"]}
    tab={tab}
    onTabChange={onTabChange}
    actions={actions}
  >
    <Tabs.Content itemKey="telemetry">
      <TelemetryForm />
    </Tabs.Content>
    <Tabs.Content itemKey="control">
      <ControlForm />
    </Tabs.Content>
    <Tabs.Content itemKey="style">
      <StyleForm />
    </Tabs.Content>
  </Properties.Tabs>
);
