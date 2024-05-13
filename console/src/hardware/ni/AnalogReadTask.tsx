// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useState, type ReactElement, useCallback } from "react";

import {
  Channel,
  Form,
  Select,
  Device,
  Header,
  Synnax,
  Nav,
  Button,
} from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { Input } from "@synnaxlabs/pluto/input";
import { Text } from "@synnaxlabs/pluto/text";

import { CSS } from "@/css";
import { ChannelList } from "@/hardware/ni/ChannelList";
import {
  AIChanType,
  AnalogReadTaskConfig,
  analogReadTaskConfigZ,
  AnalogReadTaskState,
  DEFAULT_SCALES,
  type LinearScale,
  type LinearScaleType,
} from "@/hardware/ni/types";

import "@/hardware/ni/AnalogReadTask.css";
import { Layout } from "@/layout";
import { useMutation, useQuery } from "@tanstack/react-query";
import { task } from "@synnaxlabs/client";
import { z } from "zod";
import { Icon } from "@synnaxlabs/media";
import { ANALOG_INPUT_FORMS, SelectChannelTypeField } from "./ChannelForms";

export const analogReadTaskLayout: Layout.LayoutState = {
  name: "Configure NI Analog Read Task",
  key: "niAnalogReadTask",
  type: "niAnalogReadTask",
  windowKey: "niAnalogReadTask",
  location: "window",
  window: {
    resizable: false,
    size: { width: 1200, height: 900 },
    navTop: true,
  },
};

export const AnalogReadTask: Layout.Renderer = ({ layoutKey }) => {
  const client = Synnax.use();
  const fetchTask = useQuery<AnalogReadTaskInternalProps>({
    queryKey: [layoutKey, client?.key],
    queryFn: async () => {
      if (client == null || layoutKey == analogReadTaskLayout.key)
        return {
          initialValues: {
            key: "niAnalogReadTask",
            type: "niAnalogReadTask",
            name: "NI Analog Read Task",
            config: {
              device: "",
              sampleRate: 50,
              streamRate: 25,
              channels: [],
            },
          },
        };
      const t = await client.hardware.tasks.retrieve<
        AnalogReadTaskConfig,
        AnalogReadTaskState
      >(layoutKey, { includeState: true });
      return { initialValues: t, task: t };
    },
  });
  if (fetchTask.isLoading) return <></>;
  if (fetchTask.isError) return <></>;
  return <AnalogReadTaskInternal {...fetchTask.data} />;
};

export interface AnalogReadTaskInternalProps {
  task?: task.Task<AnalogReadTaskConfig, AnalogReadTaskState>;
  initialValues: task.TaskPayload<AnalogReadTaskConfig, AnalogReadTaskState>;
}

const AnalogReadTaskInternal = ({
  task: pTask,
  initialValues,
}: AnalogReadTaskInternalProps): ReactElement | null => {
  const client = Synnax.use();
  const methods = Form.use({
    values: initialValues,
    schema: z.object({
      name: z.string(),
      config: analogReadTaskConfigZ,
    }),
  });

  const [task, setTask] = useState(pTask);
  const [taskState, setTaskState] = useState<AnalogReadTaskState | null>(null);

  const [selectedChannels, setSelectedChannels] = useState<string[]>([]);
  const [selectedChannelIndex, setSelectedChannelIndex] = useState<number | null>(null);

  const configure = useMutation({
    mutationKey: [client?.key, "configure"],
    mutationFn: async () => {
      if (!(await methods.validateAsync()) || client == null) return;
      const rack = await client.hardware.racks.retrieve("sy_node_1_rack");
      const { name, config } = methods.value();
      setTask(
        await rack.createTask<AnalogReadTaskConfig>({
          key: task?.key,
          name,
          type: "niAnalogReader",
          config,
        }),
      );
    },
  });

  const start = useMutation({
    mutationKey: [client?.key, "start"],
    mutationFn: async () => {
      if (client == null) return;
      await task?.executeCommand(
        taskState?.details?.running === true ? "stop" : "start",
      );
    },
  });

  return (
    <Align.Space className={CSS.B("ni-analog-read-task")} direction="y" grow empty>
      <Align.Space className={CSS.B("content")} grow>
        <Form.Form {...methods}>
          <Align.Space direction="x">
            <Form.Field<string> path="name">
              {(p) => <Input.Text variant="natural" level="h1" {...p} />}
            </Form.Field>
          </Align.Space>
          <Align.Space direction="x">
            <Form.Field<string> path="config.device" label="Device" grow>
              {(p) => (
                <Device.SelectSingle
                  allowNone={false}
                  grow
                  {...p}
                  searchOptions={{ makes: ["NI"] }}
                />
              )}
            </Form.Field>
            <Form.Field<number> label="Sample Rate" path="config.sampleRate">
              {(p) => <Input.Numeric {...p} />}
            </Form.Field>
            <Form.Field<number> label="Stream Rate" path="config.streamRate">
              {(p) => <Input.Numeric {...p} />}
            </Form.Field>
          </Align.Space>
          <Align.Space
            direction="x"
            className={CSS.B("channel-form-container")}
            bordered
            rounded
            grow
            empty
          >
            <ChannelList
              path="config.channels"
              selected={selectedChannels}
              onSelect={useCallback(
                (v, i) => {
                  setSelectedChannels(v);
                  setSelectedChannelIndex(i);
                },
                [setSelectedChannels, setSelectedChannelIndex],
              )}
            />
            <Align.Space className={CSS.B("channel-form")} direction="y" grow>
              <Header.Header level="h3">
                <Header.Title weight={500}>Channel Details</Header.Title>
              </Header.Header>
              <Align.Space className={CSS.B("details")}>
                {selectedChannelIndex != null && (
                  <ChannelForm selectedChannelIndex={selectedChannelIndex} />
                )}
              </Align.Space>
            </Align.Space>
          </Align.Space>
        </Form.Form>
      </Align.Space>
      <Nav.Bar location="bottom" size={48}>
        <Nav.Bar.End style={{ paddingRight: "2rem" }}>
          <Button.ToggleIcon
            loading={start.isPending}
            disabled={start.isPending || taskState == null}
            onChange={start.mutate}
          >
            {taskState?.details?.running === true ? <Icon.Pause /> : <Icon.Play />}
          </Button.ToggleIcon>
          <Button.Button
            loading={configure.isPending}
            disabled={configure.isPending}
            onClick={configure.mutate}
          >
            Configure
          </Button.Button>
        </Nav.Bar.End>
      </Nav.Bar>
    </Align.Space>
  );
};

interface ChannelFormProps {
  selectedChannelIndex: number;
}

const ChannelForm = ({ selectedChannelIndex }: ChannelFormProps): ReactElement => {
  if (selectedChannelIndex == -1) return <></>;
  const prefix = `config.channels.${selectedChannelIndex}`;
  const ctx = Form.useContext();
  const type = ctx.get<AIChanType>({ path: `${prefix}.type` });
  const TypeForm = ANALOG_INPUT_FORMS[type.value];
  const [counter, setCounter] = useState(0);
  Form.useFieldListener<AIChanType>({
    path: `${prefix}.type`,
    onChange: (v) => setCounter((c) => c + 1),
  });

  return (
    <>
      <Align.Space direction="y" className={CSS.B("channel-form-content")} empty>
        <SelectChannelTypeField path={prefix} inputProps={{ allowNone: false }} />
        <TypeForm prefix={prefix} />
      </Align.Space>
    </>
  );
};

interface ScaleEntry {
  key: LinearScaleType;
  label: string;
}

const SCALE_DATA: ScaleEntry[] = [
  {
    key: "none",
    label: "None",
  },
  {
    label: "Linear",
    key: "linear",
  },
];

const SelectScale = (
  props: Omit<Select.DropdownButtonProps<LinearScaleType, ScaleEntry>, "data">,
): ReactElement => (
  <Select.DropdownButton<LinearScaleType, ScaleEntry>
    columns={[
      {
        key: "label",
        name: "Scale",
      },
    ]}
    data={SCALE_DATA}
    entryRenderKey="label"
    {...props}
  />
);

interface ScaleFormProps {
  path: string;
}

const ScaleForm = ({ path }: ScaleFormProps): ReactElement | null => {
  const typeField = Form.useField<LinearScaleType>({
    path: `${path}.type`,
  });
  if (typeField.value === "none") return null;
  return (
    <Align.Space direction="y" grow empty>
      <Align.Space direction="x">
        <Form.Field<number> label="Raw Min" path={`${path}.one.x`} grow>
          {(p) => <Input.Numeric {...p} />}
        </Form.Field>
        <Form.Field<number> label="Raw Max" path={`${path}.two.x`} grow>
          {(p) => <Input.Numeric {...p} />}
        </Form.Field>
      </Align.Space>
      <Align.Space direction="x">
        <Form.Field<number> label="Scaled Min" path={`${path}.one.y`} grow>
          {(p) => <Input.Numeric {...p} />}
        </Form.Field>
        <Form.Field<number> label="Scaled Max" path={`${path}.two.y`} grow>
          {(p) => <Input.Numeric {...p} />}
        </Form.Field>
      </Align.Space>
    </Align.Space>
  );
};
