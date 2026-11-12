// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, status, TimeSpan } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Text } from "@synnaxlabs/lyra/text";
import { Device as PDevice, type Flux } from "@synnaxlabs/pluto";
import { caseconv, deep } from "@synnaxlabs/x";

import { retrieveScanTask } from "@/feature/mqtt/device/retrieveScanTask";
import {
  type Device,
  INTEGRATION_KEY,
  MAKE,
  SCHEMAS,
  ZERO_PROPERTIES,
} from "@/feature/mqtt/device/types";
import { TEST_CONNECTION_COMMAND_TYPE } from "@/feature/mqtt/task/types";
import { Analytics } from "@/platform/analytics";
import { Device as PlatformDevice } from "@/platform/device";

const path = deep.path<Device>();
const { properties } = path;

const INITIAL_VALUES: Device = {
  key: "",
  name: "MQTT broker",
  make: MAKE,
  model: "MQTT broker",
  location: "",
  properties: ZERO_PROPERTIES,
  rack: 0,
  configured: true,
};

const useForm = PDevice.createForm(SCHEMAS, INITIAL_VALUES);

const TEST_CONNECTION_TIMEOUT = TimeSpan.seconds(10);

const beforeSave = async ({
  client,
  get,
  set,
}: Flux.FormBeforeSaveParams<
  PDevice.RetrieveQuery,
  PDevice.FormSchema<typeof SCHEMAS>
>) => {
  const scanTask = await retrieveScanTask(client, get(path.rack).value);
  // Command arguments skip case conversion on the wire; the Core reads snake case.
  const props = caseconv.camelToSnake(get(properties).value, {
    schema: SCHEMAS.properties,
  });
  const state = await scanTask.executeCommandSync({
    type: TEST_CONNECTION_COMMAND_TYPE,
    timeout: TEST_CONNECTION_TIMEOUT,
    args: {
      location: get(path.location).value,
      properties: props,
      device: get(path.key).value,
    },
  });
  if (state.variant === "error") throw status.toError(state);
  const devStatus: device.Status = status.create<typeof device.statusDetailsZ>({
    name: get(path.name).value,
    message: "Broker connected",
    variant: "success",
    details: {
      rack: get(path.rack).value,
      connected: true,
    },
  });
  set(path.status, devStatus, { markTouched: false });
  return true;
};

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
    beforeSave,
    afterSave: ({ get }) => {
      if (deviceKey == null) capture("device_connected", { integration: "mqtt" });
      close(get(path.key).value);
    },
  });
  const rackKey = Form.useFieldValue(path.rack, { ctx: form });
  const reason = PlatformDevice.useBlockedReason({
    integration: INTEGRATION_KEY,
    name: "MQTT",
    rack: rackKey,
  });
  const secure = Form.useFieldValue(properties.secure, { ctx: form });

  return (
    <PlatformDevice.ConnectFrame>
      <Modal.Header icon={<Icon.Logo.MQTT />}>{HEADER}</Modal.Header>
      <Modal.Body gap="large">
        <Form.Form<PDevice.FormSchema<typeof SCHEMAS>> {...form}>
          <Form.TextField
            path={path.name}
            showLabel={false}
            inputProps={NAME_INPUT_PROPS}
          />
          <Form.Sections>
            <Form.Section title="Connection">
              <Flex.Box x>
                <Form.TextField
                  path={path.location}
                  label="Host"
                  padHelpText={false}
                  inputProps={HOST_INPUT_PROPS}
                />
                <Form.Field path={properties.port} label="Port" padHelpText={false}>
                  {secure ? renderSecurePort : renderPort}
                </Form.Field>
              </Flex.Box>
              <Flex.Box x>
                <Form.TextField
                  path={properties.clientId}
                  label="Client ID"
                  padHelpText={false}
                  inputProps={CLIENT_ID_INPUT_PROPS}
                />
                <Form.TimeSpanField
                  path={properties.keepAlive}
                  label="Keep alive"
                  unit={TimeSpan.SECOND}
                  emptiable
                  padHelpText={false}
                  inputProps={KEEP_ALIVE_INPUT_PROPS}
                />
              </Flex.Box>
              <PlatformDevice.RackField integration={INTEGRATION_KEY} />
            </Form.Section>
            <Form.Section title="Authentication">
              <Flex.Box x>
                <Form.TextField
                  path={properties.username}
                  label="Username"
                  padHelpText={false}
                  inputProps={USERNAME_INPUT_PROPS}
                />
                <Form.TextField
                  path={properties.password}
                  label="Password"
                  padHelpText={false}
                  inputProps={PASSWORD_INPUT_PROPS}
                />
              </Flex.Box>
            </Form.Section>
            <Form.Section title="Security">
              <Form.CheckboxField
                path={properties.secure}
                label="TLS"
                padHelpText={false}
              />
              {secure && (
                <>
                  <Form.CheckboxField
                    path={properties.verificationSkipped}
                    label="Skip verification"
                    padHelpText={false}
                  />
                  <Form.TextField
                    path={properties.caFile}
                    label="CA file"
                    padHelpText={false}
                    inputProps={CA_FILE_INPUT_PROPS}
                  />
                  <Form.TextField
                    path={properties.certFile}
                    label="Client certificate"
                    padHelpText={false}
                    inputProps={CERT_FILE_INPUT_PROPS}
                  />
                  <Form.TextField
                    path={properties.keyFile}
                    label="Client key"
                    padHelpText={false}
                    inputProps={KEY_FILE_INPUT_PROPS}
                  />
                  <Text.Text level="small" color={9}>
                    Certificate and key files are paths on the Core host. An empty CA
                    file uses the system roots.
                  </Text.Text>
                </>
              )}
            </Form.Section>
            <Form.Section title="Sparkplug B">
              <Flex.Box x>
                <Form.TextField
                  path={properties.sparkplug.hostId}
                  label="Host ID"
                  padHelpText={false}
                  inputProps={HOST_ID_INPUT_PROPS}
                />
                <Form.Field
                  path={properties.sparkplug.groups}
                  label="Groups"
                  padHelpText={false}
                >
                  {groupsRenderProp}
                </Form.Field>
              </Flex.Box>
              <Text.Text level="small" color={9}>
                Leave the host ID empty for a passive host. Separate groups with commas,
                or leave them empty to browse all groups.
              </Text.Text>
            </Form.Section>
          </Form.Sections>
        </Form.Form>
      </Modal.Body>
      <PlatformDevice.ConnectFooter
        reason={reason}
        variant={variant}
        message={stat.description}
        onConnect={() => save()}
        onBack={onBack}
      />
    </PlatformDevice.ConnectFrame>
  );
};

export const useConnectModal = PlatformDevice.createConnectModal(
  Connect,
  INTEGRATION_KEY,
);

const groupsRenderProp = Component.renderProp(
  ({ value, onChange, ...rest }: Input.Control<string[]>) => (
    <Input.Text
      {...rest}
      value={value.join(", ")}
      onChange={(text) => onChange(parseGroups(text))}
      onlyChangeOnBlur
      placeholder="All groups"
    />
  ),
);

const parseGroups = (text: string): string[] =>
  text
    .split(",")
    .map((group) => group.trim())
    .filter((group) => group !== "");

const HEADER = ["Device", "Connect", "MQTT broker"];

const NAME_INPUT_PROPS = {
  level: "h2",
  variant: "text",
  placeholder: "MQTT broker",
  "aria-label": "Name",
} as const;

const HOST_INPUT_PROPS = {
  autoFocus: true,
  placeholder: "broker.example.com",
} as const;

// An empty port selects the default, which the placeholder names.
const PORT_INPUT_PROPS = { placeholder: "1883" } as const;

const SECURE_PORT_INPUT_PROPS = { ...PORT_INPUT_PROPS, placeholder: "8883" } as const;

const CLIENT_ID_INPUT_PROPS = {
  placeholder: "Derived from the device when empty",
} as const;

const KEEP_ALIVE_INPUT_PROPS = { placeholder: "30s" } as const;

const USERNAME_INPUT_PROPS = { placeholder: "Anonymous" } as const;

const PASSWORD_INPUT_PROPS = { type: "password" } as const;

const CA_FILE_INPUT_PROPS = { placeholder: "/etc/ssl/certs/ca.pem" } as const;

const CERT_FILE_INPUT_PROPS = { placeholder: "/etc/ssl/certs/client.pem" } as const;

const KEY_FILE_INPUT_PROPS = { placeholder: "/etc/ssl/private/client.key" } as const;

const HOST_ID_INPUT_PROPS = { placeholder: "Passive host" } as const;

const createOptionalNumericRenderProp = (
  props: Omit<Input.NumericProps, "value" | "onChange" | "onClear">,
) =>
  Component.renderProp((p: Input.Control<number | undefined>) => (
    <Input.Numeric {...p} {...props} onClear={() => p.onChange(undefined)} />
  ));

const renderPort = createOptionalNumericRenderProp(PORT_INPUT_PROPS);

const renderSecurePort = createOptionalNumericRenderProp(SECURE_PORT_INPUT_PROPS);
