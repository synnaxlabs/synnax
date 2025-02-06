import { type channel, rack, task } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Button, Channel, Form, Rack, Synnax, Text } from "@synnaxlabs/pluto";
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
  SEQUENCE_TYPE,
  type StateDetails,
  type Task,
  ZERO_PAYLOAD,
} from "@/sequence/types";

export const createSequenceLayout = createLayoutCreator<Payload>(
  SEQUENCE_TYPE,
  "Control Sequence",
  "Control",
);

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
      ...createSequenceLayout({ create: true, initialValues: { name: result } }),
      name: result,
      key: layoutKey,
    };
  },
};

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
    schema: z.object({
      name: z.string(),
      rack: rack.rackKeyZ,
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
    <Align.Space style={{ padding: 0, height: "100%" }} direction="y" grow empty>
      <Form.Form {...methods} mode={base?.snapshot ? "preview" : "normal"}>
        <Form.Field<string>
          path="config.script"
          showLabel={false}
          showHelpText={false}
          padHelpText={false}
        >
          {(p) => <Editor style={{ height: "100%" }} {...p} />}
        </Form.Field>
        <Align.Pack
          direction="y"
          style={{
            position: "absolute",
            width: "calc(100% - 6rem)",
            bottom: 20,
            left: "3rem",
            border: "var(--pluto-border)",
            background: "var(--pluto-gray-l0)",
            boxShadow: "var(--pluto-shadow-menu)",
            "--pluto-pack-br": "1rem",
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
            bordered
            rounded
            style={{
              padding: "2rem",
            }}
            justify="end"
          >
            <Button.Icon
              loading={startingOrStopping}
              disabled={startingOrStopping || taskState == null || base?.snapshot}
              onClick={onStartStop}
              variant="outlined"
            >
              {taskState?.details?.running === true ? <Icon.Pause /> : <Icon.Play />}
            </Button.Icon>
            <Button.Button
              loading={configuring}
              disabled={configuring || base?.snapshot}
              onClick={onConfigure}
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

export const Configure = wrapTaskLayout(Wrapped, ZERO_PAYLOAD);
