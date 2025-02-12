// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/common/task/Form.css";

import { type rack, type Synnax, type task, UnexpectedError } from "@synnaxlabs/client";
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

import { CSS } from "@/css";
import { NULL_CLIENT_ERROR } from "@/errors";
import { Controls } from "@/hardware/common/task/Controls";
import { CopyButtons } from "@/hardware/common/task/CopyButtons";
import { ParentRangeButton } from "@/hardware/common/task/ParentRangeButton";
import {
  type ConfigSchema,
  type TaskProps,
  wrap,
  type WrapOptions,
} from "@/hardware/common/task/Task";
import { useCreate } from "@/hardware/common/task/useCreate";
import { type BaseStateDetails, useState } from "@/hardware/common/task/useState";
import { type Layout } from "@/layout";

type Schema<Config extends UnknownRecord = UnknownRecord> = z.ZodObject<{
  name: z.ZodString;
  config: ConfigSchema<Config>;
}>;

export type FormProps<
  Config extends UnknownRecord = UnknownRecord,
  Details extends BaseStateDetails = BaseStateDetails,
  Type extends string = string,
> =
  | {
      methods: PForm.ContextValue<Schema<Config>>;
      task: task.Payload<Config, Details, Type>;
      isSnapshot: boolean;
      isRunning: boolean;
      configured: false;
    }
  | {
      methods: PForm.ContextValue<Schema<Config>>;
      task: task.Task<Config, Details, Type>;
      isSnapshot: boolean;
      isRunning: boolean;
      configured: true;
    };

export interface OnConfigure<Config extends UnknownRecord = UnknownRecord> {
  (
    client: Synnax,
    config: Config,
    taskKey: task.Key,
    name: string,
  ): Promise<[Config, rack.Key]>;
}

export interface WrapFormOptions<
  Config extends UnknownRecord = UnknownRecord,
  Details extends BaseStateDetails = BaseStateDetails,
  Type extends string = string,
> extends WrapOptions<Config, Details, Type> {
  type: Type;
  onConfigure: OnConfigure<Config>;
}

const nameZ = z.string().min(1, "Name is required");

export const wrapForm = <
  Config extends UnknownRecord = UnknownRecord,
  Details extends BaseStateDetails = BaseStateDetails,
  Type extends string = string,
>(
  Properties: FC,
  Form: FC<FormProps<Config, Details, Type>>,
  {
    configSchema,
    type,
    getInitialPayload,
    onConfigure,
  }: WrapFormOptions<Config, Details, Type>,
): Layout.Renderer => {
  const schema = z.object({ name: nameZ, config: configSchema });
  const Wrapper = ({
    layoutKey,
    task: tsk,
    configured,
  }: TaskProps<Config, Details, Type>) => {
    const client = PSynnax.use();
    const handleException = Status.useExceptionHandler();
    const values = { name: tsk.name, config: tsk.config };
    const methods = PForm.use<Schema<Config>>({ schema, values });
    const create = useCreate<Config, Details, Type>(layoutKey);
    const [state, setState] = useState(tsk.key, tsk.state ?? undefined);
    const configureMutation = useMutation({
      mutationFn: async () => {
        if (client == null) throw NULL_CLIENT_ERROR;
        if (!(await methods.validateAsync())) return;
        const { config, name } = methods.value();
        if (config == null) throw new Error("Config is required");
        const [newConfig, rackKey] = await onConfigure(client, config, tsk.key, name);
        methods.set("config", newConfig);
        // current work around for Pluto form issues (Issue: SY-1465)
        if ("channels" in newConfig) methods.set("config.channels", newConfig.channels);

        await create({ key: tsk.key, name, type, config: newConfig }, rackKey);
        setState("paused");
      },
      onError: (e) => handleException(e, `Failed to configure ${values.name}`),
    });
    const startOrStopMutation = useMutation({
      mutationFn: async () => {
        if (!configured) throw new UnexpectedError("Task has not been configured");
        if (state.state === "loading")
          throw new Error("State is loading, should not be able to start or stop task");
        await tsk.executeCommand(state.state === "running" ? "stop" : "start");
      },
      onError: (e) =>
        handleException(
          e,
          `Failed to ${state.state === "running" ? "stop" : state.state === "paused" ? "start" : "start or stop"} task`,
        ),
    });
    const isSnapshot = tsk.snapshot ?? false;
    return (
      <Align.Space direction="y" className={CSS.B("task-configure")} grow empty>
        <Align.Space grow>
          <PForm.Form {...methods} mode={isSnapshot ? "preview" : "normal"}>
            <Align.Space direction="x" justify="spaceBetween">
              <PForm.Field<string> path="name">
                {(p) => <Input.Text variant="natural" level="h2" {...p} />}
              </PForm.Field>
              <CopyButtons
                getConfig={() => methods.get("config").value}
                getName={() => methods.get<string>("name").value}
                taskKey={tsk.key}
              />
            </Align.Space>
            {configured && <ParentRangeButton<Config, Details, Type> task={tsk} />}
            <Align.Space className={CSS.B("task-properties")} direction="x">
              <Properties />
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
                task={tsk as task.Task<Config, Details, Type>}
                isRunning={state.state === "running"}
                isSnapshot={isSnapshot}
                configured={configured as true}
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
            configured={configured}
          />
        </Align.Space>
      </Align.Space>
    );
  };
  Wrapper.displayName = `Form(${Form.displayName ?? Form.name})`;
  return wrap(Wrapper, { getInitialPayload, configSchema });
};
