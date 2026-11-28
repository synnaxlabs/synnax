// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { serial } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Select } from "@synnaxlabs/lyra/select";
import { Device as PDevice, type Flux } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";

import {
  type Device,
  INTEGRATION_KEY,
  MAKE,
  SCHEMAS,
  ZERO_PROPERTIES,
} from "@/feature/serial/device/types";
import { Analytics } from "@/platform/analytics";
import { Device as PlatformDevice } from "@/platform/device";
import { Rack } from "@/platform/rack";

const path = deep.path<Device>();

const INITIAL_VALUES: Device = {
  key: "",
  name: "",
  make: MAKE,
  model: MAKE,
  location: "",
  properties: ZERO_PROPERTIES,
  rack: 0,
  configured: true,
};

const useForm = PDevice.createForm(SCHEMAS, INITIAL_VALUES);

const beforeValidate = ({
  get,
  set,
}: Flux.BeforeValidateParams<
  PDevice.RetrieveQuery,
  PDevice.FormSchema<typeof SCHEMAS>
>) => set(path.location, get(path.properties.port).value);

const PARITY_NAMES: Record<serial.Parity, string> = {
  none: "None",
  even: "Even",
  odd: "Odd",
  mark: "Mark",
  space: "Space",
};

const FLOW_CONTROL_NAMES: Record<serial.FlowControl, string> = {
  none: "None",
  hardware: "RTS/CTS",
  software: "XON/XOFF",
};

const ParityField = Form.buildSelectField<serial.Parity>({
  fieldProps: { label: "Parity", padHelpText: false },
  inputProps: {
    resourceName: "parity",
    allowNone: false,
    children: serial.PARITYS.map((parity) => (
      <Select.Item<serial.Parity> key={parity} itemKey={parity}>
        {PARITY_NAMES[parity]}
      </Select.Item>
    )),
  },
});

const StopBitsField = Form.buildSelectField<serial.StopBits>({
  fieldProps: { label: "Stop bits", padHelpText: false },
  inputProps: {
    resourceName: "stop bits",
    allowNone: false,
    children: serial.STOP_BITS.map((bits) => (
      <Select.Item<serial.StopBits> key={bits} itemKey={bits}>
        {bits}
      </Select.Item>
    )),
  },
});

const FlowControlField = Form.buildSelectField<serial.FlowControl>({
  fieldProps: { label: "Flow control", padHelpText: false },
  inputProps: {
    resourceName: "flow control",
    allowNone: false,
    children: serial.FLOW_CONTROLS.map((flow) => (
      <Select.Item<serial.FlowControl> key={flow} itemKey={flow}>
        {FLOW_CONTROL_NAMES[flow]}
      </Select.Item>
    )),
  },
});

export const Connect = ({
  deviceKey,
  rack: defaultRack,
  close,
  onBack,
}: PlatformDevice.ConnectProps) => {
  const { capture } = Analytics.use();
  const {
    form,
    save,
    status: stat,
    variant,
  } = useForm({
    query: deviceKey == null ? null : { key: deviceKey },
    initialValues: { ...INITIAL_VALUES, rack: defaultRack ?? 0 },
    beforeValidate,
    afterSave: ({ get }) => {
      if (deviceKey == null)
        capture("device_connected", { integration: INTEGRATION_KEY });
      close(get(path.key).value);
    },
  });
  const rackKey = Form.useFieldValue(path.rack, { ctx: form });
  const reason = Rack.useBlockedReason({
    integration: INTEGRATION_KEY,
    name: "Serial",
    rack: rackKey,
  });
  return (
    <PlatformDevice.ConnectFrame>
      <Modal.Header icon={<Icon.Serial />}>{HEADER}</Modal.Header>
      <Modal.Body gap="large" justify="start">
        <Form.Form<PDevice.FormSchema<typeof SCHEMAS>> {...form}>
          <Form.TextField
            path={path.name}
            showLabel={false}
            inputProps={NAME_INPUT_PROPS}
          />
          <Form.Sections>
            <Form.Section title="Port">
              <Form.TextField
                path={path.properties.port}
                label="Port"
                padHelpText={false}
                required
                inputProps={PORT_INPUT_PROPS}
              />
              <Form.NumericField
                path={path.properties.baudRate}
                label="Baud rate"
                padHelpText={false}
              />
              <PlatformDevice.RackField integration={INTEGRATION_KEY} />
            </Form.Section>
            <Form.Section title="Line">
              <Form.NumericField
                path={path.properties.dataBits}
                label="Data bits"
                padHelpText={false}
              />
              <ParityField path={path.properties.parity} />
              <StopBitsField path={path.properties.stopBits} />
              <FlowControlField path={path.properties.flowControl} />
            </Form.Section>
          </Form.Sections>
        </Form.Form>
      </Modal.Body>
      <PlatformDevice.ConnectFooter
        reason={reason}
        variant={variant}
        message={stat.description}
        onConnect={() => save()}
        onCancel={() => close()}
        onBack={onBack}
      />
    </PlatformDevice.ConnectFrame>
  );
};

export const useConnectModal = PlatformDevice.createConnectModal(
  Connect,
  INTEGRATION_KEY,
);

const HEADER = ["Device", "Connect", "Serial device"];

const NAME_INPUT_PROPS = {
  level: "h2",
  variant: "text",
  placeholder: "Device name",
  "aria-label": "Name",
  autoFocus: true,
} as const;

const PORT_INPUT_PROPS = { placeholder: "/dev/ttyUSB0" } as const;
