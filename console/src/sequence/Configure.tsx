import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  Channel,
  Form,
  Synnax,
  Text,
  Triggers,
} from "@synnaxlabs/pluto";
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
  create: (layoutKey) => ({
    ...createSequenceLayout({ create: true }),
    key: layoutKey,
  }),
};

export const Wrapped = ({
  task,
  initialValues,
  layoutKey,
}: WrappedTaskLayoutProps<Task, Payload>) => {
  const client = Synnax.use();
  const methods = Form.use({
    values: initialValues,
    schema: z.object({ name: z.string(), config: configZ }),
  });

  const create = useCreate(layoutKey);

  const configure = useMutation({
    mutationFn: async () => {
      if (!(await methods.validateAsync()) || client == null) return;
      const { name, config } = methods.value();
      await create({
        key: task?.key,
        name,
        type: "sequence",
        config,
      });
    },
  });

  const taskState = useObserveState<StateDetails>(
    methods.setStatus,
    methods.clearStatuses,
    task?.key,
    task?.state,
  );

  const running = taskState?.details?.running;

  const start = useMutation({
    mutationFn: async () => {
      if (client == null) return;
      const isRunning = running === true;
      await task?.executeCommand(isRunning ? "stop" : "start");
    },
  });

  const startingOrStopping = start.isPending;
  const configuring = configure.isPending;
  const onStartStop = start.mutate;
  const onConfigure = configure.mutate;

  console.log(taskState);

  return (
    <Align.Space style={{ padding: 0, height: "100%" }} direction="y" grow empty>
      <Form.Form {...methods} mode={task?.snapshot ? "preview" : "normal"}>
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
            <Form.NumericField
              label="Loop Rate"
              path="config.rate"
              padHelpText={false}
              inputProps={{ endContent: "Hz" }}
            />
            <Form.Field<string>
              path="config.read"
              label="Read From"
              padHelpText={false}
            >
              {(p) => <Channel.SelectMultiple {...p} />}
            </Form.Field>
            <Form.Field<string>
              path="config.write"
              label="Write To"
              padHelpText={false}
            >
              {(p) => <Channel.SelectMultiple {...p} />}
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
              disabled={startingOrStopping || taskState == null || task?.snapshot}
              onClick={onStartStop}
              variant="outlined"
            >
              {taskState?.details?.running === true ? <Icon.Pause /> : <Icon.Play />}
            </Button.Icon>
            <Button.Button
              loading={configuring}
              disabled={configuring || task?.snapshot}
              onClick={onConfigure}
              tooltip={
                <Align.Space direction="x" align="center" size="small">
                  <Triggers.Text shade={7} level="small" />
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
