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
import { type deep, record, TimeSpan } from "@synnaxlabs/x";
import { type FC } from "react";

import { Select as SelectDevice } from "@/feature/ni/device/Select";
import { createTypeFields } from "@/feature/ni/task/createTypeFields";
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

type Variant<K extends CIChannelType> = Extract<CIChannel, { type: K }>;

interface FormProps<K extends CIChannelType = CIChannelType> {
  path: deep.Path<Variant<K>>;
}

const CI_FREQ_UNITS_NAMES = {
  Hz: "Hz",
  Ticks: "Ticks",
} as const satisfies Record<CIFreqUnits, string>;

const UnitsField = Form.buildSelectField<CIFreqUnits>({
  fieldProps: { label: "Units" },
  inputProps: {
    resourceName: "units",
    children: Task.selectItems(CI_FREQ_UNITS_NAMES),
  },
});

const PeriodUnitsField = Form.buildSelectField<CIPeriodUnits>({
  fieldProps: { label: "Units" },
  inputProps: {
    resourceName: "units",
    children: Task.selectItems(CI_TIME_UNITS_NAMES),
  },
});

const PulseWidthUnitsField = Form.buildSelectField<CIPulseWidthUnits>({
  fieldProps: { label: "Scaled units" },
  inputProps: {
    resourceName: "scaled units",
    children: Task.selectItems(CI_TIME_UNITS_NAMES),
  },
});

const SemiPeriodUnitsField = Form.buildSelectField<CISemiPeriodUnits>({
  fieldProps: { label: "Scaled units" },
  inputProps: {
    resourceName: "scaled units",
    children: Task.selectItems(CI_TIME_UNITS_NAMES),
  },
});

const TwoEdgeSepUnitsField = Form.buildSelectField<CITwoEdgeSepUnits>({
  fieldProps: { label: "Scaled units" },
  inputProps: {
    resourceName: "scaled units",
    children: Task.selectItems(CI_TIME_UNITS_NAMES),
  },
});

const EdgeField = Form.buildSelectField<CIEdge>({
  fieldProps: { label: "Starting edge" },
  inputProps: {
    resourceName: "starting edge",
    searchName: "edge",
    children: Task.selectItems(CI_EDGE_NAMES),
  },
});

const StartingEdgeField = Form.buildSelectField<CIEdge>({
  fieldProps: { label: "Starting edge" },
  inputProps: {
    resourceName: "starting edge",
    searchName: "edge",
    children: Task.selectItems(CI_EDGE_NAMES),
  },
});

const ActiveEdgeField = Form.buildSelectField<CIEdge>({
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
  fieldProps: { label: "Count direction" },
  inputProps: {
    resourceName: "count direction",
    searchName: "direction",
    children: Task.selectItems(CI_COUNT_DIRECTION_NAMES),
  },
});

const CI_MEAS_METHOD_NAMES = {
  LowFreq1Ctr: "One counter (low frequency)",
  HighFreq2Ctr: "Two counters (high frequency)",
  LargeRng2Ctr: "Two counters (large range)",
  DynamicAvg: "Dynamic averaging",
} as const satisfies Record<CIMeasMethod, string>;

const MeasMethodField = Form.buildSelectField<CIMeasMethod>({
  fieldProps: { label: "Measurement method" },
  inputProps: {
    resourceName: "measurement method",
    searchName: "method",
    children: Task.selectItems(CI_MEAS_METHOD_NAMES),
  },
});

const MeasTimeField = ({ path }: { path: deep.Target<number> }) => (
  <Form.TimeSpanField path={path} label="Measurement time" unit={TimeSpan.SECOND} />
);

const COUNT_INPUT_PROPS = { bounds: { lower: 1, upper: Infinity } } as const;

const INITIAL_COUNT_INPUT_PROPS = { bounds: { lower: 0, upper: Infinity } } as const;

const DivisorField = Form.buildNumericField({
  fieldProps: { label: "Divisor" },
  inputProps: COUNT_INPUT_PROPS,
});

// A device with no port list offers PFI0 to PFI15.
const FALLBACK_TERMINALS = Array.from({ length: 16 }, (_, i) => `PFI${i}`);

// The PFI terminals of the channel's device, and the current terminal.
const useTerminalItems = (path: deep.Target<{ device: string }>, current: string) => {
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
  path: deep.Target<string>;
  channelPath: deep.Target<{ device: string }>;
  label: string;
  disabled?: boolean;
}

const TerminalField = ({ path, channelPath, label, disabled }: TerminalFieldProps) => {
  const value = Form.useFieldValue(path);
  const items = useTerminalItems(channelPath, value);
  return (
    <Form.Field path={path} label={label}>
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

const InputTerminalField = ({
  path,
}: {
  path: deep.Path<{ device: string; terminal: string }>;
}) => <TerminalField path={path.terminal} channelPath={path} label="Input terminal" />;

const TerminalAField = ({
  path,
}: {
  path: deep.Path<{ device: string; terminalA: string }>;
}) => (
  <TerminalField path={path.terminalA} channelPath={path} label="Input terminal A" />
);

const TerminalBField = ({
  path,
}: {
  path: deep.Path<{ device: string; terminalB: string }>;
}) => (
  <TerminalField path={path.terminalB} channelPath={path} label="Input terminal B" />
);

const FirstEdgeField = Form.buildSelectField<CIEdge>({
  fieldProps: { label: "Edge 1" },
  inputProps: {
    resourceName: "edge 1",
    children: Task.selectItems(CI_EDGE_NAMES),
  },
});

const SecondEdgeField = Form.buildSelectField<CIEdge>({
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
  fieldProps: { label: "Scaled units" },
  inputProps: {
    resourceName: "scaled units",
    children: Task.selectItems(CI_ANGULAR_VELOCITY_UNITS_NAMES),
  },
});

const DistPerPulseField = Form.buildNumericField({
  fieldProps: { label: "Distance / pulse" },
  inputProps: {},
});

const PulsesPerRevField = Form.buildNumericField({
  fieldProps: { label: "Pulses / rev" },
  inputProps: COUNT_INPUT_PROPS,
});

const ZIndexEnabledField: FC<{ path: deep.Target<boolean> }> = ({ path }) => (
  <Form.CheckboxField path={path} label="Z index enable" />
);

const ZIndexValField: FC<{ path: deep.Target<number>; disabled?: boolean }> = ({
  path,
  disabled,
}) => <Form.NumericField path={path} label="Z index value" inputProps={{ disabled }} />;

const ZIndexPhaseField: FC<{ path: deep.Target<string>; disabled?: boolean }> = ({
  path,
  disabled,
}) => (
  <Form.Field path={path} label="Z index phase">
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

const TerminalZField = ({
  path,
  disabled,
}: {
  path: deep.Path<{ device: string; terminalZ: string }>;
  disabled?: boolean;
}) => (
  <TerminalField
    path={path.terminalZ}
    channelPath={path}
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
  fieldProps: { label: "Units" },
  inputProps: {
    resourceName: "units",
    children: Task.selectItems(CI_ANGULAR_POSITION_UNITS_NAMES),
  },
});

const useMeasMethodVisibility = (path: deep.Path<{ measMethod: CIMeasMethod }>) => {
  const measMethod = Form.useFieldValue(path.measMethod, { optional: true });
  return {
    showMeasTime: measMethod === "HighFreq2Ctr",
    showDivisor: measMethod === "LargeRng2Ctr",
  };
};

const useZIndexFieldsDisabled = (path: deep.Path<{ zIndexEnabled: boolean }>) => {
  const zIndexEnabled = Form.useFieldValue(path.zIndexEnabled, { optional: true });
  return !zIndexEnabled;
};

const FrequencyForm: FC<FormProps<"ci_frequency">> = ({ path }) => {
  const { showMeasTime, showDivisor } = useMeasMethodVisibility(path);
  return (
    <>
      <MinMaxValueFields path={path} />
      <EdgeField path={path.edge} />
      <UnitsField path={path.units} />
      <InputTerminalField path={path} />
      <MeasMethodField path={path.measMethod} />
      {showMeasTime && <MeasTimeField path={path.measTime} />}
      {showDivisor && <DivisorField path={path.divisor} />}
    </>
  );
};

const PeriodForm: FC<FormProps<"ci_period">> = ({ path }) => {
  const { showMeasTime, showDivisor } = useMeasMethodVisibility(path);
  return (
    <>
      <MinMaxValueFields path={path} />
      <StartingEdgeField path={path.startingEdge} />
      <PeriodUnitsField path={path.units} />
      <InputTerminalField path={path} />
      <MeasMethodField path={path.measMethod} />
      {showMeasTime && <MeasTimeField path={path.measTime} />}
      {showDivisor && <DivisorField path={path.divisor} />}
    </>
  );
};

const LinearPositionForm: FC<FormProps<"ci_position_linear">> = ({ path }) => {
  const zIndexFieldsDisabled = useZIndexFieldsDisabled(path);
  return (
    <>
      <Form.NumericField path={path.initialPos} label="Initial position" />
      <DistPerPulseField path={path.distPerPulse} />
      <LinearPositionUnitsField path={path.units} />
      <TerminalAField path={path} />
      <TerminalBField path={path} />
      <DecodingTypeField path={path.decodingType} />
      <ZIndexEnabledField path={path.zIndexEnabled} />
      <ZIndexValField path={path.zIndexVal} disabled={zIndexFieldsDisabled} />
      <ZIndexPhaseField path={path.zIndexPhase} disabled={zIndexFieldsDisabled} />
      <TerminalZField path={path} disabled={zIndexFieldsDisabled} />
    </>
  );
};

const AngularPositionForm: FC<FormProps<"ci_position_angular">> = ({ path }) => {
  const zIndexFieldsDisabled = useZIndexFieldsDisabled(path);
  return (
    <>
      <PulsesPerRevField path={path.pulsesPerRev} />
      <Form.NumericField path={path.initialAngle} label="Initial angle" />
      <AngularPositionUnitsField path={path.units} />
      <TerminalAField path={path} />
      <TerminalBField path={path} />
      <DecodingTypeField path={path.decodingType} />
      <ZIndexEnabledField path={path.zIndexEnabled} />
      <ZIndexValField path={path.zIndexVal} disabled={zIndexFieldsDisabled} />
      <ZIndexPhaseField path={path.zIndexPhase} disabled={zIndexFieldsDisabled} />
      <TerminalZField path={path} disabled={zIndexFieldsDisabled} />
    </>
  );
};

const CHANNEL_FORMS: { [K in CIChannelType]: FC<FormProps<K>> } = {
  ci_frequency: FrequencyForm,
  ci_edge_count: ({ path }) => (
    <>
      <ActiveEdgeField path={path.activeEdge} />
      <CountDirectionField path={path.countDirection} />
      <InputTerminalField path={path} />
      <Form.NumericField
        path={path.initialCount}
        label="Initial count"
        inputProps={INITIAL_COUNT_INPUT_PROPS}
      />
    </>
  ),
  ci_period: PeriodForm,
  ci_pulse_width: ({ path }) => (
    <>
      <MinMaxValueFields path={path} />
      <StartingEdgeField path={path.startingEdge} />
      <PulseWidthUnitsField path={path.units} />
      <InputTerminalField path={path} />
    </>
  ),
  ci_semi_period: ({ path }) => (
    <>
      <MinMaxValueFields path={path} />
      <SemiPeriodUnitsField path={path.units} />
    </>
  ),
  ci_two_edge_sep: ({ path }) => (
    <>
      <MinMaxValueFields path={path} />
      <TwoEdgeSepUnitsField path={path.units} />
      <FirstEdgeField path={path.firstEdge} />
      <SecondEdgeField path={path.secondEdge} />
    </>
  ),
  ci_velocity_linear: ({ path }) => (
    <>
      <MinMaxValueFields path={path} />
      <LinearVelocityUnitsField path={path.units} />
      <DistPerPulseField path={path.distPerPulse} />
      <DecodingTypeField path={path.decodingType} />
      <TerminalAField path={path} />
      <TerminalBField path={path} />
    </>
  ),
  ci_velocity_angular: ({ path }) => (
    <>
      <MinMaxValueFields path={path} />
      <AngularVelocityUnitsField path={path.units} />
      <PulsesPerRevField path={path.pulsesPerRev} />
      <DecodingTypeField path={path.decodingType} />
      <TerminalAField path={path} />
      <TerminalBField path={path} />
    </>
  ),
  ci_position_linear: LinearPositionForm,
  ci_position_angular: AngularPositionForm,
  ci_duty_cycle: ({ path }) => (
    <>
      <MinMaxValueFields path={path} />
      <ActiveEdgeField path={path.activeEdge} />
      <InputTerminalField path={path} />
    </>
  ),
};

export interface CIChannelFormProps {
  type: CIChannelType;
  path: deep.Path<CIChannel>;
  /** The path of the task's channels. */
  channelsPath: deep.Target<CIChannel[]>;
}

const CI_CHANNEL_TYPES = record.keys(CI_CHANNEL_TYPE_NAMES);

const deviceOf = (c: CIChannel) => c.device;

const TypeFields = createTypeFields<CIChannel>(CHANNEL_FORMS);

export const CIChannelForm = ({ type, path, channelsPath }: CIChannelFormProps) => {
  const device = useChannelDevice(path);
  const port = Form.useFieldValue(path.port);
  const counter = device?.properties.hardware?.counterInputs.find(
    (c) => c.port === port,
  );
  const types = allowedTypes(CI_CHANNEL_TYPES, type, counter);
  return (
    <Form.Sections>
      <Form.Section title="Source">
        <SelectDevice path={path.device} />
        <PortField
          path={path}
          channelsPath={channelsPath}
          device={device}
          kind={COUNTER_INPUT}
          deviceOf={deviceOf}
        />
      </Form.Section>
      <Form.Section title="Signal">
        <SelectCIChannelTypeField
          path={path}
          inputProps={{ allowNone: false, children: channelTypeItems(types) }}
        />
        <TypeFields type={type} path={path} />
      </Form.Section>
      <CustomScaleSection path={path} />
    </Form.Sections>
  );
};
