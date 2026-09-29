// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { serial } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select as LyraSelect } from "@synnaxlabs/lyra/select";
import { type ReactElement } from "react";

import { Bus } from "@/feature/bus";
import { Task } from "@/platform/task";

const PARITY_NAMES = {
  none: "None",
  even: "Even",
  odd: "Odd",
  mark: "Mark",
  space: "Space",
} as const satisfies Record<serial.Parity, string>;

const STOP_BITS_NAMES = {
  "1": "1",
  "1.5": "1.5",
  "2": "2",
} as const satisfies Record<serial.StopBits, string>;

const FLOW_CONTROL_NAMES = {
  none: "None",
  hardware: "Hardware",
  software: "Software",
} as const satisfies Record<serial.FlowControl, string>;

const DATA_BITS = [5, 6, 7, 8] as const;
type DataBits = (typeof DATA_BITS)[number];

const ParityField = Form.buildSelectField<serial.Parity>({
  fieldKey: "parity",
  fieldProps: { label: "Parity" },
  inputProps: { resourceName: "parity", children: Task.selectItems(PARITY_NAMES) },
});

const StopBitsField = Form.buildSelectField<serial.StopBits>({
  fieldKey: "stopBits",
  fieldProps: { label: "Stop bits" },
  inputProps: {
    resourceName: "stop bits",
    children: Task.selectItems(STOP_BITS_NAMES),
  },
});

const FlowControlField = Form.buildSelectField<serial.FlowControl>({
  fieldKey: "flowControl",
  fieldProps: { label: "Flow control" },
  inputProps: {
    resourceName: "flow control",
    children: Task.selectItems(FLOW_CONTROL_NAMES),
  },
});

const DataBitsField = Form.buildSelectField<DataBits>({
  fieldKey: "dataBits",
  fieldProps: { label: "Data bits" },
  inputProps: {
    resourceName: "data bits",
    // Numeric keys, so the items cannot come from a name record.
    children: DATA_BITS.map((bits) => (
      <LyraSelect.Item<DataBits> key={bits} itemKey={bits}>
        {bits.toString()}
      </LyraSelect.Item>
    )),
  },
});

const PORT_INPUT_PROPS = { placeholder: "/dev/ttyUSB0" } as const;
const BAUD_RATE_INPUT_PROPS = { endContent: "baud" } as const;

const Properties = (): ReactElement => (
  <>
    <Flex.Box x>
      <Form.TextField
        path="properties.port"
        label="Port"
        grow
        inputProps={PORT_INPUT_PROPS}
      />
      <Form.NumericField
        path="properties.baudRate"
        label="Baud rate"
        inputProps={BAUD_RATE_INPUT_PROPS}
      />
    </Flex.Box>
    <Flex.Box x>
      <DataBitsField path="properties" />
      <ParityField path="properties" />
      <StopBitsField path="properties" />
      <FlowControlField path="properties" />
    </Flex.Box>
    <Form.SwitchField path="properties.rs485" label="RS-485" />
  </>
);

export const {
  MAKE,
  SCHEMAS,
  use,
  useResult,
  useFromConfig,
  useConnectModal,
  Select,
  COMMANDS,
} = Bus.createDevice({
  make: "Serial",
  noun: "serial device",
  icon: <Icon.Connect />,
  properties: serial.propertiesZ,
  integration: "serial",
  Properties,
  getModel: () => "Serial port",
  required: { port: "Port is required" },
  getLocation: ({ port }) => port,
});
