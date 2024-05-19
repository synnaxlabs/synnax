import { useState, type ReactElement } from "react";

import { DataType, TimeSpan, type rack, type task } from "@synnaxlabs/client";
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
import { type UseMutationResult, useMutation } from "@tanstack/react-query";
import { nanoid } from "nanoid";
import { v4 as uuidv4 } from "uuid";
import { set, z } from "zod";

import { CSS } from "@/css";
import { CreateChannels } from "@/hardware/opc/device/CreateChannels";
import { type Properties, connectionConfigZ } from "@/hardware/opc/device/types";
import { type Layout } from "@/layout";

import "@/hardware/opc/device/Configure.css";

export const channelZ = z
  .object({
    dataType: z.string(),
    name: z.string(),
    nodeId: z.string().optional(),
    role: z.enum(["data", "index"]),
  })
  .superRefine((data) => {
    // Ensure that the node id is present if the role is data
    if (data.role === "data" && data.nodeId == null) {
      return { path: ["nodeId"], message: "Node ID is required for data channels" };
    }
    return true;
  });

const groupZ = z.object({
  key: z.string(),
  name: z.string(),
  channels: channelZ.array(),
});

type Group = z.infer<typeof groupZ>;

const configureZ = z.object({
  name: z.string().min(1, "Name is required"),
  connection: connectionConfigZ,
  groups: groupZ.array(),
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
        resizable: false,
        size: { height: 900, width: 1200 },
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
      name: "",
      connection: {
        endpoint: "opc.tcp://0.0.0.0:4840",
        username: "",
        password: "",
      },
      groups: [],
    },
    schema: configureZ,
  });

  const testConnection = useMutation({
    mutationKey: [client?.key],
    mutationFn: async () => {
      if (!(await methods.validateAsync()) || client == null) return;
      const rack = await client.hardware.racks.retrieve("sy_node_1_rack");
      const task = await rack.retrieveTaskByName("opc Scanner");
      return await task.executeCommandSync<{ message: string }>(
        "test_connection",
        { connection: methods.get({ path: "connection" }).value },
        TimeSpan.seconds(1),
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
          { connection: methods.get({ path: "connection" }).value },
          TimeSpan.seconds(1),
        );
        methods.set({
          path: "groups",
          value: [
            {
              key: nanoid(),
              name: "Group 1",
              channels: [
                ...deviceProperties.channels.map((c) => ({
                  ...c,
                  role: "data",
                  key: nanoid(),
                })),
                { key: nanoid(), name: "Time", dataType: "timestamp", role: "index" },
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
      )
        return;
      setProgress("Creating device...");
      await client.hardware.devices.create({
        key: uuidv4(),
        name: methods.get<string>({ path: "name" }).value,
        model: "opc",
        make: "opc",
        rack: rackKey,
        location: methods.get<string>({ path: "connection.endpoint" }).value,
        properties: deviceProperties,
      });
      setProgress("Creating channels...");
      const groups = methods.get<Group[]>({ path: "groups" }).value;
      for (const group of groups) {
        try {
          const idx = await client.channels.create({
            name: group.name,
            isIndex: true,
            dataType: DataType.TIMESTAMP.toString(),
          });
          setProgress(`Creating channels for ${group.name}...`);
          await client.channels.create(
            group.channels.map((c) => ({
              name: c.name,
              dataType: new DataType(c.dataType).toString(),
              index: idx.key,
            })),
          );
        } catch (e) {
          console.error(e);
        }
      }
    },
  });

  let content: ReactElement;
  if (step === "connect") {
    content = <Connect testConnection={testConnection} />;
  } else if (step === "createChannels" && deviceProperties != null) {
    content = <CreateChannels deviceProperties={deviceProperties} />;
  } else if (step === "confirm" && deviceProperties != null && rackKey != null) {
    content = (
      <Confirm
        confirm={confirm}
        deviceProperties={deviceProperties}
        progress={progress}
        rackKey={rackKey}
      />
    );
  } else {
    content = <div>Unknown step</div>;
  }

  return (
    <Align.Space className={CSS.B("configure")} align="stretch" grow empty>
      <Form.Form {...methods}>
        <Align.Space className={CSS.B("content")} grow>
          {content}
        </Align.Space>
        <Nav.Bar size={48} location="bottom">
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
        <Form.Field<string> path="connection.password" grow>
          {(p) => <Input.Text placeholder="password" {...p} />}
        </Form.Field>
        <Align.Space direction="x">
          <Align.Space direction="x" grow>
            {testConnection.isError && (
              <Status.Text variant="error">{testConnection.error.message}</Status.Text>
            )}
            {testConnection.isSuccess && testConnection.data?.variant != null && (
              <Status.Text variant={testConnection.data?.variant}>
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
          {/* {testConnection.} */}
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
      style={{ maxWidth: 600, padding: "20rem 20rem", borderRadius: "1rem" }}
      bordered
      rounded
      align="center"
      size={10}
    >
      <Text.Text level="h1">Ready to go?</Text.Text>
      <Text.Text level="h4" weight={500} shade={9}>
        Once you click "Configure", we'll permanently save your server's configuration
        and create all of the channels you've defined.
      </Text.Text>
      {progress != null && <Text.Text level="p">{progress}</Text.Text>}
      <Button.Button
        variant="outlined"
        size="large"
        onClick={() => confirm.mutate()}
        loading={confirm.isPending}
        disabled={confirm.isPending || confirm.isSuccess}
      >
        {confirm.isSuccess ? "Success!" : "Configure"}
      </Button.Button>
    </Align.Space>
  </Align.Center>
);
