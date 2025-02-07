import { type channel, rack, task } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  Channel,
  Form,
  Rack,
  Status,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { useMutation } from "@tanstack/react-query";
import { z } from "zod";

import { Editor } from "@/code/Editor";
import { Common } from "@/hardware/common";
import { useCreate } from "@/hardware/common/task/useCreate";
// import { createLayoutCreator } from "@/hardware/task/common/createLayoutCreator";
import { type Layout } from "@/layout";
import {
  type Config,
  configZ,
  SEQUENCE_TYPE,
  type SequenceType,
  type StateDetails,
  ZERO_PAYLOAD,
} from "@/sequence/types";

export const SEQUENCE_LAYOUT: Common.Task.LayoutBaseState = {
  ...Common.Task.LAYOUT,
  type: SEQUENCE_TYPE,
  name: ZERO_PAYLOAD.name,
  icon: "Control",
  key: SEQUENCE_TYPE,
};

export const SEQUENCE_SELECTABLE: Layout.Selectable = {
  key: SEQUENCE_TYPE,
  title: "Control Sequence",
  icon: <Icon.Control />,
  create: async ({ layoutKey, rename }) => {
    const result = await rename(
      {},
      { icon: "Control", name: "Control.Sequence.Create" },
    );
    if (result == null) return null;
    return {
      ...SEQUENCE_LAYOUT,
      name: result,
      key: layoutKey,
    };
  },
};

export const Wrapped = ({
  task: base,
  layoutKey,
}: Common.Task.TaskProps<Config, StateDetails, SequenceType>) => {
  const client = Synnax.use();
  const methods = Form.use({
    values: {
      ...base,
      rack: task.rackKey(base?.key ?? "0"),
    },
    schema: z.object({
      name: z.string(),
      rack: rack.keyZ,
      config: configZ,
    }),
  });

  const create = useCreate(layoutKey);

  const configure = useMutation({
    mutationFn: async () => {
      if (!(await methods.validateAsync()) || client == null) return;
      const { name, config, rack } = methods.value();
      console.log(rack);
      await create(
        {
          key: base?.key,
          name,
          type: SEQUENCE_TYPE,
          config,
        },
        rack,
      );
    },
  });

  const [state, _] = Common.Task.useState(base?.key, base?.state ?? undefined);

  const start = useMutation({
    mutationFn: async () => {
      if (!(base instanceof task.Task)) throw new Error("Task has not been configured");
      if (state.state === "loading")
        throw new Error("State is loading, should not be able to start or stop task");
      await base.executeCommand(state.state === "running" ? "stop" : "start");
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
      <Form.Form
        {...methods}
        mode={base?.snapshot ? "preview" : "normal"}
        // style={{
        //   height: "100%",
        //   minHeight: 0,
        //   display: "flex",
        //   flexDirection: "column",
        // }}
      >
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
              <Form.Field<rack.Key>
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
              <Status.Text variant={state.variant as Status.Variant}>
                {state.message}
              </Status.Text>
            </Align.Space>
            <Button.Icon
              loading={startingOrStopping}
              disabled={startingOrStopping || base?.snapshot}
              onClick={() => onStartStop()}
              variant="outlined"
            >
              {state.state === "running" ? <Icon.Pause /> : <Icon.Play />}
            </Button.Icon>
            <Button.Button
              loading={configuring}
              disabled={configuring || base?.snapshot}
              onClick={() => onConfigure()}
              tooltip={
                <Align.Space direction="x" align="center" size="small">
                  {/* <Triggers.Text shade={7} level="small" /> */}
                  <Text.Text shade={7} level="small">
                    To Configure
                  </Text.Text>
                </Align.Space>
              }
            >
              Configure
            </Button.Button>
          </Align.Space>
        </Align.Pack>
      </Form.Form>
    </Align.Space>
  );
};

export const Configure = Common.Task.wrap(Wrapped, {
  configSchema: configZ,
  getInitialPayload: () => ZERO_PAYLOAD,
});
