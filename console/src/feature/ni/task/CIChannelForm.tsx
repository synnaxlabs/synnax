// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Divider, Flex, Form, Select } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import { type FC } from "react";

import { PortField } from "@/feature/ni/device/PortField";
import { Select as SelectDevice } from "@/feature/ni/device/Select";
import { CustomScaleForm } from "@/feature/ni/task/CustomScaleForm";
import { MinMaxValueFields } from "@/feature/ni/task/MinMaxValueFields";
import { selectData } from "@/feature/ni/task/selectData";
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

const CI_FREQ_UNITS_NAMES = {
  Hz: "Hz",
  Ticks: "Ticks",
} as const satisfies Record<CIFreqUnits, string>;

const CI_TIME_UNITS_NAMES = {
  Seconds: "Seconds",
  Ticks: "Ticks",
} as const satisfies Record<CITimeUnits, string>;

const CI_EDGE_NAMES = {
  Rising: "Rising",
  Falling: "Falling",
} as const satisfies Record<CIEdge, string>;

const CI_COUNT_DIRECTION_NAMES = {
  CountUp: "Count up",
  CountDown: "Count down",
  ExternallyControlled: "Externally controlled",
} as const satisfies Record<CICountDirection, string>;

const CI_MEAS_METHOD_NAMES = {
  LowFreq1Ctr: "One counter (low frequency)",
  HighFreq2Ctr: "Two counters (high frequency)",
  LargeRng2Ctr: "Two counters (large range)",
  DynamicAvg: "Dynamic averaging",
} as const satisfies Record<CIMeasMethod, string>;

const CI_DECODING_TYPE_NAMES = {
  X1: "X1",
  X2: "X2",
  X4: "X4",
  TwoPulse: "Two pulse",
} as const satisfies Record<CIDecodingType, string>;

const CI_LINEAR_VELOCITY_UNITS_NAMES = {
  "m/s": "m/s",
  "in/s": "in/s",
} as const satisfies Record<CILinearVelocityUnits, string>;

const CI_ANGULAR_VELOCITY_UNITS_NAMES = {
  RPM: "RPM",
  "Radians/s": "Radians/s",
  "Degrees/s": "Degrees/s",
} as const satisfies Record<CIAngularVelocityUnits, string>;

const CI_LINEAR_POSITION_UNITS_NAMES = {
  Meters: "Meters",
  Inches: "Inches",
  Ticks: "Ticks",
} as const satisfies Record<CILinearPositionUnits, string>;

const CI_ANGULAR_POSITION_UNITS_NAMES = {
  Degrees: "Degrees",
  Radians: "Radians",
  Ticks: "Ticks",
} as const satisfies Record<CIAngularPositionUnits, string>;

interface FormProps {
  prefix: string;
}

const UnitsField = Form.buildSelectField<CIFreqUnits, record.KeyedNamed<CIFreqUnits>>({
  fieldKey: "units",
  fieldProps: { label: "Units" },
  inputProps: {
    resourceName: "units",
    data: selectData(CI_FREQ_UNITS_NAMES),
  },
});

const PeriodUnitsField = Form.buildSelectField<
  CIPeriodUnits,
  record.KeyedNamed<CIPeriodUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Units" },
  inputProps: {
    resourceName: "units",
    data: selectData(CI_TIME_UNITS_NAMES),
  },
});

const PulseWidthUnitsField = Form.buildSelectField<
  CIPulseWidthUnits,
  record.KeyedNamed<CIPulseWidthUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Scaled units" },
  inputProps: {
    resourceName: "scaled units",
    data: selectData(CI_TIME_UNITS_NAMES),
  },
});

const SemiPeriodUnitsField = Form.buildSelectField<
  CISemiPeriodUnits,
  record.KeyedNamed<CISemiPeriodUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Scaled units" },
  inputProps: {
    resourceName: "scaled units",
    data: selectData(CI_TIME_UNITS_NAMES),
  },
});

const TwoEdgeSepUnitsField = Form.buildSelectField<
  CITwoEdgeSepUnits,
  record.KeyedNamed<CITwoEdgeSepUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Scaled units" },
  inputProps: {
    resourceName: "scaled units",
    data: selectData(CI_TIME_UNITS_NAMES),
  },
});

const EdgeField = Form.buildSelectField<CIEdge, record.KeyedNamed<CIEdge>>({
  fieldKey: "edge",
  fieldProps: { label: "Starting edge" },
  inputProps: {
    resourceName: "starting edge",
    data: selectData(CI_EDGE_NAMES),
  },
});

const StartingEdgeField = Form.buildSelectField<CIEdge, record.KeyedNamed<CIEdge>>({
  fieldKey: "startingEdge",
  fieldProps: { label: "Starting edge" },
  inputProps: {
    resourceName: "starting edge",
    data: selectData(CI_EDGE_NAMES),
  },
});

const ActiveEdgeField = Form.buildSelectField<CIEdge, record.KeyedNamed<CIEdge>>({
  fieldKey: "activeEdge",
  fieldProps: { label: "Active edge" },
  inputProps: {
    resourceName: "active edge",
    data: selectData(CI_EDGE_NAMES),
  },
});

const CountDirectionField = Form.buildSelectField<
  CICountDirection,
  record.KeyedNamed<CICountDirection>
>({
  fieldKey: "countDirection",
  fieldProps: { label: "Count direction" },
  inputProps: {
    resourceName: "count direction",
    data: selectData(CI_COUNT_DIRECTION_NAMES),
  },
});

const InitialCountField = Form.buildNumericField({
  fieldKey: "initialCount",
  fieldProps: { label: "Initial count" },
  inputProps: {},
});

const MeasMethodField = Form.buildSelectField<
  CIMeasMethod,
  record.KeyedNamed<CIMeasMethod>
>({
  fieldKey: "measMethod",
  fieldProps: { label: "Measurement method" },
  inputProps: {
    resourceName: "measurement method",
    data: selectData(CI_MEAS_METHOD_NAMES),
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

const TerminalField = Form.buildSelectField<string, record.KeyedNamed>({
  fieldKey: "terminal",
  fieldProps: { label: "Input terminal" },
  inputProps: {
    resourceName: "input terminal",
    allowNone: true,
    data: COUNTER_TERMINALS.map((t) => ({ key: t, name: t })),
  },
});

const FirstEdgeField = Form.buildSelectField<CIEdge, record.KeyedNamed<CIEdge>>({
  fieldKey: "firstEdge",
  fieldProps: { label: "Edge 1" },
  inputProps: {
    resourceName: "edge 1",
    data: selectData(CI_EDGE_NAMES),
  },
});

const SecondEdgeField = Form.buildSelectField<CIEdge, record.KeyedNamed<CIEdge>>({
  fieldKey: "secondEdge",
  fieldProps: { label: "Edge 2" },
  inputProps: {
    resourceName: "edge 2",
    data: selectData(CI_EDGE_NAMES),
  },
});

const TerminalAField = Form.buildSelectField<string, record.KeyedNamed>({
  fieldKey: "terminalA",
  fieldProps: { label: "Input terminal A" },
  inputProps: {
    resourceName: "input terminal A",
    allowNone: true,
    data: COUNTER_TERMINALS.map((t) => ({ key: t, name: t })),
  },
});

const TerminalBField = Form.buildSelectField<string, record.KeyedNamed>({
  fieldKey: "terminalB",
  fieldProps: { label: "Input terminal B" },
  inputProps: {
    resourceName: "input terminal B",
    allowNone: true,
    data: COUNTER_TERMINALS.map((t) => ({ key: t, name: t })),
  },
});

const DecodingTypeField = Form.buildSelectField<
  CIDecodingType,
  record.KeyedNamed<CIDecodingType>
>({
  fieldKey: "decodingType",
  fieldProps: { label: "Decoding type" },
  inputProps: {
    resourceName: "decoding type",
    data: selectData(CI_DECODING_TYPE_NAMES),
  },
});

const LinearVelocityUnitsField = Form.buildSelectField<
  CILinearVelocityUnits,
  record.KeyedNamed<CILinearVelocityUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Scaled units" },
  inputProps: {
    resourceName: "scaled units",
    data: selectData(CI_LINEAR_VELOCITY_UNITS_NAMES),
  },
});

const AngularVelocityUnitsField = Form.buildSelectField<
  CIAngularVelocityUnits,
  record.KeyedNamed<CIAngularVelocityUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Scaled units" },
  inputProps: {
    resourceName: "scaled units",
    data: selectData(CI_ANGULAR_VELOCITY_UNITS_NAMES),
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

const ZIndexEnableField: FC<{ path: string; grow?: boolean }> = ({ path }) => (
  <Form.SwitchField path={`${path}.zIndexEnable`} label="Z index enable" />
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
      <Select.Static
        value={value}
        onChange={(v: string) => onChange(v)}
        preview={preview}
        disabled={disabled}
        resourceName="phase"
        data={[
          { key: "AHighBHigh", name: "A high B high" },
          { key: "AHighBLow", name: "A high B low" },
          { key: "ALowBHigh", name: "A low B high" },
          { key: "ALowBLow", name: "A low B low" },
        ]}
      />
    )}
  </Form.Field>
);

const TerminalZField: FC<{ path: string; grow?: boolean; disabled?: boolean }> = ({
  path,
  disabled,
}) => (
  <Form.Field<string> path={`${path}.terminalZ`} label="Input terminal Z">
    {({ value, onChange, preview }) => (
      <Select.Static
        value={value}
        onChange={(v: string | null) => onChange(v ?? "")}
        preview={preview}
        allowNone
        disabled={disabled}
        resourceName="input terminal Z"
        data={COUNTER_TERMINALS.map((t) => ({ key: t, name: t }))}
      />
    )}
  </Form.Field>
);

const LinearPositionUnitsField = Form.buildSelectField<
  CILinearPositionUnits,
  record.KeyedNamed<CILinearPositionUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Units" },
  inputProps: {
    resourceName: "units",
    data: selectData(CI_LINEAR_POSITION_UNITS_NAMES),
  },
});

const AngularPositionUnitsField = Form.buildSelectField<
  CIAngularPositionUnits,
  record.KeyedNamed<CIAngularPositionUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Units" },
  inputProps: {
    resourceName: "units",
    data: selectData(CI_ANGULAR_POSITION_UNITS_NAMES),
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
  const zIndexEnable = Form.useFieldValue<boolean>(`${prefix}.zIndexEnable`, {
    optional: true,
  });
  return !zIndexEnable;
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
          <ZIndexEnableField path={prefix} grow />
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
          <ZIndexEnableField path={prefix} grow />
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
