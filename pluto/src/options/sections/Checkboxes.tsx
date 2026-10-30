import { Flex } from "@synnaxlabs/lyra/flex";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";

import { noop, Option, Row, Scrim, Section } from "@/options/common";
import { Field } from "@/options/samples";
import { ChannelCreateModal } from "@/options/sections/Modal";

/**
 * switch: today. under: the label above, a checkbox on the input line. inline: box,
 * then label, on the input line. dot: the per-channel enable control.
 */
type Style = "switch" | "under" | "inline" | "dot";

interface Setting {
  label: string;
  value: boolean;
}

const TASK_SETTINGS: Setting[] = [
  { label: "Data saving", value: true },
  { label: "Auto start", value: false },
  { label: "NaN fill", value: false },
];

const CHANNEL_TODAY: Setting[] = [
  { label: "Virtual", value: false },
  { label: "Is index", value: false },
];

const CHANNEL_SETTINGS: Setting[] = [
  { label: "Virtual", value: false },
  { label: "Index channel", value: true },
];

/** EnabledCheckbox from the Console's task channel lists, as it renders there. */
const Dot = ({ value }: { value: boolean }) => (
  <Input.Checkbox
    value={value}
    onChange={noop}
    size="small"
    checkedIcon={<Status.Indicator variant="success" />}
    uncheckedIcon={<Status.Indicator variant="disabled" />}
  />
);

const Setting = ({ style, setting }: { style: Style; setting: Setting }) => {
  switch (style) {
    case "switch":
      return (
        <Field label={setting.label}>
          <Input.Switch value={setting.value} onChange={noop} />
        </Field>
      );
    case "under":
      return (
        <Field label={setting.label}>
          <Flex.Box x align="center" className="opt-checks-under">
            <Input.Checkbox value={setting.value} onChange={noop} />
          </Flex.Box>
        </Field>
      );
    case "inline":
      return (
        <Flex.Box x align="center" className="opt-check opt-check-row">
          <Input.Checkbox value={setting.value} onChange={noop} />
          <Text.Text>{setting.label}</Text.Text>
        </Flex.Box>
      );
    case "dot":
      return (
        <Flex.Box x align="center" className="opt-dot-check opt-check-row">
          <Dot value={setting.value} />
          <Text.Text>{setting.label}</Text.Text>
        </Flex.Box>
      );
  }
};

/** Settings that sit on the input line, below a spacer that matches a field label. */
const Settings = ({ style, settings }: { style: Style; settings: Setting[] }) => {
  const items = settings.map((s) => <Setting key={s.label} style={style} setting={s} />);
  if (style === "switch" || style === "under")
    return (
      <Flex.Box x gap="large" align="start">
        {items}
      </Flex.Box>
    );
  return (
    <Flex.Box y empty>
      <div className="opt-label-spacer" />
      <Flex.Box x gap="huge" align="center" className="opt-checks-inline">
        {items}
      </Flex.Box>
    </Flex.Box>
  );
};

const DeviceSelect = () => (
  <Select.Simple<string>
    resourceName="device"
    value={undefined as unknown as string}
    onChange={noop}
    allowNone
    full="x"
  >
    <Select.Item itemKey="alpha">Test stand alpha HTTP</Select.Item>
  </Select.Simple>
);

const TaskStrip = ({ style }: { style: Style }) => (
  <Flex.Box y gap="medium" className="opt-task">
    <Text.Text level="h2" className="opt-task-title">
      HTTP read task
    </Text.Text>
    <Flex.Box x gap="large" align="start">
      <Field label="HTTP server" grow>
        <DeviceSelect />
      </Field>
      <Field label="Rate">
        <div style={{ width: 150 }}>
          <Input.Numeric value={1} onChange={noop} endContent="Hz" />
        </div>
      </Field>
      <Settings style={style} settings={TASK_SETTINGS} />
    </Flex.Box>
  </Flex.Box>
);

const CheckboxSample = ({ style }: { style: Style }) => (
  <Flex.Box y gap="large">
    <Row label="Task form settings (save now, act on deploy)">
      <TaskStrip style={style} />
    </Row>
    <Row label="Channel create modal (applies on submit)">
      <Scrim>
        <ChannelCreateModal
          toggles={
            <Settings
              style={style}
              settings={style === "switch" ? CHANNEL_TODAY : CHANNEL_SETTINGS}
            />
          }
        />
      </Scrim>
    </Row>
  </Flex.Box>
);

export const CheckboxSection = () => (
  <Section
    id="checkboxes"
    n={10}
    title="Task form checkboxes"
    summary="Turns settings that act only on submit or deploy (Data saving, Auto start, Virtual, Index channel) from switches into checkboxes. Create more stays a switch: it acts on the dialog at once."
  >
    <Option
      letter="A"
      title="Today"
      desc="Switches with the label above, in the task form and the channel create modal. The modal asks Is index."
    >
      <CheckboxSample style="switch" />
    </Option>
    <Option
      letter="B"
      title="Checkbox under its label"
      className="cb-b"
      desc="A drop-in swap: the form grid stays, and a 16 px checkbox replaces each switch on the input line."
    >
      <CheckboxSample style="under" />
    </Option>
    <Option
      letter="C"
      title="Box, then label, on the input line"
      className="cb-c"
      desc="A 16 px checkbox with its label at its right, in one row on the input line. The label is part of the hit area."
    >
      <CheckboxSample style="inline" />
    </Option>
    <Option
      letter="D"
      title="Status dot"
      className="cb-d"
      desc="The per-channel enable control from task channel lists: a green dot when on, a gray dot when off, label at its right."
    >
      <CheckboxSample style="dot" />
    </Option>
  </Section>
);
