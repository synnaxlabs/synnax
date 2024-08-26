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
import { useMutation } from "@tanstack/react-query";
import { type ReactElement, useState } from "react";
import { v4 as uuidv4 } from "uuid";
import { z } from "zod";

import { CSS } from "@/css";
import { FS } from "@/fs";
import {
  SelectSecurityMode,
  SelectSecurityPolicy,
} from "@/hardware/opc/device/SelectSecurityPolicy";
import {
  connectionConfigZ,
  type Properties,
  SecurityMode,
  SecurityPolicy,
  TestConnCommandResponse,
  TestConnCommandState,
} from "@/hardware/opc/device/types";
import { type Layout } from "@/layout";

const configureZ = z.object({
  name: z.string().min(1, "Name is required"),
  connection: connectionConfigZ,
});

export const CONFIGURE_LAYOUT_TYPE = "configureOPCServer";

const SAVE_TRIGGER: Triggers.Trigger = ["Control", "Enter"];

export const createConfigureLayout =
  (device?: string, initial: Omit<Partial<Layout.State>, "type"> = {}) =>
  (): Layout.State => {
    const { name = "OPC UA.Connect", location = "modal", ...rest } = initial;
    return {
      key: device ?? initial.key ?? CONFIGURE_LAYOUT_TYPE,
      type: CONFIGURE_LAYOUT_TYPE,
      windowKey: device ?? initial.key ?? CONFIGURE_LAYOUT_TYPE,
      name,
      icon: "Logo.OPC",
      window: {
        navTop: true,
        resizable: true,
        size: { height: 710, width: 800 },
      },
      location,
      ...rest,
    };
  };

export const Configure: Layout.Renderer = ({ onClose }): ReactElement => {
  const client = Synnax.use();
  const [connState, setConnState] = useState<TestConnCommandState | null>(null);

  const methods = Form.use({
    values: {
      name: "My OPC UA Server",
      connection: {
        endpoint: "opc.tcp://0.0.0.0:4840",
        username: "",
        password: "",
        serverCertificate: "",
        clientCertificate: "",
        clientPrivateKey: "",
        securityPolicy: "None",
        securityMode: "None",
      },
    },
    schema: configureZ,
  });

  const testConnection = useMutation<void, Error, void>({
    mutationKey: [client?.key],
    mutationFn: async () => {
      if (!methods.validate("connection") || client == null) return;
      const rack = await client.hardware.racks.retrieve("sy_node_1_rack");
      const task = await rack.retrieveTaskByName("opc Scanner");
      const t = await task.executeCommandSync<TestConnCommandResponse>(
        "test_connection",
        { connection: methods.get("connection").value },
        TimeSpan.seconds(10),
      );
      setConnState(t);
    },
  });

  const confirm = useMutation<void, Error, void>({
    mutationKey: [client?.key],
    mutationFn: async () => {
      if (!methods.validate() || client == null) return;
      await testConnection.mutateAsync();
      if (connState?.variant !== "success") return;
      const rack = await client.hardware.racks.retrieve("sy_node_1_rack");
      try {
        await client.hardware.devices.create({
          key: uuidv4(),
          name: methods.get<string>("name").value,
          model: "opc",
          make: "opc",
          rack: rack.key,
          location: methods.get<string>("connection.endpoint").value,
          properties: {
            connection: methods.get<Properties>("connection").value,
            read: {
              index: 0,
              channels: [],
            },
            write: {
              index: 0,
              channels: [],
            }
          },
          configured: true,
        });
      } catch (e) {
        console.error(e);
      }
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
              placeholder: "name",
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
      <Nav.Bar location="bottom" style={{ paddingRight: "2rem" }}>
        <Nav.Bar.Start style={{ paddingLeft: "2rem" }} size="small">
          {connState == null ? (
            <>
              <Triggers.Text shade={7} level="small" trigger={SAVE_TRIGGER} />
              <Text.Text shade={7} level="small">
                To Test Connection
              </Text.Text>
            </>
          ) : (
            <Status.Text variant={connState.variant as Status.Variant} level="p">
              {connState.details?.message}
            </Status.Text>
          )}
        </Nav.Bar.Start>
        <Nav.Bar.End>
          <Button.Button
            variant="outlined"
            triggers={[SAVE_TRIGGER]}
            loading={testConnection.isPending}
            disabled={testConnection.isPending}
            onClick={() => testConnection.mutate()}
          >
            Test Connection
          </Button.Button>
          <Button.Button onClick={() => confirm.mutate()}>Save</Button.Button>
        </Nav.Bar.End>
      </Nav.Bar>
    </Align.Space>
  );
};
