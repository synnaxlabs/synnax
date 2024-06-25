// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/opc/device/Configure.css";

import {
  DataType,
  type rack,
  type task,
  TimeSpan,
  UnexpectedError,
} from "@synnaxlabs/client";
import {
  Align,
  Button,
  Form,
  Input,
  Nav,
  Status,
  Steps,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { useMutation, type UseMutationResult } from "@tanstack/react-query";
import { nanoid } from "nanoid";
import { type ReactElement, useState } from "react";
import { v4 as uuidv4 } from "uuid";
import { z } from "zod";

import { CSS } from "@/css";
import { FS } from "@/fs";
import { CreateChannels } from "@/hardware/opc/device/CreateChannels";
import {
  SelectSecurityMode,
  SelectSecurityPolicy,
} from "@/hardware/opc/device/SelectSecurityPolicy";
import {
  connectionConfigZ,
  GroupConfig,
  groupConfigZ,
  type Properties,
  SecurityMode,
  SecurityPolicy,
} from "@/hardware/opc/device/types";
import { type Layout } from "@/layout";

const configureZ = z.object({
  name: z.string().min(1, "Name is required"),
  connection: connectionConfigZ,
  groups: groupConfigZ.array(),
});

export const CONFIGURE_LAYOUT_TYPE = "configureOPCServer";

export const createConfigureLayout =
  (device?: string, initial: Omit<Partial<Layout.State>, "type"> = {}) =>
  (): Layout.State => {
    const { name = "Configure OPC UA Server", location = "window", ...rest } = initial;
    return {
      key: device ?? initial.key ?? CONFIGURE_LAYOUT_TYPE,
      type: CONFIGURE_LAYOUT_TYPE,
      windowKey: device ?? initial.key ?? CONFIGURE_LAYOUT_TYPE,
      name,
      window: {
        navTop: true,
        resizable: true,
        size: { height: 1000, width: 1300 },
      },
      location,
      ...rest,
    };
  };

const STEPS: Steps.Step[] = [
  {
    key: "connect",
    title: "Connect",
  },
  {
    key: "createChannels",
    title: "Create Channels",
  },
  {
    key: "confirm",
    title: "Confirm",
  },
];

export const Configure: Layout.Renderer = ({ onClose }): ReactElement => {
  const client = Synnax.use();
  const [step, setStep] = useState("connect");
  const [deviceProperties, setDeviceProperties] = useState<Properties | null>(null);
  const [rackKey, setRackKey] = useState<rack.RackKey | null>(null);
  const [progress, setProgress] = useState<string | undefined>(undefined);

  const methods = Form.use({
    values: {
      name: "My OPC UA Server",
      connection: {
        endpoint: "opc.tcp://0.0.0.0:4840",
        username: "",
        password: "",
        server_certificate: "",
        client_certificate: "",
        client_private_key: "",
        security_policy: "None",
        security_mode: "None",
      },
      groups: [],
    },
    schema: configureZ,
  });

  const testConnection = useMutation({
    mutationKey: [client?.key],
    mutationFn: async () => {
      if (!(await methods.validateAsync("connection")) || client == null) return;
      const rack = await client.hardware.racks.retrieve("sy_node_1_rack");
      const task = await rack.retrieveTaskByName("opc Scanner");
      return await task.executeCommandSync<{ message: string }>(
        "test_connection",
        { connection: methods.get("connection").value },
        TimeSpan.seconds(10),
      );
    },
  });

  const handleNextStep = useMutation({
    mutationKey: [step, client?.key],
    mutationFn: async () => {
      if (!(await methods.validateAsync()) || client == null) return;
      const rack = await client.hardware.racks.retrieve("sy_node_1_rack");
      if (step === "connect") {
        await testConnection.mutateAsync();
        const task = await rack.retrieveTaskByName("opc Scanner");
        const { details: deviceProperties } = await task.executeCommandSync<Properties>(
          "scan",
          { connection: methods.get("connection").value },
          TimeSpan.seconds(20),
        );
        if (deviceProperties == null) return;
        methods.set({
          path: "groups",
          value: [
            {
              key: nanoid(),
              name: "Group 1",
              channels: [
                {
                  key: nanoid(),
                  name: "group_1_time",
                  dataType: "timestamp",
                  nodeId: "",
                  isIndex: true,
                  isArray: false,
                },
                ...deviceProperties.channels.map((c) => ({
                  ...c,
                  key: nanoid(),
                  isIndex: false,
                })),
              ],
            },
          ],
        });
        setDeviceProperties(deviceProperties);
        setRackKey(rack.key);
        setStep("createChannels");
      } else if (step === "createChannels") {
        setStep("confirm");
      } else {
        onClose();
      }
    },
  });

  const confirm = useMutation({
    mutationKey: [client?.key],
    mutationFn: async () => {
      if (
        !(await methods.validateAsync()) ||
        client == null ||
        rackKey == null ||
        deviceProperties == null
      ) {
        console.log("ExiT");
        return;
      }
      setProgress("Creating channels...");
      const groups = methods.get<GroupConfig[]>({ path: "groups" }).value;
      const mapped = new Map<string, number>();
      for (const group of groups) {
        // find the index channel
        const idxBase = group.channels.find((c) => c.isIndex);
        if (idxBase == null) throw new UnexpectedError("No index channel found");
        const idx = await client.channels.create({
          name: idxBase.name,
          isIndex: true,
          dataType: DataType.TIMESTAMP.toString(),
        });
        mapped.set(idxBase.nodeId, idx.key);
        setProgress(`Creating channels for ${group.name}...`);
        const toCreate = group.channels.filter((c) => !c.isIndex);
        const channels = await client.channels.create(
          toCreate.map((c) => ({
            name: c.name,
            dataType: new DataType(c.dataType).toString(),
            index: idx.key,
          })),
        );
        channels.forEach((c, i) => {
          const nodeId = toCreate[i].nodeId;
          if (nodeId != null) mapped.set(nodeId, c.key);
        });
      }
      setProgress("Creating device...");
      deviceProperties.channels.forEach((c) => {
        const synnaxChannel = mapped.get(c.nodeId);
        if (synnaxChannel == null) return;
        c.synnaxChannel = synnaxChannel;
      });
      await client.hardware.devices.create({
        key: uuidv4(),
        name: methods.get<string>({ path: "name" }).value,
        model: "opc",
        make: "opc",
        rack: rackKey,
        location: methods.get<string>({ path: "connection.endpoint" }).value,
        properties: deviceProperties,
        configured: true,
      });
    },
  });

  let content: ReactElement;
  if (step === "connect") {
    content = <Connect testConnection={testConnection} />;
  } else if (step === "createChannels" && deviceProperties != null) {
    content = <CreateChannels deviceProperties={deviceProperties} />;
  } else if (step === "confirm" && deviceProperties != null) {
    content = <Confirm confirm={confirm} progress={progress} />;
  } else {
    content = <div>Unknown step</div>;
  }

  return (
    <Align.Space className={CSS.B("configure")} align="stretch" grow empty>
      <Form.Form {...methods}>
        <Align.Space className={CSS.B("content")} grow>
          {content}
        </Align.Space>
        <Nav.Bar size={48} location="bottom" style={{ position: "fixed", bottom: 0 }}>
          <Nav.Bar.Start>
            <Steps.Steps value={step} onChange={setStep} steps={STEPS} />
          </Nav.Bar.Start>
          <Nav.Bar.End>
            {confirm.isIdle && (
              <Button.Button variant="outlined" onClick={onClose}>
                Cancel
              </Button.Button>
            )}
            <Button.Button
              onClick={() => handleNextStep.mutate()}
              loading={handleNextStep.isPending && step !== "confirm"}
              disabled={
                handleNextStep.isPending ||
                confirm.isPending ||
                (confirm.isIdle && step === "confirm")
              }
            >
              {confirm.isSuccess ? "Done" : "Next"}
            </Button.Button>
          </Nav.Bar.End>
        </Nav.Bar>
      </Form.Form>
    </Align.Space>
  );
};

interface ConnectProps {
  testConnection: UseMutationResult<
    | task.State<{
        message: string;
      }>
    | undefined,
    Error,
    void,
    unknown
  >;
}

const Connect = ({ testConnection }: ConnectProps): ReactElement => {
  const hasSecPolicy =
    Form.useFieldValue<SecurityPolicy>("connection.security_policy") != "None";
  return (
    <Align.Space
      direction="x"
      justify="center"
      className={CSS.B("connect")}
      align="start"
      empty
      grow
      size={15}
    >
      <Align.Space className={CSS.B("description")} direction="y">
        <Text.Text level="h1">Let's connect your OPC UA Server</Text.Text>
        <Text.Text level="p">
          To start off, we'll need to know the connection details for your server along
          with a name to identify it later.
        </Text.Text>
        <Text.Text level="p">
          Use the "Test Connection" button in the bottom right corner to verify that the
          connection is successful.
        </Text.Text>
        <Text.Text level="p">
          A detailed walkthrough on how to configure your server can be found in our{" "}
          <Text.Link
            level="p"
            href="https://docs.synnaxlabs.com/reference/device-drivers/opcua/connect-server"
            target="_blank"
            style={{ display: "inline" }}
          >
            documentation.
          </Text.Link>
        </Text.Text>
      </Align.Space>
      <Align.Space
        direction="y"
        grow
        align="stretch"
        className={CSS.B("form")}
        style={{ padding: "5rem" }}
      >
        <Form.Field<string> path="name">
          {(p) => <Input.Text placeholder="Name" autoFocus {...p} />}
        </Form.Field>
        <Form.Field<string> path="connection.endpoint">
          {(p) => (
            <Input.Text placeholder="opc.tcp://localhost:4840" autoFocus {...p} />
          )}
        </Form.Field>
        <Form.Field<string> path="connection.username">
          {(p) => <Input.Text placeholder="admin" {...p} />}
        </Form.Field>
        <Form.Field<string> path="connection.password">
          {(p) => <Input.Text placeholder="password" type="password" {...p} />}
        </Form.Field>
        <Form.Field<SecurityMode> path="connection.security_mode" label="Security Mode">
          {(p) => <SelectSecurityMode {...p} />}
        </Form.Field>
        <Form.Field<SecurityPolicy>
          path="connection.security_policy"
          label="Security Policy"
          grow={!hasSecPolicy}
        >
          {(p) => <SelectSecurityPolicy {...p} />}
        </Form.Field>
        {hasSecPolicy && (
          <>
            <Form.Field<string>
              path="connection.client_certificate"
              label="Client Certificate"
            >
              {(p) => <FS.InputFilePath grow {...p} />}
            </Form.Field>
            <Form.Field<string>
              path="connection.client_private_key"
              label="Client Private Key"
            >
              {(p) => <FS.InputFilePath grow {...p} />}
            </Form.Field>
            <Form.Field<string>
              path="connection.server_certificate"
              label="Server Certificate"
              grow
            >
              {(p) => <FS.InputFilePath grow {...p} />}
            </Form.Field>
          </>
        )}
        <Align.Space direction="x">
          <Align.Space direction="x" grow>
            {testConnection.isError && (
              <Status.Text variant="error">{testConnection.error.message}</Status.Text>
            )}
            {testConnection.isSuccess && testConnection.data?.variant != null && (
              <Status.Text variant={testConnection.data?.variant as Status.Variant}>
                {testConnection.data?.details?.message}
              </Status.Text>
            )}
          </Align.Space>
          <Button.Button
            variant="outlined"
            loading={testConnection.isPending}
            disabled={testConnection.isPending}
            onClick={() => testConnection.mutate()}
          >
            Test Connection
          </Button.Button>
        </Align.Space>
      </Align.Space>
    </Align.Space>
  );
};

interface ConfirmProps {
  confirm: UseMutationResult<void, Error, void, unknown>;
  progress?: string;
}

const Confirm = ({ progress, confirm }: ConfirmProps): ReactElement => (
  <Align.Center>
    <Align.Space
      style={{
        maxWidth: 600,
        padding: "20rem 20rem",
        borderRadius: "1rem",
        backgroundColor: "var(--pluto-gray-l1)",
      }}
      bordered
      rounded
      align="center"
      size={10}
    >
      <Text.Text level="h1">Ready to go?</Text.Text>
      <Text.Text level="h4" weight={400} shade={9}>
        Once you click "Configure", we'll permanently save your server's configuration
        and create all of the channels you've defined.
      </Text.Text>
      {progress != null && <Text.Text level="p">{progress}</Text.Text>}
      <Button.Button
        size="large"
        onClick={() => confirm.mutate()}
        loading={confirm.isPending}
        disabled={confirm.isPending || confirm.isSuccess || confirm.isError}
      >
        {confirm.isSuccess ? "Success!" : "Configure"}
      </Button.Button>
      {confirm.isError && (
        <Status.Text variant="error">{confirm.error.message}</Status.Text>
      )}
    </Align.Space>
  </Align.Center>
);
