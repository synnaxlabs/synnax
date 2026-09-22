// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/mqtt/device/Connect.css";

import { type device, type rack, status, TimeSpan } from "@synnaxlabs/client";
import {
  Button,
  Component,
  Device as PDevice,
  Divider,
  Flex,
  type Flux,
  Form,
  Icon,
  Input,
  Nav,
  Rack,
  Status,
  Text,
} from "@synnaxlabs/pluto";
import { caseconv } from "@synnaxlabs/x";
import { useCallback } from "react";

import { retrieveScanTask } from "@/feature/mqtt/device/retrieveScanTask";
import {
  type Device,
  MAKE,
  type Properties,
  SCHEMAS,
  ZERO_PROPERTIES,
} from "@/feature/mqtt/device/types";
import { TEST_CONNECTION_COMMAND_TYPE } from "@/feature/mqtt/task/types";
import { CSS } from "@/platform/css";
import { type Device as PlatformDevice } from "@/platform/device";
import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";

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

const useForm = PDevice.createForm(SCHEMAS);

const TEST_CONNECTION_TIMEOUT = TimeSpan.seconds(10);

const SPARKPLUG_PATH = "properties.sparkplug";

// The form schema holds the properties as an open record, so it does not check them.
const beforeValidate = ({
  get,
  setStatus,
}: Flux.BeforeValidateParams<PDevice.RetrieveQuery, typeof PDevice.formSchema>) => {
  const { sparkplug } = SCHEMAS.properties.shape;
  const result = sparkplug.safeParse(get(SPARKPLUG_PATH).value);
  result.error?.issues.forEach(({ path, message }) =>
    setStatus(`${SPARKPLUG_PATH}.${path.join(".")}`, { variant: "error", message }),
  );
  return result.success;
};

const beforeSave = async ({
  client,
  get,
  set,
}: Flux.FormBeforeSaveParams<PDevice.RetrieveQuery, typeof PDevice.formSchema>) => {
  const scanTask = await retrieveScanTask(client, get<rack.Key>("rack").value);
  // Command arguments skip case conversion on the wire; the driver reads snake case.
  const properties = caseconv.camelToSnake(get<Properties>("properties").value, {
    schema: SCHEMAS.properties,
  });
  const state = await scanTask.executeCommandSync({
    type: TEST_CONNECTION_COMMAND_TYPE,
    timeout: TEST_CONNECTION_TIMEOUT,
    args: { location: get<string>("location").value, properties },
  });
  if (state.variant === "error") throw new Error(state.message);
  const devStatus: device.Status = status.create<typeof device.statusDetailsZ>({
    message: "Broker connected",
    variant: "success",
    details: {
      rack: get<rack.Key>("rack").value,
      device: get<device.Key>("key").value,
    },
  });
  set("status", devStatus, { markTouched: false });
  return true;
};

export const useConnectModal = Modals.create<PlatformDevice.ConnectParams>(
  ({ deviceKey, close }) => {
    const {
      form,
      save,
      status: stat,
      variant,
    } = useForm({
      query: deviceKey == null ? null : { key: deviceKey },
      initialValues: INITIAL_VALUES,
      beforeValidate,
      beforeSave,
      afterSave: useCallback(() => close(), [close]),
    });

    const secure = Form.useFieldValue<boolean, boolean, typeof PDevice.formSchema>(
      "properties.secure",
      { ctx: form },
    );

    return (
      <Modals.Frame className={CSS.B("mqtt-connect")}>
        <Modals.Header icon={<Icon.Logo.MQTT />}>Broker.Connect</Modals.Header>
        <Flex.Box className={CSS.B("content")} grow gap="large">
          <Form.Form<typeof PDevice.formSchema> {...form}>
            <Flex.Box gap="small">
              <Form.TextField path="name" inputProps={NAME_INPUT_PROPS} />
              <Form.Field<rack.Key> path="rack" label="Connect from" required>
                {selectRackRenderProp}
              </Form.Field>
              <Flex.Box x align="end">
                <Form.TextField
                  grow
                  path="location"
                  label="Host"
                  inputProps={HOST_INPUT_PROPS}
                />
                <Form.NumericField
                  path="properties.port"
                  label="Port"
                  inputProps={secure ? SECURE_PORT_INPUT_PROPS : PORT_INPUT_PROPS}
                />
                <Form.SwitchField path="properties.secure" label="TLS" />
              </Flex.Box>
              <Divider.Divider x />
            </Flex.Box>
            {secure && (
              <Flex.Box gap="small">
                <Form.SwitchField
                  path="properties.verificationSkipped"
                  label="Skip certificate verification"
                  align="start"
                />
                <Text.Text level="small" color={9}>
                  Certificate and key files are paths on the Core host. An empty CA file
                  uses the system roots.
                </Text.Text>
                <Form.TextField
                  path="properties.caFile"
                  label="CA file"
                  inputProps={CA_FILE_INPUT_PROPS}
                />
                <Flex.Box x>
                  <Form.TextField
                    grow
                    path="properties.certFile"
                    label="Client certificate"
                    inputProps={CERT_FILE_INPUT_PROPS}
                  />
                  <Form.TextField
                    grow
                    path="properties.keyFile"
                    label="Client key"
                    inputProps={KEY_FILE_INPUT_PROPS}
                  />
                </Flex.Box>
                <Divider.Divider x />
              </Flex.Box>
            )}
            <Flex.Box gap="small">
              <Flex.Box x justify="between">
                <Form.TextField grow path="properties.username" label="Username" />
                <Form.TextField
                  grow
                  path="properties.password"
                  label="Password"
                  inputProps={PASSWORD_INPUT_PROPS}
                />
              </Flex.Box>
              <Flex.Box x align="end">
                <Form.TextField
                  grow
                  path="properties.clientId"
                  label="Client ID"
                  inputProps={CLIENT_ID_INPUT_PROPS}
                />
                <Form.NumericField
                  path="properties.keepAlive"
                  label="Keep alive"
                  inputProps={KEEP_ALIVE_INPUT_PROPS}
                />
              </Flex.Box>
              <Divider.Divider x />
            </Flex.Box>
            <Flex.Box gap="small">
              <Text.Text level="small" weight={500} color={9}>
                Sparkplug B
              </Text.Text>
              <Text.Text level="small" color={9}>
                Leave the host ID empty for a passive host. Separate groups with commas,
                or leave them empty to browse all groups.
              </Text.Text>
              <Flex.Box x>
                <Form.TextField
                  grow
                  path="properties.sparkplug.hostId"
                  label="Host ID"
                  inputProps={HOST_ID_INPUT_PROPS}
                />
                <Form.Field<string[]>
                  grow
                  path="properties.sparkplug.groups"
                  label="Groups"
                >
                  {groupsRenderProp}
                </Form.Field>
              </Flex.Box>
            </Flex.Box>
          </Form.Form>
        </Flex.Box>
        <Modals.Footer>
          <Nav.Bar.Start gap="small">
            {variant == "success" ? (
              <Triggers.SaveHelpText action="Connect" noBar />
            ) : (
              <Status.Summary variant={variant} message={stat.description} />
            )}
          </Nav.Bar.Start>
          <Nav.Bar.End>
            <Button.Button
              status={status.keepVariants(variant, "loading")}
              onClick={() => save()}
              variant="filled"
              trigger={Triggers.SAVE}
            >
              Connect
            </Button.Button>
          </Nav.Bar.End>
        </Modals.Footer>
      </Modals.Frame>
    );
  },
);

const INITIAL_RACK_QUERY: rack.RetrieveParams = { integration: MAKE };

const selectRackRenderProp = Component.renderProp(
  (props: Pick<Rack.SelectSingleProps, "value" | "onChange">) => (
    <Rack.SelectSingle {...props} initialQuery={INITIAL_RACK_QUERY} />
  ),
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

const HOST_ID_INPUT_PROPS = { placeholder: "Passive host" } as const;

const NAME_INPUT_PROPS = {
  level: "h2",
  variant: "text",
  placeholder: "MQTT broker",
} as const;

const HOST_INPUT_PROPS = {
  autoFocus: true,
  placeholder: "broker.example.com",
} as const;

// A port of 0 selects the default, which the placeholder names.
const PORT_INPUT_PROPS = {
  bounds: { lower: 0, upper: 65535 },
  emptyValue: 0,
  placeholder: "1883",
  style: { width: "23rem" },
} as const;

const SECURE_PORT_INPUT_PROPS = { ...PORT_INPUT_PROPS, placeholder: "8883" } as const;

const CA_FILE_INPUT_PROPS = { placeholder: "/etc/ssl/certs/ca.pem" } as const;

const CERT_FILE_INPUT_PROPS = { placeholder: "/etc/ssl/certs/client.pem" } as const;

const KEY_FILE_INPUT_PROPS = { placeholder: "/etc/ssl/private/client.key" } as const;

const PASSWORD_INPUT_PROPS = { type: "password" } as const;

const CLIENT_ID_INPUT_PROPS = {
  placeholder: "Derived from the device when empty",
} as const;

const KEEP_ALIVE_INPUT_PROPS = {
  bounds: { lower: 0, upper: Infinity },
  emptyValue: 0,
  endContent: "s",
  placeholder: "30",
  style: { width: "23rem" },
} as const;
