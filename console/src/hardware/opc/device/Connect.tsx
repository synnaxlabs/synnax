// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/opc/device/Connect.css";

import { DisconnectedError, rack, TimeSpan, UnexpectedError } from "@synnaxlabs/client";
import {
  Button,
  Divider,
  Flex,
  Form,
  Input,
  Nav,
  Rack,
  Status,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { deep, uuid } from "@synnaxlabs/x";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { z } from "zod";

import { CSS } from "@/css";
import { FS } from "@/fs";
import { Common } from "@/hardware/common";
import { SelectSecurityMode } from "@/hardware/opc/device/SelectSecurityMode";
import { SelectSecurityPolicy } from "@/hardware/opc/device/SelectSecurityPolicy";
import {
  type ConnectionConfig,
  connectionConfigZ,
  MAKE,
  migrateProperties,
  NO_SECURITY_MODE,
  type Properties,
  type SecurityMode,
  type SecurityPolicy,
  ZERO_CONNECTION_CONFIG,
  ZERO_PROPERTIES,
} from "@/hardware/opc/device/types";
import {
  SCAN_SCHEMAS,
  SCAN_TYPE,
  TEST_CONNECTION_COMMAND_TYPE,
  type TestConnectionStatus,
} from "@/hardware/opc/task/types";
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

const formSchema = z.object({
  name: Common.Device.nameZ,
  rack: rack.keyZ.refine((k) => k > 0, "Must select a location to connect from"),
  connection: connectionConfigZ,
});
interface FormSchema extends z.infer<typeof formSchema> {}

interface InternalProps extends Pick<Layout.RendererProps, "layoutKey" | "onClose"> {
  initialValues: FormSchema;
  properties?: Properties;
}

const Internal = ({ initialValues, layoutKey, onClose, properties }: InternalProps) => {
  const client = Synnax.use();
  const [connectionState, setConnectionState] = useState<TestConnectionStatus>();
  const handleError = Status.useErrorHandler();
  const methods = Form.use({ values: initialValues, schema: formSchema });
  const testConnectionMutation = useMutation({
    onError: (e) => handleError(e, "Failed to test connection"),
    mutationFn: async () => {
      if (client == null) throw new DisconnectedError();
      if (!methods.validate()) return;
      const rack = await client.hardware.racks.retrieve({
        key: methods.get<rack.Key>("rack").value,
      });
      const scanTasks = await client.hardware.tasks.retrieve({
        types: [SCAN_TYPE],
        rack: rack.key,
        schemas: SCAN_SCHEMAS,
      });
      if (scanTasks.length === 0)
        throw new UnexpectedError(`No scan task found for driver ${rack.name}`);
      const task = scanTasks[0];
      const state = await task.executeCommandSync(
        TEST_CONNECTION_COMMAND_TYPE,
        TimeSpan.seconds(10),
        { connection: methods.get("connection").value },
      );
      setConnectionState(state);
    },
  });
  const connectMutation = useMutation({
    onError: (e) => handleError(e, "Failed to connect to OPC UA Server"),
    mutationFn: async () => {
      if (client == null) throw new DisconnectedError();
      if (!methods.validate()) return;
      await testConnectionMutation.mutateAsync();
      if (connectionState?.variant !== "success")
        throw new Error("Connection test failed");
      const rack = await client.hardware.racks.retrieve({
        key: methods.get<rack.Key>("rack").value,
      });
      const key = layoutKey === CONNECT_LAYOUT_TYPE ? uuid.create() : layoutKey;
      await client.hardware.devices.create<Properties>({
        key,
        name: methods.get<string>("name").value,
        model: MAKE,
        make: MAKE,
        rack: rack.key,
        location: methods.get<string>("connection.endpoint").value,
        properties: {
          ...ZERO_PROPERTIES,
          ...properties,
          connection: methods.get<ConnectionConfig>("connection").value,
        },
        configured: true,
      });
      onClose();
    },
  });
  const hasSecurity =
    Form.useFieldValue<SecurityMode, SecurityMode, typeof formSchema>(
      "connection.securityMode",
      { ctx: methods },
    ) != NO_SECURITY_MODE;
  const status =
    testConnectionMutation.isPending || connectMutation.isPending
      ? "loading"
      : undefined;
  return (
    <Flex.Box align="start" className={CSS.B("opc-connect")} justify="center">
      <Flex.Box className={CSS.B("content")} grow gap="small">
        <Form.Form<typeof formSchema> {...methods}>
          <Form.TextField
            inputProps={{
              level: "h2",
              placeholder: "OPC UA Server",
              variant: "text",
            }}
            path="name"
          />
          <Form.Field<rack.Key> path="rack" label="Connect From" required>
            {({ value, onChange }) => (
              <Rack.SelectSingle value={value} onChange={onChange} allowNone={false} />
            )}
          </Form.Field>
          <Form.Field<string> path="connection.endpoint">
            {(p) => (
              <Input.Text autoFocus placeholder="opc.tcp://localhost:4840" {...p} />
            )}
          </Form.Field>
          <Divider.Divider x padded="bottom" />
          <Flex.Box x justify="between">
            <Form.Field<string> grow path="connection.username">
              {(p) => <Input.Text placeholder="admin" {...p} />}
            </Form.Field>
            <Form.Field<string> grow path="connection.password">
              {(p) => <Input.Text placeholder="password" type="password" {...p} />}
            </Form.Field>
            <Form.Field<SecurityMode>
              label="Security Mode"
              path="connection.securityMode"
            >
              {({ value, onChange }) => (
                <SelectSecurityMode value={value} onChange={onChange} />
              )}
            </Form.Field>
          </Flex.Box>
          <Divider.Divider x padded="bottom" />
          <Form.Field<SecurityPolicy>
            grow={!hasSecurity}
            path="connection.securityPolicy"
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
                path="connection.clientCertificate"
              >
                {FS.InputFilePath}
              </Form.Field>
              <Form.Field<string>
                label="Client Private Key"
                path="connection.clientPrivateKey"
              >
                {FS.InputFilePath}
              </Form.Field>
              <Form.Field<string>
                grow
                label="Server Certificate"
                path="connection.serverCertificate"
              >
                {FS.InputFilePath}
              </Form.Field>
            </>
          )}
        </Form.Form>
      </Flex.Box>
      <Modals.BottomNavBar>
        <Nav.Bar.Start gap="small">
          {connectionState == null ? (
            <Triggers.SaveHelpText action="Test Connection" noBar />
          ) : (
            <Text.Text status={connectionState.variant}>
              {connectionState.message}
            </Text.Text>
          )}
        </Nav.Bar.Start>
        <Nav.Bar.End>
          <Button.Button
            trigger={Triggers.SAVE}
            status={status}
            onClick={() => testConnectionMutation.mutate()}
          >
            Test Connection
          </Button.Button>
          <Button.Button
            status={status}
            onClick={() => connectMutation.mutate()}
            variant="filled"
          >
            Save
          </Button.Button>
        </Nav.Bar.End>
      </Modals.BottomNavBar>
    </Flex.Box>
  );
};

export const Connect: Layout.Renderer = ({ layoutKey, onClose }) => {
  const client = Synnax.use();
  const { isPending, isError, data, error } = useQuery<[FormSchema, Properties]>({
    queryKey: [layoutKey, client?.key],
    queryFn: async () => {
      if (client == null || layoutKey === CONNECT_LAYOUT_TYPE)
        return [
          { name: "OPC UA Server", connection: { ...ZERO_CONNECTION_CONFIG }, rack: 0 },
          deep.copy(ZERO_PROPERTIES),
        ];
      const dev = await client.hardware.devices.retrieve<Properties>({
        key: layoutKey,
      });
      dev.properties = migrateProperties(dev.properties);
      return [
        { name: dev.name, rack: dev.rack, connection: dev.properties.connection },
        dev.properties,
      ];
    },
  });
  if (isPending)
    return (
      <Text.Text center level="h4" status="loading">
        Loading Configuration from Synnax Server
      </Text.Text>
    );
  if (isError)
    return (
      <Flex.Box style={{ padding: "3rem" }}>
        <Text.Text level="h2" status="error">
          Failed to load configuration for server with key {layoutKey}
        </Text.Text>
        <Text.Text status="error">{error.message}</Text.Text>
      </Flex.Box>
    );

  const [initialValues, properties] = data;
  return (
    <Internal
      initialValues={initialValues}
      layoutKey={layoutKey}
      onClose={onClose}
      properties={properties}
    />
  );
};
