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
import { TimeSpan, type UnknownRecord } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { type FC, useEffect } from "react";
import { useDispatch } from "react-redux";
import { z } from "zod";

import { CSS } from "@/css";
import { NULL_CLIENT_ERROR } from "@/errors";
import { Controls } from "@/hardware/common/task/Controls";
import { CopyButtons } from "@/hardware/common/task/CopyButtons";
import { ParentRangeButton } from "@/hardware/common/task/ParentRangeButton";
import { Rack } from "@/hardware/common/task/Rack";
import {
  type ConfigSchema,
  type TaskProps,
  wrap,
  type WrapOptions,
} from "@/hardware/common/task/Task";
import { RUNNING_STATUS, type StartOrStopCommand } from "@/hardware/common/task/types";
import { useCreate } from "@/hardware/common/task/useCreate";
import { type StateDetails, useState } from "@/hardware/common/task/useState";
import { Layout } from "@/layout";

export type Schema<Config extends UnknownRecord = UnknownRecord> = z.ZodObject<{
  name: z.ZodString;
  config: ConfigSchema<Config>;
}>;

export type FormProps<
  Config extends UnknownRecord = UnknownRecord,
  Details extends StateDetails = StateDetails,
  Type extends string = string,
> = { methods: PForm.ContextValue<Schema<Config>> } & (
  | {
      configured: false;
      task: task.Payload<Config, Details, Type>;
      isSnapshot: false;
      isRunning: false;
    }
  | ({ configured: true; task: task.Task<Config, Details, Type> } & (
      | { isSnapshot: false; isRunning: boolean }
      | { isSnapshot: true; isRunning: false }
    ))
);

export interface OnConfigure<Config extends UnknownRecord = UnknownRecord> {
  (
    client: Synnax,
    config: Config,
    taskKey: task.Key,
    name: string,
  ): Promise<[Config, rack.Key]>;
}

export interface WrapFormArgs<
  Config extends UnknownRecord = UnknownRecord,
  Details extends StateDetails = StateDetails,
  Type extends string = string,
> extends WrapOptions<Config, Details, Type> {
  Properties: FC<{}>;
  Form: FC<FormProps<Config, Details, Type>>;
  type: Type;
  onConfigure: OnConfigure<Config>;
}

const nameZ = z.string().min(1, "Name is required");

export const wrapForm = <
  Config extends UnknownRecord = UnknownRecord,
  Details extends StateDetails = StateDetails,
  Type extends string = string,
>({
  Properties,
  Form,
  configSchema,
  type,
  getInitialPayload,
  onConfigure,
}: WrapFormArgs<Config, Details, Type>): Layout.Renderer => {
  const schema = z.object({ name: nameZ, config: configSchema });
  const Wrapper = ({
    layoutKey,
    task: tsk,
    configured,
  }: TaskProps<Config, Details, Type>) => {
    const client = PSynnax.use();
    const handleError = Status.useErrorHandler();
    const values = { name: tsk.name, config: tsk.config };
    const methods = PForm.use<Schema<Config>>({ schema, values });
    const create = useCreate<Config, Details, Type>(layoutKey);
    const dispatch = useDispatch();
    const name = Layout.useSelectName(layoutKey);
    useEffect(() => {
      if (name != null) methods.set("name", name);
    }, [name]);
    const [state, triggerLoading, triggerError] = useState(
      tsk.key,
      tsk.state ?? undefined,
    );
    const configureMutation = useMutation({
      mutationFn: async () => {
        if (client == null) throw NULL_CLIENT_ERROR;
        if (tsk.snapshot) return;
        if (!(await methods.validateAsync())) return;
        const { config, name } = methods.value();
        if (config == null) throw new Error("Config is required");
        const [newConfig, rackKey] = await onConfigure(client, config, tsk.key, name);
        methods.set("config", newConfig);
        // current work around for Pluto form issues (Issue: SY-1465)
        if ("channels" in newConfig) methods.set("config.channels", newConfig.channels);
        dispatch(Layout.rename({ key: layoutKey, name }));
        await create({ key: tsk.key, name, type, config: newConfig }, rackKey);
      },
      onError: (e) => handleError(e, `Failed to configure ${values.name}`),
    });
    const startOrStopMutation = useMutation({
      mutationFn: async (command: StartOrStopCommand) => {
        if (!configured) throw new UnexpectedError("Task has not been configured");
        triggerLoading();
        try {
          await tsk.executeCommandSync(command, {}, TimeSpan.fromSeconds(10));
        } catch (e) {
          if (e instanceof Error) triggerError(e.message);
          throw e;
        }
      },
      onError: (e, command) => handleError(e, `Failed to ${command} task`),
    });
    const isSnapshot = configured ? tsk.snapshot : false;
    const isRunning =
      configured && !isSnapshot ? state.status === RUNNING_STATUS : false;
    const formProps = {
      methods,
      configured,
      task: tsk,
      isSnapshot,
      isRunning,
    } as FormProps<Config, Details, Type>;

    return (
      <Align.Space
        direction="y"
        className={CSS(CSS.B("task-configure"), CSS.BM("task-configure", type))}
        grow
        empty
      >
        <Align.Space grow>
          <PForm.Form {...methods} mode={isSnapshot ? "preview" : "normal"}>
            <Align.Space direction="x" justify="spaceBetween">
              <PForm.Field<string> path="name">
                {(p) => <Input.Text variant="natural" level="h2" {...p} />}
              </PForm.Field>
              <Align.Space align="end" size="small">
                <CopyButtons
                  getConfig={() => methods.get("config").value}
                  getName={() => methods.get<string>("name").value}
                  taskKey={tsk.key}
                />
                <Rack taskKey={tsk.key} />
              </Align.Space>
            </Align.Space>
            {configured && isSnapshot && (
              <ParentRangeButton<Config, Details, Type> task={tsk} />
            )}
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
              <Form {...formProps} />
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
