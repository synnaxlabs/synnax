// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Divider } from "@synnaxlabs/lyra/divider";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Select } from "@synnaxlabs/lyra/select";
import { type FC } from "react";

import { PortField } from "@/feature/ni/device/PortField";
import { Select as SelectDevice } from "@/feature/ni/device/Select";
import { CustomScaleForm } from "@/feature/ni/task/CustomScaleForm";
import { MinMaxValueFields } from "@/feature/ni/task/MinMaxValueFields";
import {
  type CIAngularPositionUnits,
  type CIAngularVelocityUnits,
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
    children: Task.selectItems(CI_EDGE_NAMES),
  },
});

const StartingEdgeField = Form.buildSelectField<CIEdge>({
  fieldKey: "startingEdge",
  fieldProps: { label: "Starting edge" },
  inputProps: {
    resourceName: "starting edge",
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

const COUNTER_TERMINALS = [
  "PFI0",
  "PFI1",
  "PFI2",
  "PFI3",
  "PFI4",
  "PFI5",
  "PFI6",
  "PFI7",
  "PFI8",
  "PFI9",
  "PFI10",
  "PFI11",
  "PFI12",
  "PFI13",
  "PFI14",
  "PFI15",
] as const;

const COUNTER_TERMINAL_ITEMS = COUNTER_TERMINALS.map((t) => (
  <Select.Item key={t} itemKey={t}>
    {t}
  </Select.Item>
));

const TerminalField = Form.buildSelectField<string>({
  fieldKey: "terminal",
  fieldProps: { label: "Input terminal" },
  inputProps: {
    resourceName: "input terminal",
    allowNone: true,
    children: COUNTER_TERMINAL_ITEMS,
  },
});

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

const TerminalAField = Form.buildSelectField<string>({
  fieldKey: "terminalA",
  fieldProps: { label: "Input terminal A" },
  inputProps: {
    resourceName: "input terminal A",
    allowNone: true,
    children: COUNTER_TERMINAL_ITEMS,
  },
});

const TerminalBField = Form.buildSelectField<string>({
  fieldKey: "terminalB",
  fieldProps: { label: "Input terminal B" },
  inputProps: {
    resourceName: "input terminal B",
    allowNone: true,
    children: COUNTER_TERMINAL_ITEMS,
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
  fieldProps: { label: "Distance / Pulse" },
  inputProps: {},
});

const PulsesPerRevField = Form.buildNumericField({
  fieldKey: "pulsesPerRev",
  fieldProps: { label: "Pulses / Rev" },
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

const ZIndexEnabledField: FC<{ path: string; grow?: boolean }> = ({ path }) => (
  <Form.SwitchField path={`${path}.zIndexEnabled`} label="Z index enable" />
);

const ZIndexValField: FC<{ path: string; grow?: boolean; disabled?: boolean }> = ({
  path,
  disabled,
}) => (
  <Form.NumericField
    path={`${path}.zIndexVal`}
    label="Value"
    inputProps={{ disabled }}
  />
);

const ZIndexPhaseField: FC<{ path: string; grow?: boolean; disabled?: boolean }> = ({
  path,
  disabled,
}) => (
  <Form.Field<string> path={`${path}.zIndexPhase`} label="Phase">
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

const TerminalZField: FC<{ path: string; grow?: boolean; disabled?: boolean }> = ({
  path,
  disabled,
}) => (
  <Form.Field<string> path={`${path}.terminalZ`} label="Input terminal Z">
    {({ value, onChange, preview }) => (
      <Select.Simple<string>
        value={value}
        onChange={(v: string | null) => onChange(v ?? "")}
        preview={preview}
        allowNone
        disabled={disabled}
        resourceName="input terminal Z"
      >
        {COUNTER_TERMINAL_ITEMS}
      </Select.Simple>
    )}
  </Form.Field>
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

const CHANNEL_FORMS: Record<CIChannelType, FC<FormProps>> = {
  ci_frequency: ({ prefix }) => {
    const { showMeasTime, showDivisor } = useMeasMethodVisibility(prefix);
    return (
      <>
        <MinMaxValueFields path={prefix} />
        <Divider.Divider x padded="bottom" />
        <Flex.Box x>
          <EdgeField path={prefix} grow />
          <UnitsField path={prefix} grow />
        </Flex.Box>
        <Flex.Box x>
          <TerminalField path={prefix} grow />
          <MeasMethodField path={prefix} grow />
        </Flex.Box>
        {showMeasTime && (
          <Flex.Box x>
            <MeasTimeField path={prefix} grow />
          </Flex.Box>
        )}
        {showDivisor && (
          <Flex.Box x>
            <DivisorField path={prefix} grow />
          </Flex.Box>
        )}
        <Divider.Divider x padded="bottom" />
        <CustomScaleForm prefix={prefix} />
      </>
    );
  },
  ci_edge_count: ({ prefix }: FormProps) => (
    <>
      <Flex.Box x>
        <ActiveEdgeField path={prefix} grow />
        <CountDirectionField path={prefix} grow />
      </Flex.Box>
      <Flex.Box x>
        <TerminalField path={prefix} grow />
        <InitialCountField path={prefix} grow />
      </Flex.Box>
    </>
  ),
  ci_period: ({ prefix }: FormProps) => {
    const { showMeasTime, showDivisor } = useMeasMethodVisibility(prefix);
    return (
      <>
        <MinMaxValueFields path={prefix} />
        <Divider.Divider x padded="bottom" />
        <Flex.Box x>
          <StartingEdgeField path={prefix} grow />
          <PeriodUnitsField path={prefix} grow />
        </Flex.Box>
        <Flex.Box x>
          <TerminalField path={prefix} grow />
          <MeasMethodField path={prefix} grow />
        </Flex.Box>
        {showMeasTime && (
          <Flex.Box x>
            <MeasTimeField path={prefix} grow />
          </Flex.Box>
        )}
        {showDivisor && (
          <Flex.Box x>
            <DivisorField path={prefix} grow />
          </Flex.Box>
        )}
        <Divider.Divider x padded="bottom" />
        <CustomScaleForm prefix={prefix} />
      </>
    );
  },
  ci_pulse_width: ({ prefix }: FormProps) => (
    <>
      <MinMaxValueFields path={prefix} />
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <StartingEdgeField path={prefix} grow />
        <PulseWidthUnitsField path={prefix} grow />
      </Flex.Box>
      <Flex.Box x>
        <TerminalField path={prefix} grow />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ci_semi_period: ({ prefix }: FormProps) => (
    <>
      <MinMaxValueFields path={prefix} />
      <Divider.Divider x padded="bottom" />
      <SemiPeriodUnitsField path={prefix} />
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ci_two_edge_sep: ({ prefix }: FormProps) => (
    <>
      <MinMaxValueFields path={prefix} />
      <Divider.Divider x padded="bottom" />
      <TwoEdgeSepUnitsField path={prefix} />
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <FirstEdgeField path={prefix} grow />
        <SecondEdgeField path={prefix} grow />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ci_velocity_linear: ({ prefix }: FormProps) => (
    <>
      <MinMaxValueFields path={prefix} />
      <Divider.Divider x padded="bottom" />
      <LinearVelocityUnitsField path={prefix} />
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <DistPerPulseField path={prefix} grow />
        <DecodingTypeField path={prefix} grow />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <TerminalAField path={prefix} grow />
        <TerminalBField path={prefix} grow />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ci_velocity_angular: ({ prefix }: FormProps) => (
    <>
      <MinMaxValueFields path={prefix} />
      <Divider.Divider x padded="bottom" />
      <AngularVelocityUnitsField path={prefix} />
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <PulsesPerRevField path={prefix} grow />
        <DecodingTypeField path={prefix} grow />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <TerminalAField path={prefix} grow />
        <TerminalBField path={prefix} grow />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ci_position_linear: ({ prefix }: FormProps) => {
    const zIndexFieldsDisabled = useZIndexFieldsDisabled(prefix);
    return (
      <>
        <Flex.Box x>
          <InitialPosField path={prefix} grow />
          <DistPerPulseField path={prefix} grow />
          <LinearPositionUnitsField path={prefix} grow />
        </Flex.Box>
        <Divider.Divider x padded="bottom" />
        <Flex.Box x>
          <TerminalAField path={prefix} grow />
          <TerminalBField path={prefix} grow />
          <DecodingTypeField path={prefix} grow />
        </Flex.Box>
        <Divider.Divider x padded="bottom" />
        <Flex.Box x>
          <ZIndexEnabledField path={prefix} grow />
          <ZIndexValField path={prefix} grow disabled={zIndexFieldsDisabled} />
          <ZIndexPhaseField path={prefix} grow disabled={zIndexFieldsDisabled} />
          <TerminalZField path={prefix} grow disabled={zIndexFieldsDisabled} />
        </Flex.Box>
        <Divider.Divider x padded="bottom" />
        <CustomScaleForm prefix={prefix} />
      </>
    );
  },
  ci_position_angular: ({ prefix }: FormProps) => {
    const zIndexFieldsDisabled = useZIndexFieldsDisabled(prefix);
    return (
      <>
        <Flex.Box x>
          <PulsesPerRevField path={prefix} grow />
          <InitialAngleField path={prefix} grow />
          <AngularPositionUnitsField path={prefix} grow />
        </Flex.Box>
        <Divider.Divider x padded="bottom" />
        <Flex.Box x>
          <TerminalAField path={prefix} grow />
          <TerminalBField path={prefix} grow />
          <DecodingTypeField path={prefix} grow />
        </Flex.Box>
        <Divider.Divider x padded="bottom" />
        <Flex.Box x>
          <ZIndexEnabledField path={prefix} grow />
          <ZIndexValField path={prefix} grow disabled={zIndexFieldsDisabled} />
          <ZIndexPhaseField path={prefix} grow disabled={zIndexFieldsDisabled} />
          <TerminalZField path={prefix} grow disabled={zIndexFieldsDisabled} />
        </Flex.Box>
        <Divider.Divider x padded="bottom" />
        <CustomScaleForm prefix={prefix} />
      </>
    );
  },
  ci_duty_cycle: ({ prefix }: FormProps) => (
    <>
      <MinMaxValueFields path={prefix} />
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <ActiveEdgeField path={prefix} grow />
        <TerminalField path={prefix} grow />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
};

export interface CIChannelFormProps {
  type: CIChannelType;
  prefix: string;
}

export const CIChannelForm = ({ type, prefix }: CIChannelFormProps) => {
  const Form = CHANNEL_FORMS[type];
  return (
    <>
      <Flex.Box x wrap>
        <SelectDevice path={`${prefix}.device`} />
        <PortField path={prefix} />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <Form prefix={prefix} />
    </>
  );
};
