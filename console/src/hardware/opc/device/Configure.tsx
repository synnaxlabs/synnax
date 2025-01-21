// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/opc/device/Configure.css";

import { TimeSpan } from "@synnaxlabs/client";
import {
  Align,
  Button,
  Divider,
  Form,
  Input,
  Nav,
  Status,
  Synnax,
  Text,
  Triggers,
} from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import { useMutation, useQuery } from "@tanstack/react-query";
import { type ReactElement, useState } from "react";
import { v4 as uuid } from "uuid";
import { z } from "zod";

import { CSS } from "@/css";
import { FS } from "@/fs";
import { SelectSecurityMode } from "@/hardware/opc/device/SelectSecurityMode";
import { SelectSecurityPolicy } from "@/hardware/opc/device/SelectSecurityPolicy";
import {
  type ConnectionConfig,
  connectionConfigZ,
  MAKE,
  migrateProperties,
  type Properties,
  type SecurityMode,
  type SecurityPolicy,
  type TestConnectionCommandResponse,
  type TestConnectionCommandState,
  ZERO_CONNECTION_CONFIG,
  ZERO_PROPERTIES,
} from "@/hardware/opc/device/types";
import { Layout } from "@/layout";

export const CONFIGURE_LAYOUT_TYPE = "configureOPCServer";

export const CONFIGURE_LAYOUT: Layout.BaseState = {
  key: CONFIGURE_LAYOUT_TYPE,
  type: CONFIGURE_LAYOUT_TYPE,
  name: "OPC UA.Server.Connect",
  icon: "Logo.OPC",
  location: "modal",
  window: { resizable: false, size: { height: 710, width: 800 }, navTop: true },
};

const formSchema = z.object({
  name: z.string().min(1, "Name is required"),
  connection: connectionConfigZ,
});
interface FormSchema extends z.infer<typeof formSchema> {}

export const Configure: Layout.Renderer = ({ layoutKey, onClose }): ReactElement => {
  const client = Synnax.use();
  const { isPending, isError, data, error } = useQuery<[FormSchema, Properties]>({
    queryKey: [layoutKey, client?.key],
    queryFn: async () => {
      if (client == null || layoutKey === CONFIGURE_LAYOUT_TYPE)
        return [
          { name: "OPC UA Server", connection: { ...ZERO_CONNECTION_CONFIG } },
          deep.copy(ZERO_PROPERTIES),
        ];
      const dev = await client.hardware.devices.retrieve<Properties>(layoutKey);
      dev.properties = migrateProperties(dev.properties);
      return [
        { name: dev.name, connection: dev.properties.connection },
        dev.properties,
      ];
    },
  });
  if (isPending)
    return (
      <Status.Text.Centered variant="loading" level="h2">
        Loading Configuration from Synnax Server
      </Status.Text.Centered>
    );
  if (isError)
    return (
      <Align.Space direction="y" grow align="center" justify="center">
        <Text.Text level="h2" color={Status.variantColors.error}>
          Failed to load configuration for server with key {layoutKey}
        </Text.Text>
        <Text.Text level="p" color={Status.variantColors.error}>
          {error.message}
        </Text.Text>
      </Align.Space>
    );
  const [initialValues, properties] = data;
  return (
    <Internal
      onClose={onClose}
      layoutKey={layoutKey}
      properties={properties}
      initialValues={initialValues}
      visible
      focused
    />
  );
};

const SAVE_TRIGGER: Triggers.Trigger = ["Control", "Enter"];

interface InternalProps extends Layout.RendererProps {
  initialValues: FormSchema;
  properties?: Properties;
}

const Internal = ({
  initialValues,
  layoutKey,
  onClose,
  properties,
}: InternalProps): ReactElement => {
  const client = Synnax.use();
  const [connectionState, setConnectionState] =
    useState<TestConnectionCommandState | null>(null);
  const handleException = Status.useExceptionHandler();
  const methods = Form.use({ values: initialValues, schema: formSchema });
  const testConnection = useMutation<void, Error, void>({
    onError: (e) => handleException(e, "Failed to test connection"),
    mutationFn: async () => {
      if (client == null) throw new Error("Cannot reach Synnax server");
      if (!methods.validate("connection")) throw new Error("Invalid configuration");
      const rack = await client.hardware.racks.retrieve("sy_node_1_rack");
      const task = await rack.retrieveTaskByName("opc Scanner");
      const t = await task.executeCommandSync<TestConnectionCommandResponse>(
        "test_connection",
        { connection: methods.get("connection").value },
        TimeSpan.seconds(10),
      );
      setConnectionState(t);
    },
  });
  const confirm = useMutation<void, Error, void>({
    onError: (e) => handleException(e, "Failed to connect to OPC UA Server"),
    mutationFn: async () => {
      if (client == null) throw new Error("Cannot reach Synnax server");
      if (!methods.validate()) throw new Error("Invalid configuration");
      await testConnection.mutateAsync();
      if (connectionState?.variant !== "success")
        throw new Error("Connection test failed");
      const rack = await client.hardware.racks.retrieve("sy_node_1_rack");
      const key = layoutKey === CONFIGURE_LAYOUT_TYPE ? uuid() : layoutKey;
      await client.hardware.devices.create<Properties>({
        key,
        name: methods.get<string>("name").value,
        model: "opc",
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
  const hasSecPolicy =
    Form.useFieldValue<SecurityPolicy>("connection.securityMode", undefined, methods) !=
    "None";
  return (
    <Align.Space
      direction="y"
      justify="center"
      className={CSS.B("connect")}
      align="start"
      grow
    >
      <Align.Space direction="y" style={{ padding: "3rem 4rem" }} grow size="small">
        <Form.Form {...methods}>
          <Form.TextField
            path="name"
            inputProps={{
              level: "h2",
              variant: "natural",
              placeholder: "OPC UA Server",
            }}
          />
          <Form.Field<string> path="connection.endpoint">
            {(p) => (
              <Input.Text placeholder="opc.tcp://localhost:4840" autoFocus {...p} />
            )}
          </Form.Field>
          <Divider.Divider direction="x" padded="bottom" />
          <Align.Space direction="x" justify="spaceBetween">
            <Form.Field<string> path="connection.username" grow>
              {(p) => <Input.Text placeholder="admin" {...p} />}
            </Form.Field>
            <Form.Field<string> path="connection.password" grow>
              {(p) => <Input.Text placeholder="password" type="password" {...p} />}
            </Form.Field>
            <Form.Field<SecurityMode>
              path="connection.securityMode"
              label="Security Mode"
            >
              {(p) => <SelectSecurityMode {...p} />}
            </Form.Field>
          </Align.Space>
          <Divider.Divider direction="x" padded="bottom" />
          <Form.Field<SecurityPolicy>
            path="connection.securityPolicy"
            label="Security Policy"
            grow={!hasSecPolicy}
          >
            {(p) => <SelectSecurityPolicy size="medium" {...p} />}
          </Form.Field>
          {hasSecPolicy && (
            <>
              <Form.Field<string>
                path="connection.clientCertificate"
                label="Client Certificate"
              >
                {(p) => <FS.InputFilePath grow {...p} />}
              </Form.Field>
              <Form.Field<string>
                path="connection.clientPrivateKey"
                label="Client Private Key"
              >
                {(p) => <FS.InputFilePath grow {...p} />}
              </Form.Field>
              <Form.Field<string>
                path="connection.serverCertificate"
                label="Server Certificate"
                grow
              >
                {(p) => <FS.InputFilePath grow {...p} />}
              </Form.Field>
            </>
          )}
        </Form.Form>
      </Align.Space>
      <Layout.BottomNavBar>
        <Nav.Bar.Start size="small">
          {connectionState == null ? (
            <>
              <Triggers.Text shade={7} level="small" trigger={SAVE_TRIGGER} />
              <Text.Text shade={7} level="small">
                To Test Connection
              </Text.Text>
            </>
          ) : (
            <Status.Text variant={connectionState.variant as Status.Variant} level="p">
              {connectionState.details?.message}
            </Status.Text>
          )}
        </Nav.Bar.Start>
        <Nav.Bar.End>
          <Button.Button
            variant="outlined"
            triggers={[SAVE_TRIGGER]}
            loading={testConnection.isPending}
            disabled={testConnection.isPending}
            onClick={() => {
              testConnection.mutate();
            }}
          >
            Test Connection
          </Button.Button>
          <Button.Button onClick={() => confirm.mutate()}>Save</Button.Button>
        </Nav.Bar.End>
      </Layout.BottomNavBar>
    </Align.Space>
  );
};
