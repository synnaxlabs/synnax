// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, schematic } from "@synnaxlabs/client";
import { Form as Base } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Tabs } from "@synnaxlabs/lyra/tabs";
import { Text } from "@synnaxlabs/lyra/text";
import { deep, notation } from "@synnaxlabs/x";
import type { ReactElement } from "react";

import { Channel } from "@/channel";
import { Notation } from "@/notation";
import { Form as NodeForm } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Orientation } from "@/schematic/node/common/orientation";
import type { FormProps } from "@/schematic/node/spec";
import { type Resolved, useResolved } from "@/schematic/node/symbol/resolve";
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
  onChange: (next: Binding) => void;
}

const SourceFields = ({ binding, onChange }: BindingProps): ReactElement => {
  const { source } = binding;
  const setSource = (next?: schematic.Source): void =>
    onChange({ ...binding, source: next });
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
        onChange={(n: notation.Notation) => onChange({ ...binding, notation: n })}
      />
    </Input.Item>
    <Input.Item label="Precision" padHelpText={false}>
      <Input.Numeric
        value={binding.precision ?? 2}
        onChange={(precision) => onChange({ ...binding, precision })}
        integer
        bounds={notation.PRECISION_BOUNDS}
      />
    </Input.Item>
    <Input.Item label="Units" padHelpText={false}>
      <Input.Text
        value={binding.units ?? ""}
        onChange={(units) =>
          onChange({ ...binding, units: units === "" ? undefined : units })
        }
      />
    </Input.Item>
  </>
);

interface StateSectionProps {
  state: schematic.symbol.State;
}

// A state with no binding shows the defaults, and its first edit adds the binding.
const StateSection = ({ state }: StateSectionProps): ReactElement => {
  const { value, onChange } = Base.useField(path.states);
  const binding =
    value.find((b) => b.key === state.key) ??
    schematic.stateBindingZ.parse({ key: state.key });
  const handleChange = (next: Binding): void => onChange(upsert(value, next));
  return (
    <Base.Section title={state.name}>
      <SourceFields binding={binding} onChange={handleChange} />
      {state.type === "number" && (
        <FormatFields binding={binding} onChange={handleChange} />
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
        <StateSection key={state.key} state={state} />
      ))}
      <Base.Section title="Staleness">
        <Staleness.TimeoutField />
      </Base.Section>
    </Base.Sections>
  );
};

const ControlForm = (): ReactElement => {
  const { spec, missing, failed } = useSymbol();
  const commands = spec?.commands.filter((c) => c.state == null) ?? [];
  return (
    <Base.Sections x>
      <WarningSection missing={missing} failed={failed} />
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
 * Edits a symbol node: the source and format of each state of its symbol, the channel
 * of each command with no state, its clicks, and its style.
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
