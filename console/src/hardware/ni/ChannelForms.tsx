import { Form, Channel, List, Select, Input } from "@synnaxlabs/pluto";
import {
  AIChan,
  AIChanType,
  AccelSensitivityUnits,
  ExcitationSource,
  TerminalConfig,
} from "@/hardware/ni/types";
import { FC, ReactElement } from "react";

interface FormProps {
  prefix: string;
  fieldKey?: string;
  label?: string;
}

const ChannelField = ({
  prefix,
  fieldKey = "channel",
  label = "Synnax Channel",
}: FormProps): ReactElement => (
  <Form.Field path={`${prefix}.${fieldKey}`} label={label}>
    {(p) => <Channel.SelectSingle {...p} />}
  </Form.Field>
);

interface NamedKey<K extends string = string> {
  key: K;
  name: string;
}

const NAMED_KEY_COLS: List.ColumnSpec<string, NamedKey>[] = [
  {
    key: "name",
    name: "Name",
  },
];

const buildNamedKeySelect =
  <K extends string>(
    label_: string,
    fieldKey_: string,
    data: NamedKey<K>[],
  ): FC<FormProps> =>
  ({ prefix, fieldKey, label }): ReactElement => (
    <Form.Field<K> path={`${prefix}.${fieldKey}`} label={label}>
      {(p) => (
        <Select.Single<K, NamedKey<K>>
          {...p}
          columns={NAMED_KEY_COLS}
          data={data}
          entryRenderKey="name"
        />
      )}
    </Form.Field>
  );

const TerminalConfigField = buildNamedKeySelect(
  "Terminal Configuration",
  "terminalConfig",
  [
    {
      key: "RSE",
      name: "Referenced Single Ended",
    },
    {
      key: "NRSE",
      name: "Non-Referenced Single Ended",
    },
    {
      key: "PseudoDiff",
      name: "Pseudo-Differential",
    },
    {
      key: "Cfg_Default",
      name: "Default",
    },
  ],
);

const AccelSensitivityUnitsField = buildNamedKeySelect<AccelSensitivityUnits>(
  "Sensitivity Units",
  "sensitivityUnits",
  [
    {
      key: "mVoltsPerG",
      name: "mV/g",
    },
    {
      key: "VoltsPerG",
      name: "mV/(m/s^2)",
    },
  ],
);

const ExcitSourceField = buildNamedKeySelect<ExcitationSource>(
  "Excitation Source",
  "excitSource",
  [
    {
      key: "Internal",
      name: "Internal",
    },
    {
      key: "External",
      name: "External",
    },
  ],
);

const buildNumericField =
  (label: string, fieldKey_: string): FC<FormProps> =>
  ({ prefix: path, fieldKey = fieldKey_ }): ReactElement => (
    <Form.Field<number> path={`${path}.${fieldKey}`} label={label}>
      {(p) => <Input.Numeric {...p} />}
    </Form.Field>
  );

const MinValueField = buildNumericField("Minimum Value", "minValue");
const MaxValueField = buildNumericField("Maximum Value", "maxValue");
const SensitivityField = buildNumericField("Sensitivity", "sensitivity");
const CurrentExcitValField = buildNumericField(
  "Current Excitation Value",
  "currExcitVal",
);

const ANALOG_INPUT_FORMS: Record<AIChanType, FC<FormProps>> = {
  ai_accel: ({ prefix: path }) => {
    return (
      <>
        <ChannelField prefix={path} />
        <TerminalConfigField prefix={path} />
        <MinValueField prefix={path} />
        <MaxValueField prefix={path} />
        <SensitivityField prefix={path} />
        <AccelSensitivityUnitsField prefix={path} />
        <ExcitSourceField prefix={path} fieldKey="currExcitSource" />
        <CurrentExcitValField prefix={path} />
      </>
    );
  },
};
