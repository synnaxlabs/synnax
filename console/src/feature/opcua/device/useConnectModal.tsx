// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/opcua/device/Connect.css";

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
  Nav,
  Rack,
  Status,
} from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { retrieveScanTask } from "@/feature/opcua/device/retrieveScanTask";
import { SelectSecurityMode } from "@/feature/opcua/device/SelectSecurityMode";
import { SelectSecurityPolicy } from "@/feature/opcua/device/SelectSecurityPolicy";
import {
  type Device,
  NO_SECURITY_MODE,
  SCHEMAS,
  type SecurityMode,
  type SecurityPolicy,
  ZERO_PROPERTIES,
} from "@/feature/opcua/device/types";
import { TEST_CONNECTION_COMMAND_TYPE } from "@/feature/opcua/task/types";
import { CSS } from "@/platform/css";
import { type Device as PlatformDevice } from "@/platform/device";
import { FS } from "@/platform/fs";
import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";

const useForm = PDevice.createForm(SCHEMAS);

const TEST_CONNECTION_TIMEOUT = TimeSpan.seconds(10);

const INITIAL_VALUES: Device = {
  key: "",
  name: "OPC UA server",
  make: "opc",
  model: "OPC UA server",
  location: "",
  properties: ZERO_PROPERTIES,
  rack: 0,
  configured: true,
};

const beforeValidate = ({
  get,
  set,
}: Flux.BeforeValidateParams<PDevice.RetrieveQuery, typeof PDevice.formSchema>) =>
  set("location", get("properties.connection.endpoint").value);

const beforeSave = async ({
  client,
  get,
  set,
}: Flux.FormBeforeSaveParams<PDevice.RetrieveQuery, typeof PDevice.formSchema>) => {
  const scanTask = await retrieveScanTask(client, get<rack.Key>("rack").value);
  const scanStatus = await scanTask.executeCommandSync({
    type: TEST_CONNECTION_COMMAND_TYPE,
    timeout: TEST_CONNECTION_TIMEOUT,
    args: { connection: get("properties.connection").value },
  });
  if (scanStatus.variant === "error") throw new Error(scanStatus.message);
  // Since we just scanned successfully, we create a default healthy status for the
  // device that can then be overwritten by the scanner if we lose connection.
  const devStatus: device.Status = status.create<typeof device.statusDetailsZ>({
    message: "Server connected",
    variant: "success",
    details: {
      rack: get<rack.Key>("rack").value,
      device: get<device.Key>("key").value,
    },
  });
  set("status", devStatus, { notifyOnChange: false, markTouched: false });
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

    const hasSecurity =
      Form.useFieldValue<SecurityMode, SecurityMode, typeof PDevice.formSchema>(
        "properties.connection.securityMode",
        { ctx: form },
      ) != NO_SECURITY_MODE;
    return (
      <Modals.Frame className={CSS.B("opc-connect")}>
        <Modals.Header icon={<Icon.Logo.OPCUA />}>Server.Connect</Modals.Header>
        <Modals.Body gap="small">
          <Form.Form<typeof PDevice.formSchema> {...form}>
            <Form.TextField inputProps={NAME_INPUT_PROPS} path="name" />
            <Form.Field<rack.Key> path="rack" label="Connect from" required>
              {selectRackRenderProp}
            </Form.Field>
            <Form.TextField
              path="properties.connection.endpoint"
              inputProps={ENDPOINT_INPUT_PROPS}
            />
            <Divider.Divider x padded="bottom" />
            <Flex.Box x justify="between">
              <Form.TextField
                grow
                path="properties.connection.username"
                inputProps={USERNAME_INPUT_PROPS}
              />
              <Form.TextField
                grow
                path="properties.connection.password"
                inputProps={PASSWORD_INPUT_PROPS}
              />
              <Form.Field<SecurityMode>
                label="Security mode"
                path="properties.connection.securityMode"
              >
                {({ value, onChange }) => (
                  <SelectSecurityMode value={value} onChange={onChange} />
                )}
              </Form.Field>
            </Flex.Box>
            <Divider.Divider x padded="bottom" />
            <Form.Field<SecurityPolicy>
              grow={!hasSecurity}
              path="properties.connection.securityPolicy"
              label="Security policy"
            >
              {({ value, onChange }) => (
                <SelectSecurityPolicy value={value} onChange={onChange} />
              )}
            </Form.Field>
            {hasSecurity && (
              <>
                <Form.Field<string>
                  label="Client certificate"
                  path="properties.connection.clientCertificate"
                >
                  {clientCertField}
                </Form.Field>
                <Form.Field<string>
                  label="Client private key"
                  path="properties.connection.clientPrivateKey"
                >
                  {clientPrivateKeyField}
                </Form.Field>
                <Form.Field<string>
                  grow
                  label="Server certificate"
                  path="properties.connection.serverCertificate"
                >
                  {serverCertField}
                </Form.Field>
              </>
            )}
          </Form.Form>
        </Modals.Body>
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

const clientCertField = Component.renderProp(({ value, onChange }) => (
  <FS.InputPath value={value} onChange={onChange} title="Select a client certificate" />
));

const clientPrivateKeyField = Component.renderProp(({ value, onChange }) => (
  <FS.InputPath value={value} onChange={onChange} title="Select a client private key" />
));

const serverCertField = Component.renderProp(({ value, onChange }) => (
  <FS.InputPath value={value} onChange={onChange} title="Select a server certificate" />
));

const INITIAL_RACK_QUERY: rack.RetrieveParams = { integration: "opc" };

const selectRackRenderProp = Component.renderProp(
  (props: Pick<Rack.SelectSingleProps, "value" | "onChange">) => (
    <Rack.SelectSingle {...props} initialQuery={INITIAL_RACK_QUERY} />
  ),
);

const NAME_INPUT_PROPS = {
  level: "h2",
  variant: "text",
  placeholder: "OPC UA server",
} as const;

const ENDPOINT_INPUT_PROPS = {
  placeholder: "opc.tcp://localhost:4840",
  autoFocus: true,
} as const;

const USERNAME_INPUT_PROPS = { placeholder: "admin" } as const;

const PASSWORD_INPUT_PROPS = { type: "password" } as const;
