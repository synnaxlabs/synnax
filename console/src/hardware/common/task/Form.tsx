// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax, task as clientTask } from "@synnaxlabs/client";
import {
  Align,
  Form as PForm,
  Input,
  Status,
  Synnax as PSynnax,
} from "@synnaxlabs/pluto";
import { type UnknownRecord } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { type FC } from "react";
import { z } from "zod";

import { Controls } from "@/hardware/common/task/Controls";
import {
  checkDesiredStateMatch,
  useDesiredState,
} from "@/hardware/common/task/desiredState";
import { ParentRangeButton } from "@/hardware/common/task/ParentRangeButton";
import { type ConfigSchema, type TaskProps, wrap } from "@/hardware/common/task/Task";
import { useCreate } from "@/hardware/common/task/useCreate";
import { useObserveState } from "@/hardware/common/task/useObserveState";
import { type Layout } from "@/layout";

type BaseStateDetails = { running: boolean };

type Schema<C extends UnknownRecord = UnknownRecord> = z.ZodObject<{
  name: z.ZodString;
  config: ConfigSchema<C>;
}>;

export interface FormProps<
  C extends UnknownRecord = UnknownRecord,
  D extends BaseStateDetails = BaseStateDetails,
  T extends string = string,
> {
  methods: PForm.ContextValue<Schema<C>>;
  task: clientTask.Task<C, D, T> | clientTask.Payload<C, D, T>;
  taskState?: clientTask.State<D>;
}

export interface WrapFormOptions<
  C extends UnknownRecord = UnknownRecord,
  D extends BaseStateDetails = BaseStateDetails,
  T extends string = string,
> {
  configSchema: ConfigSchema<C>;
  type: T;
  zeroPayload: clientTask.Payload<C, D, T>;
  onConfigure: (client: Synnax, config: C) => Promise<C>;
}

const nameZ = z.string().min(1, "Name is required");

export const wrapForm = <
  C extends UnknownRecord = UnknownRecord,
  D extends BaseStateDetails = BaseStateDetails,
  T extends string = string,
>(
  Form: FC<FormProps<C, D, T>>,
  { configSchema, type, zeroPayload, onConfigure }: WrapFormOptions<C, D, T>,
): Layout.Renderer => {
  const Wrapper = ({ layoutKey, task }: TaskProps<C, D, T>) => {
    const client = PSynnax.use();
    const handleException = Status.useExceptionHandler();
    const schema = z.object({ name: nameZ, config: configSchema });
    const values = { name: task.name, config: task.config };
    const methods = PForm.use<Schema<C>>({ schema, values });
    const createTask = useCreate<C, D, T>(layoutKey);
    const taskState = useObserveState<D>(
      methods.setStatus,
      methods.clearStatuses,
      task?.key,
      task?.state ?? undefined,
    );
    const running = taskState?.details?.running;
    const initialState =
      running === true ? "running" : running === false ? "paused" : undefined;
    const [desiredState, setDesiredState] = useDesiredState(initialState, task?.key);
    const configureMutation = useMutation({
      mutationFn: async () => {
        if (client == null) throw new Error("Client not found");
        if (!(await methods.validateAsync())) return;
        const { config, name } = methods.value();
        if (config == null) throw new Error("Config is required");
        const newConfig = await onConfigure(client, config);
        methods.set("config", newConfig);
        // current work around for Pluto form issues
        if ("channels" in newConfig) methods.set("config.channels", newConfig.channels);
        createTask({ key: task?.key, name, type, config: newConfig });
        setDesiredState("paused");
      },
      onError: (e) => handleException(e, "Failed to configure task"),
    });
    const startOrStopMutation = useMutation({
      mutationFn: async () => {
        if (!(task instanceof clientTask.Task)) return;
        const isRunning = running === true;
        setDesiredState(isRunning ? "paused" : "running");
        await task.executeCommand(isRunning ? "stop" : "start");
      },
      onError: (e) =>
        handleException(e, `Failed to ${running ? "stop" : "start"} task`),
    });
    const snapshot = task.snapshot;
    return (
      <Align.Space direction="y">
        <PForm.Form {...methods} mode={snapshot ? "preview" : "normal"}>
          <Align.Space direction="y">
            <Align.Space direction="x">
              <PForm.Field<string> path="name">
                {(p) => <Input.Text variant="natural" level="h2" {...p} />}
              </PForm.Field>
              {/* TODO: Add copy buttons */}
            </Align.Space>
            <ParentRangeButton key={task.key} />
            <Form methods={methods} task={task} taskState={taskState} />
          </Align.Space>
        </PForm.Form>
        <Controls
          layoutKey={layoutKey}
          state={taskState}
          startingOrStopping={
            startOrStopMutation.isPending ||
            (!checkDesiredStateMatch(desiredState, running) &&
              taskState?.variant === "success")
          }
          configuring={configureMutation.isPending}
          onStartStop={startOrStopMutation.mutate}
          onConfigure={configureMutation.mutate}
          snapshot={snapshot}
        />
      </Align.Space>
    );
  };
  Wrapper.displayName = `Form(${Form.displayName ?? Form.name})`;
  return wrap(Wrapper, { zeroPayload, configSchema });
};
