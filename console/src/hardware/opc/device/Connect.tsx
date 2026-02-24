// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/opc/device/Connect.css";

import { type device, type rack, TimeSpan } from "@synnaxlabs/client";
import {
  Button,
  Device as PDevice,
  Divider,
  Flex,
  type Flux,
  Form,
  Nav,
  Rack,
  Status,
} from "@synnaxlabs/pluto";
import { status } from "@synnaxlabs/x";
import { useCallback } from "react";

import { CSS } from "@/css";
import { FS } from "@/fs";
import { retrieveScanTask } from "@/hardware/opc/device/retrieveScanTask";
import { SelectSecurityMode } from "@/hardware/opc/device/SelectSecurityMode";
import { SelectSecurityPolicy } from "@/hardware/opc/device/SelectSecurityPolicy";
import {
  type Device,
  NO_SECURITY_MODE,
  SCHEMAS,
  type SecurityMode,
  type SecurityPolicy,
  ZERO_PROPERTIES,
} from "@/hardware/opc/device/types";
import { TEST_CONNECTION_COMMAND_TYPE } from "@/hardware/opc/task/types";
import { type Layout } from "@/layout";
import { Modals } from "@/modals";
import { Triggers } from "@/triggers";

export const CONNECT_LAYOUT_TYPE = "configureOPCServer";

export const CONNECT_LAYOUT: Layout.BaseState = {
  key: CONNECT_LAYOUT_TYPE,
  type: CONNECT_LAYOUT_TYPE,
  name: "Server.Connect",
  icon: "Logo.OPC",
  location: "modal",
  window: { resizable: false, size: { height: 720, width: 915 }, navTop: true },
};

const useForm = PDevice.createForm(SCHEMAS);

const INITIAL_VALUES: Device = {
  key: "",
  name: "OPC UA Server",
  make: "opc",
  model: "OPC UA Server",
  location: "",
  properties: ZERO_PROPERTIES,
  rack: 0,
  configured: true,
};

const beforeValidate = ({
  get,
  set,
}: Flux.BeforeValidateArgs<
  PDevice.RetrieveQuery,
  typeof PDevice.formSchema,
  PDevice.FluxSubStore
>) => set("location", get("properties.connection.endpoint").value);

const beforeSave = async ({
  client,
  get,
  store,
  set,
}: Flux.FormBeforeSaveParams<
  PDevice.RetrieveQuery,
  typeof PDevice.formSchema,
  PDevice.FluxSubStore
>) => {
  const scanTask = await retrieveScanTask(client, store, get<rack.Key>("rack").value);
  const scanStatus = await scanTask.executeCommandSync({
    type: TEST_CONNECTION_COMMAND_TYPE,
    timeout: TimeSpan.seconds(10),
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

export const Connect: Layout.Renderer = ({ layoutKey, onClose }) => {
  const {
    form,
    save,
    status: stat,
    variant,
  } = useForm({
    query: { key: layoutKey === CONNECT_LAYOUT_TYPE ? "" : layoutKey },
    initialValues: INITIAL_VALUES,
    beforeValidate,
    beforeSave,
    afterSave: useCallback(() => onClose(), [onClose]),
  });

  const hasSecurity =
    Form.useFieldValue<SecurityMode, SecurityMode, typeof PDevice.formSchema>(
      "properties.connection.securityMode",
      { ctx: form },
    ) != NO_SECURITY_MODE;
  return (
    <Flex.Box align="start" className={CSS.B("opc-connect")} justify="center">
      <Flex.Box className={CSS.B("content")} grow gap="small">
        <Form.Form<typeof PDevice.formSchema> {...form}>
          <Form.TextField inputProps={NAME_INPUT_PROPS} path="name" />
          <Form.Field<rack.Key> path="rack" label="Connect From" required>
            {({ value, onChange }) => (
              <Rack.SelectSingle value={value} onChange={onChange} allowNone={false} />
            )}
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
              label="Security Mode"
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
            label="Security Policy"
          >
            {({ value, onChange }) => (
              <SelectSecurityPolicy value={value} onChange={onChange} />
            )}
          </Form.Field>
          {hasSecurity && (
            <>
              <Form.Field<string>
                label="Client Certificate"
                path="properties.connection.clientCertificate"
              >
                {FS.InputFilePath}
              </Form.Field>
              <Form.Field<string>
                label="Client Private Key"
                path="properties.connection.clientPrivateKey"
              >
                {FS.InputFilePath}
              </Form.Field>
              <Form.Field<string>
                grow
                label="Server Certificate"
                path="properties.connection.serverCertificate"
              >
                {FS.InputFilePath}
              </Form.Field>
            </>
          )}
        </Form.Form>
      </Flex.Box>
      <Modals.BottomNavBar>
        <Nav.Bar.Start gap="small">
          {variant == "success" ? (
            <Triggers.SaveHelpText action="Test Connection" noBar />
          ) : (
            <Status.Summary variant={variant} message={stat.description} />
          )}
        </Nav.Bar.Start>
        <Nav.Bar.End>
          <Button.Button
            status={status.keepVariants(variant, "loading")}
            onClick={() => save()}
            variant="filled"
          >
            Connect
          </Button.Button>
        </Nav.Bar.End>
      </Modals.BottomNavBar>
    </Flex.Box>
  );
};

const NAME_INPUT_PROPS = {
  level: "h2",
  variant: "text",
  placeholder: "OPC UA Server",
} as const;

const ENDPOINT_INPUT_PROPS = {
  placeholder: "opc.tcp://localhost:4840",
  autoFocus: true,
} as const;

const USERNAME_INPUT_PROPS = { placeholder: "admin" } as const;

const PASSWORD_INPUT_PROPS = { type: "password" } as const;
