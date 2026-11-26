// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/symbol/Form.css";

import { type channel, type enumeration, query, schematic } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as Base } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Tabs } from "@synnaxlabs/lyra/tabs";
import { Text } from "@synnaxlabs/lyra/text";
import { deep, notation } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { Channel } from "@/channel";
import { CSS } from "@/css";
import { Enumeration } from "@/enumeration";
import { Notation } from "@/notation";
import { Form as NodeForm } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Orientation } from "@/schematic/node/common/orientation";
import type { FormProps } from "@/schematic/node/spec";
import { type Resolved, useResolved } from "@/schematic/node/symbol/resolve";
import { Synnax } from "@/synnax";
import { Properties } from "@/vis/properties";
import { Staleness } from "@/vis/staleness";

type Binding = schematic.StateBinding;

const path = deep.path<schematic.SymbolNodeConfig>();

const NEW_SOURCES: Record<schematic.SourceType, schematic.Source> = {
  channel: { type: "channel" },
  local: { type: "local", value: 0 },
};

const upsert = <T extends { key: string }>(entries: T[], next: T): T[] =>
  entries.some((e) => e.key === next.key)
    ? entries.map((e) => (e.key === next.key ? next : e))
    : [...entries, next];

type BindingChange = (patch: Partial<Binding>) => void;

// A state with no binding shows the defaults, and its first edit adds the binding.
const useBinding = (key: string): [Binding, BindingChange] => {
  const { get, set } = Base.useContext();
  const find = (states: Binding[]): Binding =>
    states.find((b) => b.key === key) ?? schematic.stateBindingZ.parse({ key });
  const binding = find(Base.useFieldValue(path.states));
  // A change reads the states as it applies, so one that lands after a create keeps
  // the edits made meanwhile.
  const handleChange: BindingChange = (patch) => {
    const states = get<Binding[]>(path.states).value;
    set(path.states, upsert(states, { ...find(states), ...patch }));
  };
  return [binding, handleChange];
};

interface BindingProps {
  binding: Binding;
  onChange: BindingChange;
}

const SOURCE_CLASS = CSS.BE("symbol-form", "source");

// No source draws as a channel source with no channel does, so the field shows both as
// Channel. The section names the field, and the channel and the value share one slot,
// so a switch moves nothing.
interface SourceFieldProps extends BindingProps {
  /** The enumeration of the state, whose entries a local value picks from. */
  enumeration?: enumeration.Key;
}

const SourceField = ({
  binding: { source },
  onChange,
  enumeration: key,
}: SourceFieldProps): ReactElement => {
  const type = source?.type ?? "channel";
  // A switch back restores the earlier source of that type, its command included.
  const [earlier, setEarlier] = useState<
    Partial<Record<schematic.SourceType, schematic.Source>>
  >({});
  const handleTypeChange = (next: schematic.SourceType): void => {
    if (next === type) return;
    if (source != null) setEarlier((prev) => ({ ...prev, [type]: source }));
    onChange({ source: earlier[next] ?? NEW_SOURCES[next] });
  };
  return (
    <Flex.Box y gap={2} align="start" className={SOURCE_CLASS}>
      <Select.Buttons value={type} onChange={handleTypeChange} aria-label="Source">
        <Select.Item itemKey="channel" tooltip="Reads a channel">
          Channel
        </Select.Item>
        <Select.Item itemKey="local" tooltip="Holds a value in the schematic">
          Local
        </Select.Item>
      </Select.Buttons>
      {source?.type === "local" && key != null && (
        <EntrySelect
          enumeration={key}
          value={source.value}
          onChange={(value) => onChange({ source: { ...source, value } })}
          aria-label="Value"
        />
      )}
      {source?.type === "local" && key == null && (
        <Input.Numeric
          aria-label="Value"
          value={source.value}
          onChange={(value) => onChange({ source: { ...source, value } })}
        />
      )}
      {source?.type !== "local" && (
        <Channel.SelectSingle
          allowNone
          value={source?.key}
          onChange={(k: channel.Key | null) =>
            onChange({
              source: {
                type: "channel",
                key: k ?? undefined,
                command: source?.command,
              },
            })
          }
        />
      )}
    </Flex.Box>
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

interface RangeFieldProps extends BindingProps {
  state: schematic.symbol.NumberState;
}

const RangeField = ({ state, binding, onChange }: RangeFieldProps): ReactElement => (
  <Input.Item label="Range" padHelpText={false}>
    <NodeForm.RangeInput
      label="Range"
      value={binding.bounds ?? state.bounds}
      onChange={(bounds) => onChange({ bounds })}
    />
  </Input.Item>
);

interface EntrySelectProps {
  enumeration: enumeration.Key;
  value: number;
  onChange: (next: number) => void;
  "aria-label"?: string;
}

// EntrySelect picks the value of an entry by its name.
const ENTRY_CLASS = CSS.BE("symbol-form", "entry");

const EntrySelect = ({
  enumeration: key,
  value,
  onChange,
  "aria-label": ariaLabel,
}: EntrySelectProps): ReactElement => {
  const { data } = Enumeration.useResult({ key });
  return (
    <Select.Simple<number>
      value={value}
      onChange={onChange}
      resourceName="entry"
      allowNone={false}
      triggerProps={{ "aria-label": ariaLabel, className: ENTRY_CLASS }}
    >
      {data?.entries.map((e) => (
        <Select.Item<number> key={e.value} itemKey={e.value} textValue={e.name}>
          <Enumeration.Colors entries={[e]} />
          {e.name}
        </Select.Item>
      ))}
    </Select.Simple>
  );
};

interface DefaultEntryProps extends Omit<EntrySelectProps, "aria-label"> {}

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
      <EntrySelect enumeration={key} value={value} onChange={onChange} />
    </Input.Item>
  );
};

interface EnumerationFieldsProps extends BindingProps {
  state: schematic.symbol.EnumerationState;
  /** A command writes the state, so the state reads exact values. */
  commanded: boolean;
}

const EnumerationFields = ({
  state,
  binding,
  onChange,
  commanded,
}: EnumerationFieldsProps): ReactElement => {
  const shown = binding.enumeration ?? state.enumeration;
  const value = binding.default ?? state.default;
  const match = binding.match ?? state.match;
  const setDefault = (next: number): void => onChange({ default: next });
  const client = Synnax.use();
  const { get } = Base.useContext();
  // A new enumeration starts as a copy of the one that the state shows.
  const draft = (): Enumeration.CreatePromptParams => {
    const cached =
      shown == null
        ? undefined
        : client?.enumerations.watch({ key: shown }).getCached();
    return {
      name: get<string>(path.label.label).value.trim() || state.name,
      entries: query.isLive(cached) ? cached.entries : Enumeration.createEntries(),
    };
  };
  return (
    <>
      <Input.Item label="Enumeration" padHelpText={false}>
        <NodeForm.EnumerationSelect
          value={binding.enumeration}
          inherited={state.enumeration}
          onChange={(next) => onChange({ enumeration: next })}
          draft={draft}
        />
      </Input.Item>
      {shown == null ? (
        <Input.Item label="Default" padHelpText={false}>
          <Input.Numeric value={value} onChange={setDefault} />
        </Input.Item>
      ) : (
        <DefaultEntry enumeration={shown} value={value} onChange={setDefault} />
      )}
      {!commanded && match.type === "within" && (
        <Input.Item label="Threshold" padHelpText={false}>
          <NodeForm.RangeInput
            label="Threshold"
            value={match.bounds}
            onChange={(bounds) => onChange({ match: { type: "within", bounds } })}
          />
        </Input.Item>
      )}
    </>
  );
};

const overridden = (b: Binding): boolean =>
  b.enumeration != null || b.match != null || b.bounds != null || b.default != null;

const RESET: Partial<Binding> = {
  enumeration: undefined,
  match: undefined,
  bounds: undefined,
  default: undefined,
};

interface StateSectionsProps {
  state: schematic.symbol.State;
  commanded: boolean;
  /** Whether the symbol has other states, so one section holds this whole state. */
  named: boolean;
}

/**
 * The sections of a state. The one state of a symbol splits by topic, under the names
 * of the topics. Each state of a symbol with more takes one section, under its name.
 */
const StateSections = ({
  state,
  commanded,
  named,
}: StateSectionsProps): ReactElement => {
  const [binding, handleChange] = useBinding(state.key);
  const reset = overridden(binding) && (
    <Button.Button
      variant="text"
      size="small"
      onClick={() => handleChange(RESET)}
      tooltip="Use the settings of the symbol"
    >
      <Icon.Reset />
      Reset
    </Button.Button>
  );
  const shown =
    state.type === "enumeration"
      ? (binding.enumeration ?? state.enumeration)
      : undefined;
  const source = (
    <SourceField binding={binding} onChange={handleChange} enumeration={shown} />
  );
  if (state.type === "number") {
    const format = <FormatFields binding={binding} onChange={handleChange} />;
    if (named)
      return (
        <Base.Section title={state.name} actions={reset}>
          <Base.Column>
            {source}
            <RangeField state={state} binding={binding} onChange={handleChange} />
          </Base.Column>
          <Base.Column>{format}</Base.Column>
        </Base.Section>
      );
    return (
      <>
        <Base.Section title="Source">{source}</Base.Section>
        <Base.Section title="Range" actions={reset}>
          <NodeForm.RangeInput
            label="Range"
            value={binding.bounds ?? state.bounds}
            onChange={(bounds) => handleChange({ bounds })}
          />
        </Base.Section>
        <Base.Section title="Format">{format}</Base.Section>
      </>
    );
  }
  const fields = (
    <EnumerationFields
      state={state}
      binding={binding}
      onChange={handleChange}
      commanded={commanded}
    />
  );
  if (named)
    return (
      <Base.Section title={state.name} actions={reset}>
        <Base.Column>{source}</Base.Column>
        <Base.Column>{fields}</Base.Column>
      </Base.Section>
    );
  return (
    <>
      <Base.Section title="Source">{source}</Base.Section>
      <Base.Section title="States" actions={reset}>
        {fields}
      </Base.Section>
    </>
  );
};

interface CommandFieldProps {
  label: string;
}

interface StateCommandFieldProps extends CommandFieldProps {
  state: schematic.symbol.State;
}

// A command of a state writes to the source of the state.
const StateCommandField = ({ state, label }: StateCommandFieldProps): ReactElement => {
  const [{ source }, onChange] = useBinding(state.key);
  return (
    <Input.Item label={label} padHelpText={false}>
      {source?.type === "local" ? (
        <Text.Text level="small" color={9} className={CSS.BE("symbol-form", "local")}>
          Commands set the local value
        </Text.Text>
      ) : (
        <Channel.SelectSingle
          allowNone
          value={source?.command}
          onChange={(command: channel.Key | null) =>
            onChange({
              source: {
                type: "channel",
                key: source?.key,
                command: command ?? undefined,
              },
            })
          }
        />
      )}
    </Input.Item>
  );
};

interface StatelessCommandFieldProps extends CommandFieldProps {
  command: schematic.symbol.Command;
}

const StatelessCommandField = ({
  command,
  label,
}: StatelessCommandFieldProps): ReactElement => {
  const { value, onChange } = Base.useField(path.commands);
  const handleChange = (key: channel.Key | null): void =>
    onChange(
      key == null
        ? value.filter((b) => b.key !== command.key)
        : upsert(value, { key: command.key, channel: key }),
    );
  return (
    <Input.Item label={label} padHelpText={false}>
      <Channel.SelectSingle
        allowNone
        value={value.find((b) => b.key === command.key)?.channel}
        onChange={handleChange}
      />
    </Input.Item>
  );
};

const useSymbol = (): Resolved => useResolved(Base.useFieldValue(path.spec));

// A symbol that has not resolved keeps every field, since the node stores them.
const hasStates = (spec?: schematic.symbol.Spec): boolean =>
  spec == null || spec.states.length > 0;

const clickable = (spec?: schematic.symbol.Spec): boolean =>
  spec == null || spec.interactions.length > 0;

const commanded = (spec?: schematic.symbol.Spec): boolean =>
  spec == null || spec.commands.length > 0;

const writes = (spec: schematic.symbol.Spec, state: schematic.symbol.State): boolean =>
  spec.commands.some((c) => c.state === state.key);

// WarningSection tells why a tab shows no states or commands.
const WarningSection = ({
  missing,
  failed,
}: Pick<Resolved, "missing" | "failed">): ReactElement | null =>
  !missing && !failed ? null : (
    <Base.Section title="Symbol">
      <Text.Text level="p" status="warning">
        <Icon.Warning />
        {missing
          ? "The symbol of this node was not found."
          : "The symbol of this node failed to load."}
      </Text.Text>
    </Base.Section>
  );

const TelemetryForm = (): ReactElement => {
  const { spec, missing, failed } = useSymbol();
  return (
    <Base.Sections x>
      <WarningSection missing={missing} failed={failed} />
      {spec?.states.map((state) => (
        <StateSections
          key={state.key}
          state={state}
          commanded={writes(spec, state)}
          named={spec.states.length > 1}
        />
      ))}
      <Base.Section title="Staleness">
        <Staleness.TimeoutField />
      </Base.Section>
    </Base.Sections>
  );
};

const ControlForm = (): ReactElement => {
  const { spec, missing, failed } = useSymbol();
  const states = spec?.states.filter((state) => writes(spec, state)) ?? [];
  const commands = spec?.commands.filter((c) => c.state == null) ?? [];
  // One target needs no name, since the section names it.
  const single = states.length + commands.length === 1;
  return (
    <Base.Sections x>
      <WarningSection missing={missing} failed={failed} />
      {commanded(spec) && (
        <Base.Section title={single ? "Command" : "Commands"}>
          {states.map((state) => (
            <StateCommandField
              key={state.key}
              state={state}
              label={single ? "Channel" : state.name}
            />
          ))}
          {commands.map((command) => (
            <StatelessCommandField
              key={command.key}
              command={command}
              label={single ? "Channel" : command.name}
            />
          ))}
          <NodeForm.ControlChipField />
        </Base.Section>
      )}
      {clickable(spec) && (
        <Base.Section title="Clicks">
          <NodeForm.NegatedSwitchField
            path={path.disabled}
            label="Clickable"
            padHelpText={false}
          />
          <NodeForm.ActivationDelayField />
        </Base.Section>
      )}
    </Base.Sections>
  );
};

const StyleForm = (): ReactElement => (
  <Base.Sections x>
    <Base.Section title="Label">
      <Label.Form path={path.label} />
      <Orientation.Field path={path} hideInner label="Position" />
    </Base.Section>
    <Base.Section title="Appearance">
      <NodeForm.ColorField path={path.color} label="Color" />
      <NodeForm.ScaleField path={path.scale} />
      <NodeForm.NegatedSwitchField
        path={path.motionDisabled}
        label="Motion"
        padHelpText={false}
      />
      <Orientation.Field path={path} hideOuter label="Orientation" />
    </Base.Section>
  </Base.Sections>
);

/**
 * Edits a symbol node: the source and the display of each state of its symbol, the
 * channel of each command, its clicks, and its style. The Telemetry tab shows only for
 * a symbol with a state, and the Control tab only for one with a click or a command.
 */
export const Form = ({ tab, onTabChange }: FormProps): ReactElement => {
  const { spec } = useSymbol();
  const telemetry = hasStates(spec);
  const control = clickable(spec) || commanded(spec);
  const tabs: Properties.TabKey[] = [];
  if (telemetry) tabs.push("telemetry");
  if (control) tabs.push("control");
  tabs.push("style");
  return (
    <Properties.Tabs tabs={tabs} tab={tab} onTabChange={onTabChange}>
      {telemetry && (
        <Tabs.Content itemKey="telemetry">
          <TelemetryForm />
        </Tabs.Content>
      )}
      {control && (
        <Tabs.Content itemKey="control">
          <ControlForm />
        </Tabs.Content>
      )}
      <Tabs.Content itemKey="style">
        <StyleForm />
      </Tabs.Content>
    </Properties.Tabs>
  );
};
