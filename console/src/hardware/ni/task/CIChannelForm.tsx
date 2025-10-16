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
  CI_FREQUENCY_CHAN_TYPE,
  type CIChannelType,
  type CIEdge,
  type CIFreqUnits,
  type CIMeasMethod,
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

const MeasMethodField = Form.buildSelectField<CIMeasMethod, record.KeyedNamed<CIMeasMethod>>({
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
