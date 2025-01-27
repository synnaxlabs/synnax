// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/common/task/Form.css";

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
import { type FC, type ReactNode } from "react";
import { z } from "zod";

import { CSS } from "@/css";
import { Controls } from "@/hardware/common/task/Controls";
import { ParentRangeButton } from "@/hardware/common/task/ParentRangeButton";
import {
  type ConfigSchema,
  type TaskProps,
  wrap,
  type WrapOptions,
} from "@/hardware/common/task/Task";
import { useCreate } from "@/hardware/common/task/useCreate";
import { useDesiredState } from "@/hardware/common/task/useDesiredState";
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
  isSnapshot: boolean;
  isRunning: boolean;
}

export interface WrapFormOptions<
  C extends UnknownRecord = UnknownRecord,
  D extends BaseStateDetails = BaseStateDetails,
  T extends string = string,
> {
  configSchema: ConfigSchema<C>;
  type: T;
  zeroPayload: WrapOptions<C, D, T>["zeroPayload"];
  onConfigure: (client: Synnax, config: C, taskKey: clientTask.Key) => Promise<C>;
}

const nameZ = z.string().min(1, "Name is required");

export const wrapForm = <
  C extends UnknownRecord = UnknownRecord,
  D extends BaseStateDetails = BaseStateDetails,
  T extends string = string,
>(
  Properties: ReactNode,
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
    const [state, setState] = useDesiredState(task?.key, task?.state ?? undefined);
    const configureMutation = useMutation({
      mutationFn: async () => {
        if (client == null) throw new Error("Client not found");
        if (!(await methods.validateAsync())) return;
        const { config, name } = methods.value();
        if (config == null) throw new Error("Config is required");
        const newConfig = await onConfigure(client, config, task.key);
        methods.set("config", newConfig);
        // current work around for Pluto form issues
        if ("channels" in newConfig) methods.set("config.channels", newConfig.channels);
        await createTask({ key: task?.key, name, type, config: newConfig });
        setState("paused");
      },
      onError: (e) => handleException(e, "Failed to configure task"),
    });
    const startOrStopMutation = useMutation({
      mutationFn: async () => {
        if (!(task instanceof clientTask.Task))
          throw new Error("Task has not been configured");
        if (state.state === "loading")
          throw new Error("State is loading, should not be able to start or stop task");
        await task.executeCommand(state.state === "running" ? "stop" : "start");
      },
      onError: (e) =>
        handleException(
          e,
          `Failed to ${state.state === "running" ? "stop" : state.state === "paused" ? "start" : "start or stop"} task`,
        ),
    });
    const isSnapshot = task.snapshot ?? false;
    return (
      <Align.Space direction="y" className={CSS.B("task-configure")} grow empty>
        <Align.Space grow>
          <PForm.Form {...methods} mode={isSnapshot ? "preview" : "normal"}>
            <Align.Space direction="x" justify="spaceBetween">
              <PForm.Field<string> path="name">
                {(p) => <Input.Text variant="natural" level="h2" {...p} />}
              </PForm.Field>
              {/* TODO: Add copy buttons */}
            </Align.Space>
            <ParentRangeButton key={task.key} />
            <Align.Space direction="x" className={CSS.B("task-properties")}>
              <Align.Space direction="x" grow>
                {Properties}
              </Align.Space>
            </Align.Space>
            <Align.Space
              direction="x"
              className={CSS.B("task-channel-form-container")}
              bordered
              rounded
              grow
              empty
            >
              <Form
                methods={methods}
                task={task}
                isRunning={state.state === "running"}
                isSnapshot={isSnapshot}
              />
            </Align.Space>
          </PForm.Form>
          <Controls
            layoutKey={layoutKey}
            state={state}
            isConfiguring={configureMutation.isPending}
            onStartStop={startOrStopMutation.mutate}
            onConfigure={configureMutation.mutate}
            isSnapshot={isSnapshot}
          />
        </Align.Space>
      </Align.Space>
    );
  };
  Wrapper.displayName = `Form(${Form.displayName ?? Form.name})`;
  return wrap(Wrapper, { zeroPayload, configSchema });
};
