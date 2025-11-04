// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Divider, Flex, Form } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import { type FC } from "react";

import { Device } from "@/hardware/ni/device";
import { CustomScaleForm } from "@/hardware/ni/task/CustomScaleForm";
import { MinMaxValueFields } from "@/hardware/ni/task/MinMaxValueFields";
import {
  CI_EDGE_COUNT_CHAN_TYPE,
  CI_FREQUENCY_CHAN_TYPE,
  CI_PERIOD_CHAN_TYPE,
  CI_PULSE_WIDTH_CHAN_TYPE,
  CI_SEMI_PERIOD_CHAN_TYPE,
  CI_TWO_EDGE_SEP_CHAN_TYPE,
  CI_VELOCITY_ANGULAR_CHAN_TYPE,
  CI_VELOCITY_LINEAR_CHAN_TYPE,
  type CIAngularVelocityUnits,
  type CIChannelType,
  type CICountDirection,
  type CIDecodingType,
  type CIEdge,
  type CIFreqUnits,
  type CILinearVelocityUnits,
  type CIMeasMethod,
  type CIPeriodUnits,
  type CIPulseWidthUnits,
  type CISemiPeriodUnits,
  type CITwoEdgeSepUnits,
} from "@/hardware/ni/task/types";

interface FormProps {
  prefix: string;
}

const UnitsField = Form.buildSelectField<CIFreqUnits, record.KeyedNamed<CIFreqUnits>>({
  fieldKey: "units",
  fieldProps: { label: "Units" },
  inputProps: {
    resourceName: "Units",
    data: [
      { key: "Hz", name: "Hz" },
      { key: "Ticks", name: "Ticks" },
    ],
  },
});

const PeriodUnitsField = Form.buildSelectField<
  CIPeriodUnits,
  record.KeyedNamed<CIPeriodUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Units" },
  inputProps: {
    resourceName: "Units",
    data: [
      { key: "Seconds", name: "Seconds" },
      { key: "Ticks", name: "Ticks" },
      { key: "FromCustomScale", name: "Custom" },
    ],
  },
});

const PulseWidthUnitsField = Form.buildSelectField<
  CIPulseWidthUnits,
  record.KeyedNamed<CIPulseWidthUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Scaled Units" },
  inputProps: {
    resourceName: "Scaled Units",
    data: [
      { key: "Seconds", name: "Seconds" },
      { key: "Ticks", name: "Ticks" },
      { key: "FromCustomScale", name: "Custom" },
    ],
  },
});

const SemiPeriodUnitsField = Form.buildSelectField<
  CISemiPeriodUnits,
  record.KeyedNamed<CISemiPeriodUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Scaled Units" },
  inputProps: {
    resourceName: "Scaled Units",
    data: [
      { key: "Seconds", name: "Seconds" },
      { key: "Ticks", name: "Ticks" },
      { key: "FromCustomScale", name: "Custom" },
    ],
  },
});

const TwoEdgeSepUnitsField = Form.buildSelectField<
  CITwoEdgeSepUnits,
  record.KeyedNamed<CITwoEdgeSepUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Scaled Units" },
  inputProps: {
    resourceName: "Scaled Units",
    data: [
      { key: "Seconds", name: "Seconds" },
      { key: "Ticks", name: "Ticks" },
    ],
  },
});

const EdgeField = Form.buildSelectField<CIEdge, record.KeyedNamed<CIEdge>>({
  fieldKey: "edge",
  fieldProps: { label: "Starting Edge" },
  inputProps: {
    resourceName: "Starting Edge",
    data: [
      { key: "Rising", name: "Rising" },
      { key: "Falling", name: "Falling" },
    ],
  },
});

const StartingEdgeField = Form.buildSelectField<CIEdge, record.KeyedNamed<CIEdge>>({
  fieldKey: "startingEdge",
  fieldProps: { label: "Starting Edge" },
  inputProps: {
    resourceName: "Starting Edge",
    data: [
      { key: "Rising", name: "Rising" },
      { key: "Falling", name: "Falling" },
    ],
  },
});

const ActiveEdgeField = Form.buildSelectField<CIEdge, record.KeyedNamed<CIEdge>>({
  fieldKey: "activeEdge",
  fieldProps: { label: "Active Edge" },
  inputProps: {
    resourceName: "Active Edge",
    data: [
      { key: "Rising", name: "Rising" },
      { key: "Falling", name: "Falling" },
    ],
  },
});

const CountDirectionField = Form.buildSelectField<
  CICountDirection,
  record.KeyedNamed<CICountDirection>
>({
  fieldKey: "countDirection",
  fieldProps: { label: "Count Direction" },
  inputProps: {
    resourceName: "Count Direction",
    data: [
      { key: "CountUp", name: "Count Up" },
      { key: "CountDown", name: "Count Down" },
      { key: "ExternallyControlled", name: "Externally Controlled" },
    ],
  },
});

const InitialCountField = Form.buildNumericField({
  fieldKey: "initialCount",
  fieldProps: { label: "Initial Count" },
  inputProps: {},
});

const MeasMethodField = Form.buildSelectField<
  CIMeasMethod,
  record.KeyedNamed<CIMeasMethod>
>({
  fieldKey: "measMethod",
  fieldProps: { label: "Measurement Method" },
  inputProps: {
    resourceName: "Measurement Method",
    data: [
      { key: "LowFreq1Ctr", name: "1 Counter (Low Frequency)" },
      { key: "HighFreq2Ctr", name: "2 Counters (High Frequency)" },
      { key: "LargeRng2Ctr", name: "2 Counters (Large Range)" },
      { key: "DynamicAvg", name: "Dynamic Averaging" },
    ],
  },
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
  fieldProps: { label: "Input Terminal" },
  inputProps: {
    resourceName: "Input Terminal",
    allowNone: true,
    data: COUNTER_TERMINALS.map((t) => ({ key: t, name: t })),
  },
});

const FirstEdgeField = Form.buildSelectField<CIEdge, record.KeyedNamed<CIEdge>>({
  fieldKey: "firstEdge",
  fieldProps: { label: "Edge 1" },
  inputProps: {
    resourceName: "Edge 1",
    data: [
      { key: "Rising", name: "Rising" },
      { key: "Falling", name: "Falling" },
    ],
  },
});

const SecondEdgeField = Form.buildSelectField<CIEdge, record.KeyedNamed<CIEdge>>({
  fieldKey: "secondEdge",
  fieldProps: { label: "Edge 2" },
  inputProps: {
    resourceName: "Edge 2",
    data: [
      { key: "Rising", name: "Rising" },
      { key: "Falling", name: "Falling" },
    ],
  },
});

const _FirstTerminalField = Form.buildSelectField<string, record.KeyedNamed>({
  fieldKey: "firstTerminal",
  fieldProps: { label: "First Terminal" },
  inputProps: {
    resourceName: "First Terminal",
    allowNone: true,
    data: COUNTER_TERMINALS.map((t) => ({ key: t, name: t })),
  },
});

const _SecondTerminalField = Form.buildSelectField<string, record.KeyedNamed>({
  fieldKey: "secondTerminal",
  fieldProps: { label: "Second Terminal" },
  inputProps: {
    resourceName: "Second Terminal",
    allowNone: true,
    data: COUNTER_TERMINALS.map((t) => ({ key: t, name: t })),
  },
});

const TerminalAField = Form.buildSelectField<string, record.KeyedNamed>({
  fieldKey: "terminalA",
  fieldProps: { label: "Input Terminal A" },
  inputProps: {
    resourceName: "Input Terminal A",
    allowNone: true,
    data: COUNTER_TERMINALS.map((t) => ({ key: t, name: t })),
  },
});

const TerminalBField = Form.buildSelectField<string, record.KeyedNamed>({
  fieldKey: "terminalB",
  fieldProps: { label: "Input Terminal B" },
  inputProps: {
    resourceName: "Input Terminal B",
    allowNone: true,
    data: COUNTER_TERMINALS.map((t) => ({ key: t, name: t })),
  },
});

const DecodingTypeField = Form.buildSelectField<
  CIDecodingType,
  record.KeyedNamed<CIDecodingType>
>({
  fieldKey: "decodingType",
  fieldProps: { label: "Decoding Type" },
  inputProps: {
    resourceName: "Decoding Type",
    data: [
      { key: "X1", name: "X1" },
      { key: "X2", name: "X2" },
      { key: "X4", name: "X4" },
      { key: "TwoPulse", name: "Two Pulse" },
    ],
  },
});

const LinearVelocityUnitsField = Form.buildSelectField<
  CILinearVelocityUnits,
  record.KeyedNamed<CILinearVelocityUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Scaled Units" },
  inputProps: {
    resourceName: "Scaled Units",
    data: [
      { key: "m/s", name: "m/s" },
      { key: "in/s", name: "in/s" },
      { key: "FromCustomScale", name: "Custom" },
    ],
  },
});

const AngularVelocityUnitsField = Form.buildSelectField<
  CIAngularVelocityUnits,
  record.KeyedNamed<CIAngularVelocityUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Scaled Units" },
  inputProps: {
    resourceName: "Scaled Units",
    data: [
      { key: "RPM", name: "RPM" },
      { key: "Radians/s", name: "Radians/s" },
      { key: "Degrees/s", name: "Degrees/s" },
      { key: "FromCustomScale", name: "Custom" },
    ],
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

const CHANNEL_FORMS: Record<CIChannelType, FC<FormProps>> = {
  [CI_FREQUENCY_CHAN_TYPE]: ({ prefix }) => (
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
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  [CI_EDGE_COUNT_CHAN_TYPE]: ({ prefix }: FormProps) => (
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
  [CI_PERIOD_CHAN_TYPE]: ({ prefix }: FormProps) => (
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
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  [CI_PULSE_WIDTH_CHAN_TYPE]: ({ prefix }: FormProps) => (
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
  [CI_SEMI_PERIOD_CHAN_TYPE]: ({ prefix }: FormProps) => (
    <>
      <MinMaxValueFields path={prefix} />
      <Divider.Divider x padded="bottom" />
      <SemiPeriodUnitsField path={prefix} />
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  [CI_TWO_EDGE_SEP_CHAN_TYPE]: ({ prefix }: FormProps) => (
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
  [CI_VELOCITY_LINEAR_CHAN_TYPE]: ({ prefix }: FormProps) => (
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
  [CI_VELOCITY_ANGULAR_CHAN_TYPE]: ({ prefix }: FormProps) => (
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
        <Device.Select path={`${prefix}.device`} />
        <Device.PortField path={prefix} />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <Form prefix={prefix} />
    </>
  );
};
