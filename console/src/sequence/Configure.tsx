// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, rack, task } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Button, Channel, Form, Rack, Status, Synnax } from "@synnaxlabs/pluto";
import { useMutation } from "@tanstack/react-query";
import { z } from "zod";

import { Editor } from "@/code/Editor";
import {
  useCreate,
  useObserveState,
  type WrappedTaskLayoutProps,
  wrapTaskLayout,
} from "@/hardware/task/common/common";
import { createLayoutCreator } from "@/hardware/task/common/createLayoutCreator";
import { type Layout } from "@/layout";
import {
  configZ,
  type Payload,
  type StateDetails,
  type Task,
  TYPE,
  ZERO_PAYLOAD,
} from "@/sequence/types";

export const createSequenceLayout = createLayoutCreator<Payload>(
  TYPE,
  "Control Sequence",
  "Control",
);

export const SEQUENCE_SELECTABLE: Layout.Selectable = {
  key: TYPE,
  title: "Control Sequence",
  icon: <Icon.Control />,
  create: async ({ layoutKey, rename }) => {
    const result = await rename(
      {},
      { icon: "Control", name: "Control.Sequence.Create" },
    );
    if (result == null) return null;
    return {
      ...createSequenceLayout({ create: true, initialValues: { name: result } }),
      name: result,
      key: layoutKey,
    };
  },
};

const schema = z.object({
  name: z.string(),
  rack: rack.rackKeyZ,
  config: configZ,
});

export const Wrapped = ({
  task: base,
  initialValues,
  layoutKey,
}: WrappedTaskLayoutProps<Task, Payload>) => {
  const client = Synnax.use();
  const methods = Form.use({
    values: {
      ...initialValues,
      rack: task.rackKey(base?.key ?? "0"),
    },
    schema,
  });

  const create = useCreate(layoutKey);

  const configure = useMutation({
    mutationFn: async () => {
      if (!(await methods.validateAsync()) || client == null) return;
      const { name, config, rack } = methods.value();
      await create(
        {
          key: base?.key,
          name,
          type: "sequence",
          config,
        },
        rack,
      );
    },
  });

  const taskState = useObserveState<StateDetails>(
    methods.setStatus,
    methods.clearStatuses,
    base?.key,
    base?.state,
  );

  const running = taskState?.details?.running;

  const start = useMutation({
    mutationFn: async () => {
      if (client == null) return;
      const isRunning = running === true;
      await base?.executeCommand(isRunning ? "stop" : "start");
    },
  });

  const startingOrStopping = start.isPending;
  const configuring = configure.isPending;
  const onStartStop = start.mutate;
  const onConfigure = configure.mutate;

  return (
    <Align.Space
      style={{ padding: 0, height: "100%", minHeight: 0 }}
      direction="y"
      empty
    >
      <Form.Form {...methods} mode={base?.snapshot ? "preview" : "normal"}>
        <Form.Field<string>
          path="config.script"
          showLabel={false}
          showHelpText={false}
          padHelpText={false}
          style={{
            height: "100%",
            width: "100%",
            minHeight: 0,
            display: "flex",
            flex: 1,
            flexShrink: 1,
            overflow: "hidden",
          }}
        >
          {(p) => <Editor style={{ height: "100%", width: "100%", flex: 1 }} {...p} />}
        </Form.Field>
        <Align.Pack
          direction="y"
          bordered={false}
          style={{
            width: "100%",
            background: "var(--pluto-gray-l0)",
            boxShadow: "var(--pluto-shadow-menu)",
            borderTop: "var(--pluto-border)",
            flexShrink: 0, // Prevent the bottom section from shrinking
          }}
        >
          <Align.Space direction="y" style={{ padding: "2rem" }}>
            <Align.Space direction="x">
              <Form.Field<rack.RackKey>
                path="rack"
                label="Location"
                padHelpText={false}
                grow
              >
                {(p) => <Rack.SelectSingle allowNone={false} {...p} />}
              </Form.Field>
              <Form.NumericField
                label="Loop Rate"
                path="config.rate"
                padHelpText={false}
                style={{ width: 120 }}
                inputProps={{
                  endContent: "Hz",
                  bounds: { lower: 1, upper: 1001 },
                  dragScale: { x: 1, y: 1 },
                }}
              />
            </Align.Space>
            <Form.Field<channel.Key[]>
              path="config.read"
              label="Read From"
              padHelpText={false}
            >
              {({ value, onChange }) => (
                <Channel.SelectMultiple value={value} onChange={onChange} />
              )}
            </Form.Field>
            <Form.Field<channel.Key[]>
              path="config.write"
              label="Write To"
              padHelpText={false}
            >
              {({ value, onChange }) => (
                <Channel.SelectMultiple value={value} onChange={onChange} />
              )}
            </Form.Field>
          </Align.Space>

          <Align.Space
            direction="x"
            rounded
            style={{
              padding: "2rem",
              borderTop: "var(--pluto-border)",
            }}
            justify="spaceBetween"
          >
            <Align.Space
              direction="x"
              style={{
                borderRadius: "1rem",
                width: "100%",
              }}
            >
              {taskState?.details?.message != null && (
                <Status.Text variant={taskState?.variant as Status.Variant}>
                  {taskState?.details?.message}
                </Status.Text>
              )}
            </Align.Space>
            <Button.Icon
              loading={startingOrStopping}
              disabled={startingOrStopping || taskState == null || base?.snapshot}
              onClick={() => onStartStop()}
              variant="outlined"
            >
              {taskState?.details?.running === true ? <Icon.Pause /> : <Icon.Play />}
            </Button.Icon>
            <Button.Button
              loading={configuring}
              disabled={configuring || base?.snapshot}
              onClick={() => onConfigure()}
            >
              Configure
            </Button.Button>
          </Align.Space>
        </Align.Pack>
      </Form.Form>
    </Align.Space>
  );
};

export const Configure = wrapTaskLayout(Wrapped, ZERO_PAYLOAD);
