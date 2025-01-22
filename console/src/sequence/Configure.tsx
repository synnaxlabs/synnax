import { Icon } from "@synnaxlabs/media";
import { Align, Channel, Form, Input, Synnax } from "@synnaxlabs/pluto";
import { useMutation } from "@tanstack/react-query";
import { z } from "zod";

import { Editor } from "@/code/Editor";
import {
  Controls,
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
  "New LabJack Sequence Task",
);

export const SEQUENCE_SELECTABLE: Layout.Selectable = {
  key: SEQUENCE_TYPE,
  title: "LabJack Sequence Task",
  icon: <Icon.Logo.LabJack />,
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
  console.log(initialValues);
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
      console.log(config);
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

  return (
    <Align.Space style={{ padding: 0 }} direction="y" grow empty>
      <Form.Form {...methods} mode={task?.snapshot ? "preview" : "normal"}>
        <Form.Field<string>
          path="config.script"
          showLabel={false}
          showHelpText={false}
          padHelpText={!task?.snapshot}
        >
          {(p) => <Editor style={{ height: 600 }} {...p} />}
        </Form.Field>
        <Form.Field<string> path="name" padHelpText={!task?.snapshot}>
          {(p) => <Input.Text variant="natural" level="h2" {...p} />}
        </Form.Field>
        <Form.NumericField path="config.rate" padHelpText={!task?.snapshot} />
        <Form.Field<string>
          path="config.read"
          label="Read From"
          padHelpText={!task?.snapshot}
        >
          {(p) => <Channel.SelectMultiple {...p} />}
        </Form.Field>
        <Form.Field<string>
          path="config.write"
          label="Write To"
          padHelpText={!task?.snapshot}
        >
          {(p) => <Channel.SelectMultiple {...p} />}
        </Form.Field>
      </Form.Form>
      <Controls
        layoutKey={layoutKey}
        state={taskState}
        snapshot={task?.snapshot}
        startingOrStopping={start.isPending || taskState?.variant === "success"}
        configuring={configure.isPending}
        onConfigure={configure.mutate}
        onStartStop={start.mutate}
      />
    </Align.Space>
  );
};

export const Configure = wrapTaskLayout(Wrapped, ZERO_PAYLOAD);
