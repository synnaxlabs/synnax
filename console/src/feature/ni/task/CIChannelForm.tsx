// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { Select } from "@synnaxlabs/lyra/select";
import { record } from "@synnaxlabs/x";
import { type FC } from "react";

import { Select as SelectDevice } from "@/feature/ni/device/Select";
import { CustomScaleSection } from "@/feature/ni/task/CustomScaleForm";
import { MinMaxValueFields } from "@/feature/ni/task/MinMaxValueFields";
import { PortField } from "@/feature/ni/task/PortField";
import { COUNTER_INPUT } from "@/feature/ni/task/ports";
import {
  channelTypeItems,
  SelectCIChannelTypeField,
} from "@/feature/ni/task/SelectCIChannelTypeField";
import {
  CI_CHANNEL_TYPE_NAMES,
  type CIAngularPositionUnits,
  type CIAngularVelocityUnits,
  type CIChannel,
  type CIChannelType,
  type CICountDirection,
  type CIDecodingType,
  type CIEdge,
  type CIFreqUnits,
  type CILinearPositionUnits,
  type CILinearVelocityUnits,
  type CIMeasMethod,
  type CIPeriodUnits,
  type CIPulseWidthUnits,
  type CISemiPeriodUnits,
  type CITimeUnits,
  type CITwoEdgeSepUnits,
} from "@/feature/ni/task/types";
import { allowedTypes, useChannelDevice } from "@/feature/ni/task/usePort";
import { Task } from "@/platform/task";

const CI_TIME_UNITS_NAMES = {
  Seconds: "Seconds",
  Ticks: "Ticks",
} as const satisfies Record<CITimeUnits, string>;

const CI_EDGE_NAMES = {
  Rising: "Rising",
  Falling: "Falling",
} as const satisfies Record<CIEdge, string>;

interface FormProps {
  prefix: string;
}

const CI_FREQ_UNITS_NAMES = {
  Hz: "Hz",
  Ticks: "Ticks",
} as const satisfies Record<CIFreqUnits, string>;

const UnitsField = Form.buildSelectField<CIFreqUnits>({
  fieldKey: "units",
  fieldProps: { label: "Units" },
  inputProps: {
    resourceName: "units",
    children: Task.selectItems(CI_FREQ_UNITS_NAMES),
  },
});

const PeriodUnitsField = Form.buildSelectField<CIPeriodUnits>({
  fieldKey: "units",
  fieldProps: { label: "Units" },
  inputProps: {
    resourceName: "units",
    children: Task.selectItems(CI_TIME_UNITS_NAMES),
  },
});

const PulseWidthUnitsField = Form.buildSelectField<CIPulseWidthUnits>({
  fieldKey: "units",
  fieldProps: { label: "Scaled units" },
  inputProps: {
    resourceName: "scaled units",
    children: Task.selectItems(CI_TIME_UNITS_NAMES),
  },
});

const SemiPeriodUnitsField = Form.buildSelectField<CISemiPeriodUnits>({
  fieldKey: "units",
  fieldProps: { label: "Scaled units" },
  inputProps: {
    resourceName: "scaled units",
    children: Task.selectItems(CI_TIME_UNITS_NAMES),
  },
});

const TwoEdgeSepUnitsField = Form.buildSelectField<CITwoEdgeSepUnits>({
  fieldKey: "units",
  fieldProps: { label: "Scaled units" },
  inputProps: {
    resourceName: "scaled units",
    children: Task.selectItems(CI_TIME_UNITS_NAMES),
  },
});

const EdgeField = Form.buildSelectField<CIEdge>({
  fieldKey: "edge",
  fieldProps: { label: "Starting edge" },
  inputProps: {
    resourceName: "starting edge",
    searchName: "edge",
    children: Task.selectItems(CI_EDGE_NAMES),
  },
});

const StartingEdgeField = Form.buildSelectField<CIEdge>({
  fieldKey: "startingEdge",
  fieldProps: { label: "Starting edge" },
  inputProps: {
    resourceName: "starting edge",
    searchName: "edge",
    children: Task.selectItems(CI_EDGE_NAMES),
  },
});

const ActiveEdgeField = Form.buildSelectField<CIEdge>({
  fieldKey: "activeEdge",
  fieldProps: { label: "Active edge" },
  inputProps: {
    resourceName: "active edge",
    children: Task.selectItems(CI_EDGE_NAMES),
  },
});

const CI_COUNT_DIRECTION_NAMES = {
  CountUp: "Count up",
  CountDown: "Count down",
  ExternallyControlled: "Externally controlled",
} as const satisfies Record<CICountDirection, string>;

const CountDirectionField = Form.buildSelectField<CICountDirection>({
  fieldKey: "countDirection",
  fieldProps: { label: "Count direction" },
  inputProps: {
    resourceName: "count direction",
    searchName: "direction",
    children: Task.selectItems(CI_COUNT_DIRECTION_NAMES),
  },
});

const InitialCountField = Form.buildNumericField({
  fieldKey: "initialCount",
  fieldProps: { label: "Initial count" },
  inputProps: {},
});

const CI_MEAS_METHOD_NAMES = {
  LowFreq1Ctr: "One counter (low frequency)",
  HighFreq2Ctr: "Two counters (high frequency)",
  LargeRng2Ctr: "Two counters (large range)",
  DynamicAvg: "Dynamic averaging",
} as const satisfies Record<CIMeasMethod, string>;

const MeasMethodField = Form.buildSelectField<CIMeasMethod>({
  fieldKey: "measMethod",
  fieldProps: { label: "Measurement method" },
  inputProps: {
    resourceName: "measurement method",
    searchName: "method",
    children: Task.selectItems(CI_MEAS_METHOD_NAMES),
  },
});

const MeasTimeField = Form.buildNumericField({
  fieldKey: "measTime",
  fieldProps: { label: "Measurement time (s)" },
  inputProps: {},
});

const DivisorField = Form.buildNumericField({
  fieldKey: "divisor",
  fieldProps: { label: "Divisor" },
  inputProps: {},
});

// A device with no port list offers PFI0 to PFI15.
const FALLBACK_TERMINALS = Array.from({ length: 16 }, (_, i) => `PFI${i}`);

// The PFI terminals of the channel's device, and the current terminal.
const useTerminalItems = (path: string, current: string) => {
  const hardware = useChannelDevice(path)?.properties.hardware;
  const terminals =
    hardware == null
      ? [...FALLBACK_TERMINALS]
      : hardware.terminals.filter((t) => t.startsWith("PFI"));
  if (current !== "" && !terminals.includes(current)) terminals.push(current);
  return terminals.map((t) => (
    <Select.Item key={t} itemKey={t}>
      {t}
    </Select.Item>
  ));
};

interface TerminalFieldProps {
  path: string;
  fieldKey: "terminal" | "terminalA" | "terminalB" | "terminalZ";
  label: string;
  disabled?: boolean;
}

const TerminalField = ({ path, fieldKey, label, disabled }: TerminalFieldProps) => {
  const fieldPath = `${path}.${fieldKey}`;
  const value = Form.useFieldValue<string>(fieldPath);
  const items = useTerminalItems(path, value);
  return (
    <Form.Field<string> path={fieldPath} label={label}>
      {({ value, onChange, preview }) => (
        <Select.Simple<string>
          value={value}
          onChange={(v: string | null) => onChange(v ?? "")}
          preview={preview}
          allowNone
          disabled={disabled}
          resourceName={label.charAt(0).toLowerCase() + label.slice(1)}
          searchName="terminal"
        >
          {items}
        </Select.Simple>
      )}
    </Form.Field>
  );
};

const InputTerminalField = ({ path }: { path: string }) => (
  <TerminalField path={path} fieldKey="terminal" label="Input terminal" />
);

const TerminalAField = ({ path }: { path: string }) => (
  <TerminalField path={path} fieldKey="terminalA" label="Input terminal A" />
);

const TerminalBField = ({ path }: { path: string }) => (
  <TerminalField path={path} fieldKey="terminalB" label="Input terminal B" />
);

const FirstEdgeField = Form.buildSelectField<CIEdge>({
  fieldKey: "firstEdge",
  fieldProps: { label: "Edge 1" },
  inputProps: {
    resourceName: "edge 1",
    children: Task.selectItems(CI_EDGE_NAMES),
  },
});

const SecondEdgeField = Form.buildSelectField<CIEdge>({
  fieldKey: "secondEdge",
  fieldProps: { label: "Edge 2" },
  inputProps: {
    resourceName: "edge 2",
    children: Task.selectItems(CI_EDGE_NAMES),
  },
});

const CI_DECODING_TYPE_NAMES = {
  X1: "X1",
  X2: "X2",
  X4: "X4",
  TwoPulse: "Two pulse",
} as const satisfies Record<CIDecodingType, string>;

const DecodingTypeField = Form.buildSelectField<CIDecodingType>({
  fieldKey: "decodingType",
  fieldProps: { label: "Decoding type" },
  inputProps: {
    resourceName: "decoding type",
    searchName: "type",
    children: Task.selectItems(CI_DECODING_TYPE_NAMES),
  },
});

const CI_LINEAR_VELOCITY_UNITS_NAMES = {
  "m/s": "m/s",
  "in/s": "in/s",
} as const satisfies Record<CILinearVelocityUnits, string>;

const LinearVelocityUnitsField = Form.buildSelectField<CILinearVelocityUnits>({
  fieldKey: "units",
  fieldProps: { label: "Scaled units" },
  inputProps: {
    resourceName: "scaled units",
    children: Task.selectItems(CI_LINEAR_VELOCITY_UNITS_NAMES),
  },
});

const CI_ANGULAR_VELOCITY_UNITS_NAMES = {
  RPM: "RPM",
  "Radians/s": "Radians/s",
  "Degrees/s": "Degrees/s",
} as const satisfies Record<CIAngularVelocityUnits, string>;

const AngularVelocityUnitsField = Form.buildSelectField<CIAngularVelocityUnits>({
  fieldKey: "units",
  fieldProps: { label: "Scaled units" },
  inputProps: {
    resourceName: "scaled units",
    children: Task.selectItems(CI_ANGULAR_VELOCITY_UNITS_NAMES),
  },
});

const DistPerPulseField = Form.buildNumericField({
  fieldKey: "distPerPulse",
  fieldProps: { label: "Distance / pulse" },
  inputProps: {},
});

const PulsesPerRevField = Form.buildNumericField({
  fieldKey: "pulsesPerRev",
  fieldProps: { label: "Pulses / rev" },
  inputProps: {},
});

const InitialPosField = Form.buildNumericField({
  fieldKey: "initialPos",
  fieldProps: { label: "Initial position" },
  inputProps: {},
});

const InitialAngleField = Form.buildNumericField({
  fieldKey: "initialAngle",
  fieldProps: { label: "Initial angle" },
  inputProps: {},
});

const ZIndexEnabledField: FC<{ path: string }> = ({ path }) => (
  <Form.CheckboxField path={`${path}.zIndexEnabled`} label="Z index enable" />
);

const ZIndexValField: FC<{ path: string; disabled?: boolean }> = ({
  path,
  disabled,
}) => (
  <Form.NumericField
    path={`${path}.zIndexVal`}
    label="Z index value"
    inputProps={{ disabled }}
  />
);

const ZIndexPhaseField: FC<{ path: string; disabled?: boolean }> = ({
  path,
  disabled,
}) => (
  <Form.Field<string> path={`${path}.zIndexPhase`} label="Z index phase">
    {({ value, onChange, preview }) => (
      <Select.Simple<string>
        value={value}
        onChange={(v: string) => onChange(v)}
        preview={preview}
        disabled={disabled}
        resourceName="phase"
      >
        <Select.Item itemKey="AHighBHigh">A high B high</Select.Item>
        <Select.Item itemKey="AHighBLow">A high B low</Select.Item>
        <Select.Item itemKey="ALowBHigh">A low B high</Select.Item>
        <Select.Item itemKey="ALowBLow">A low B low</Select.Item>
      </Select.Simple>
    )}
  </Form.Field>
);

const TerminalZField = ({ path, disabled }: { path: string; disabled?: boolean }) => (
  <TerminalField
    path={path}
    fieldKey="terminalZ"
    label="Input terminal Z"
    disabled={disabled}
  />
);

const CI_LINEAR_POSITION_UNITS_NAMES = {
  Meters: "Meters",
  Inches: "Inches",
  Ticks: "Ticks",
} as const satisfies Record<CILinearPositionUnits, string>;

const LinearPositionUnitsField = Form.buildSelectField<CILinearPositionUnits>({
  fieldKey: "units",
  fieldProps: { label: "Units" },
  inputProps: {
    resourceName: "units",
    children: Task.selectItems(CI_LINEAR_POSITION_UNITS_NAMES),
  },
});

const CI_ANGULAR_POSITION_UNITS_NAMES = {
  Degrees: "Degrees",
  Radians: "Radians",
  Ticks: "Ticks",
} as const satisfies Record<CIAngularPositionUnits, string>;

const AngularPositionUnitsField = Form.buildSelectField<CIAngularPositionUnits>({
  fieldKey: "units",
  fieldProps: { label: "Units" },
  inputProps: {
    resourceName: "units",
    children: Task.selectItems(CI_ANGULAR_POSITION_UNITS_NAMES),
  },
});

const useMeasMethodVisibility = (prefix: string) => {
  const measMethod = Form.useFieldValue<CIMeasMethod>(`${prefix}.measMethod`, {
    optional: true,
  });
  return {
    showMeasTime: measMethod === "HighFreq2Ctr",
    showDivisor: measMethod === "LargeRng2Ctr",
  };
};

const useZIndexFieldsDisabled = (prefix: string) => {
  const zIndexEnabled = Form.useFieldValue<boolean>(`${prefix}.zIndexEnabled`, {
    optional: true,
  });
  return !zIndexEnabled;
};

const FrequencyForm: FC<FormProps> = ({ prefix }) => {
  const { showMeasTime, showDivisor } = useMeasMethodVisibility(prefix);
  return (
    <>
      <MinMaxValueFields path={prefix} />
      <EdgeField path={prefix} />
      <UnitsField path={prefix} />
      <InputTerminalField path={prefix} />
      <MeasMethodField path={prefix} />
      {showMeasTime && <MeasTimeField path={prefix} />}
      {showDivisor && <DivisorField path={prefix} />}
    </>
  );
};

const PeriodForm: FC<FormProps> = ({ prefix }) => {
  const { showMeasTime, showDivisor } = useMeasMethodVisibility(prefix);
  return (
    <>
      <MinMaxValueFields path={prefix} />
      <StartingEdgeField path={prefix} />
      <PeriodUnitsField path={prefix} />
      <InputTerminalField path={prefix} />
      <MeasMethodField path={prefix} />
      {showMeasTime && <MeasTimeField path={prefix} />}
      {showDivisor && <DivisorField path={prefix} />}
    </>
  );
};

const LinearPositionForm: FC<FormProps> = ({ prefix }) => {
  const zIndexFieldsDisabled = useZIndexFieldsDisabled(prefix);
  return (
    <>
      <InitialPosField path={prefix} />
      <DistPerPulseField path={prefix} />
      <LinearPositionUnitsField path={prefix} />
      <TerminalAField path={prefix} />
      <TerminalBField path={prefix} />
      <DecodingTypeField path={prefix} />
      <ZIndexEnabledField path={prefix} />
      <ZIndexValField path={prefix} disabled={zIndexFieldsDisabled} />
      <ZIndexPhaseField path={prefix} disabled={zIndexFieldsDisabled} />
      <TerminalZField path={prefix} disabled={zIndexFieldsDisabled} />
    </>
  );
};

const AngularPositionForm: FC<FormProps> = ({ prefix }) => {
  const zIndexFieldsDisabled = useZIndexFieldsDisabled(prefix);
  return (
    <>
      <PulsesPerRevField path={prefix} />
      <InitialAngleField path={prefix} />
      <AngularPositionUnitsField path={prefix} />
      <TerminalAField path={prefix} />
      <TerminalBField path={prefix} />
      <DecodingTypeField path={prefix} />
      <ZIndexEnabledField path={prefix} />
      <ZIndexValField path={prefix} disabled={zIndexFieldsDisabled} />
      <ZIndexPhaseField path={prefix} disabled={zIndexFieldsDisabled} />
      <TerminalZField path={prefix} disabled={zIndexFieldsDisabled} />
    </>
  );
};

const CHANNEL_FORMS: Record<CIChannelType, FC<FormProps>> = {
  ci_frequency: FrequencyForm,
  ci_edge_count: ({ prefix }: FormProps) => (
    <>
      <ActiveEdgeField path={prefix} />
      <CountDirectionField path={prefix} />
      <InputTerminalField path={prefix} />
      <InitialCountField path={prefix} />
    </>
  ),
  ci_period: PeriodForm,
  ci_pulse_width: ({ prefix }: FormProps) => (
    <>
      <MinMaxValueFields path={prefix} />
      <StartingEdgeField path={prefix} />
      <PulseWidthUnitsField path={prefix} />
      <InputTerminalField path={prefix} />
    </>
  ),
  ci_semi_period: ({ prefix }: FormProps) => (
    <>
      <MinMaxValueFields path={prefix} />
      <SemiPeriodUnitsField path={prefix} />
    </>
  ),
  ci_two_edge_sep: ({ prefix }: FormProps) => (
    <>
      <MinMaxValueFields path={prefix} />
      <TwoEdgeSepUnitsField path={prefix} />
      <FirstEdgeField path={prefix} />
      <SecondEdgeField path={prefix} />
    </>
  ),
  ci_velocity_linear: ({ prefix }: FormProps) => (
    <>
      <MinMaxValueFields path={prefix} />
      <LinearVelocityUnitsField path={prefix} />
      <DistPerPulseField path={prefix} />
      <DecodingTypeField path={prefix} />
      <TerminalAField path={prefix} />
      <TerminalBField path={prefix} />
    </>
  ),
  ci_velocity_angular: ({ prefix }: FormProps) => (
    <>
      <MinMaxValueFields path={prefix} />
      <AngularVelocityUnitsField path={prefix} />
      <PulsesPerRevField path={prefix} />
      <DecodingTypeField path={prefix} />
      <TerminalAField path={prefix} />
      <TerminalBField path={prefix} />
    </>
  ),
  ci_position_linear: LinearPositionForm,
  ci_position_angular: AngularPositionForm,
  ci_duty_cycle: ({ prefix }: FormProps) => (
    <>
      <MinMaxValueFields path={prefix} />
      <ActiveEdgeField path={prefix} />
      <InputTerminalField path={prefix} />
    </>
  ),
};

export interface CIChannelFormProps {
  type: CIChannelType;
  prefix: string;
}

const CI_CHANNEL_TYPES = record.keys(CI_CHANNEL_TYPE_NAMES);

const deviceOf = (c: CIChannel) => c.device;

export const CIChannelForm = ({ type, prefix }: CIChannelFormProps) => {
  const TypeForm = CHANNEL_FORMS[type];
  const device = useChannelDevice(prefix);
  const port = Form.useFieldValue<number>(`${prefix}.port`);
  const counter = device?.properties.hardware?.counterInputs.find(
    (c) => c.port === port,
  );
  const types = allowedTypes(CI_CHANNEL_TYPES, type, counter);
  return (
    <Form.Sections>
      <Form.Section title="Source">
        <SelectDevice path={`${prefix}.device`} />
        <PortField
          path={prefix}
          device={device}
          kind={COUNTER_INPUT}
          deviceOf={deviceOf}
        />
      </Form.Section>
      <Form.Section title="Signal">
        <SelectCIChannelTypeField
          path={prefix}
          inputProps={{ allowNone: false, children: channelTypeItems(types) }}
        />
        <TypeForm prefix={prefix} />
      </Form.Section>
      <CustomScaleSection prefix={prefix} />
    </Form.Sections>
  );
};
