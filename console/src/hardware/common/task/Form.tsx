// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/common/task/Form.css";

import { type rack, type Synnax, task, UnexpectedError } from "@synnaxlabs/client";
import {
  Align,
  Form as PForm,
  Input,
  Status,
  Synnax as PSynnax,
} from "@synnaxlabs/pluto";
import { status, TimeSpan, type UnknownRecord } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { type FC, useCallback, useEffect, useState as useReactState } from "react";
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
import {
  type State,
  type StateDetails,
  useState,
} from "@/hardware/common/task/useState";
import { Layout } from "@/layout";
import { useConfirm } from "@/modals/Confirm";

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
  (client: Synnax, config: Config, name: string): Promise<[Config, rack.Key]>;
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

export interface UseFormArgs<
  Config extends UnknownRecord = UnknownRecord,
  Details extends StateDetails = StateDetails,
  Type extends string = string,
> extends TaskProps<Config, Details, Type>,
    Pick<
      WrapFormArgs<Config, Details, Type>,
      "configSchema" | "onConfigure" | "type"
    > {}

export interface UseFormReturn<
  Config extends UnknownRecord = UnknownRecord,
  Details extends StateDetails = StateDetails,
  Type extends string = string,
> {
  formProps: FormProps<Config, Details, Type>;
  handleConfigure: (config: Config, name: string) => Promise<void>;
  handleStartOrStop: (command: StartOrStopCommand) => Promise<void>;
  state: State;
  isConfiguring: boolean;
}

const nameZ = z.string().min(1, "Name is required");

export const useForm = <
  Config extends UnknownRecord = UnknownRecord,
  Details extends StateDetails = StateDetails,
  Type extends string = string,
>({
  task: initialTask,
  layoutKey,
  configSchema,
  onConfigure,
  type,
}: UseFormArgs<Config, Details, Type>) => {
  const schema = z.object({ name: nameZ, config: configSchema });
  const client = PSynnax.use();
  const handleError_ = Status.useErrorHandler();
  const values = { name: initialTask.name, config: initialTask.config };
  const dispatch = useDispatch();
  const handleUnsavedChanges = useCallback(
    (hasUnsavedChanges: boolean) => {
      dispatch(
        Layout.setUnsavedChanges({ key: layoutKey, unsavedChanges: hasUnsavedChanges }),
      );
    },
    [dispatch, layoutKey],
  );
  const methods = PForm.use<Schema<Config>>({
    schema,
    values,
    onHasTouched: handleUnsavedChanges,
  });
  const create = useCreate<Config, Details, Type>(layoutKey);
  const name = Layout.useSelectName(layoutKey);
  useEffect(() => {
    if (name != null) methods.set("name", name);
  }, [name]);
  const [task_, setTask_] = useReactState(initialTask);
  const configured = task_.key.length > 0;
  const { state, triggerError, triggerLoading } = useState(
    task_.key,
    initialTask.state ?? undefined,
  );
  const handleError = (e: Error, action: string) => {
    triggerError(e.message);
    handleError_(e, `Failed to ${action} ${values.name}`);
  };

  const confirm = useConfirm();

  const { mutate: handleConfigure, isPending: isConfiguring } = useMutation({
    mutationFn: async () => {
      if (client == null) throw NULL_CLIENT_ERROR;
      if (initialTask.snapshot) return;
      if (!(await methods.validateAsync())) return;
      const { config, name } = methods.value();
      if (config == null) throw new Error("Config is required");
      const [newConfig, rackKey] = await onConfigure(client, config, name);
      if (task_.key != "" && rackKey != task.getRackKey(task_.key)) {
        const confirmed = await confirm({
          message: "Device has been moved to different driver.",
          description:
            "This means that the task will need to be deleted and recreated on the new driver. Do you want to continue?",
          confirm: { label: "Confirm", variant: status.ERROR_VARIANT },
          cancel: { label: "Cancel" },
        });
        if (!confirmed) return;
        await client.hardware.tasks.delete(BigInt(task_.key));
      }

      methods.setCurrentStateAsInitialValues();
      methods.set("config", newConfig);
      // current work around for Pluto form issues (Issue: SY-1465)
      if ("channels" in newConfig) methods.set("config.channels", newConfig.channels);
      dispatch(Layout.rename({ key: layoutKey, name }));
      const t = await create(
        { key: task_.key, name, type, config: newConfig },
        rackKey,
      );
      setTask_(t);
    },
    onError: (e: Error) => handleError(e, "configure"),
  });
  const { mutate: handleStartOrStop } = useMutation({
    mutationFn: async (command: StartOrStopCommand) => {
      if (!configured) throw new UnexpectedError("Task has not been configured");
      triggerLoading();
      const sugaredTask = client?.hardware.tasks.sugar({
        ...initialTask,
        key: task_.key,
      });
      await sugaredTask?.executeCommandSync(command, TimeSpan.fromSeconds(10));
    },
    onError: handleError,
  });
  const isSnapshot = configured ? (initialTask.snapshot ?? false) : false;
  const isRunning = configured && !isSnapshot ? state.status === RUNNING_STATUS : false;
  const formProps = {
    methods,
    configured,
    task: task_,
    isSnapshot,
    isRunning,
  } as FormProps<Config, Details, Type>;
  return { formProps, handleConfigure, handleStartOrStop, state, isConfiguring };
};

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
  const Wrapper = ({ layoutKey, ...rest }: TaskProps<Config, Details, Type>) => {
    const { formProps, handleConfigure, handleStartOrStop, state, isConfiguring } =
      useForm({
        ...rest,
        layoutKey,
        configSchema,
        type,
        onConfigure,
      });
    const { isSnapshot, methods, configured, task } = formProps;
    return (
      <Align.Space
        y
        className={CSS(CSS.B("task-configure"), CSS.BM("task-configure", type))}
        grow
        empty
      >
        <Align.Space grow>
          <PForm.Form {...methods} mode={isSnapshot ? "preview" : "normal"}>
            <Align.Space x justify="spaceBetween">
              <PForm.Field<string> path="name">
                {(p) => <Input.Text variant="natural" level="h2" {...p} />}
              </PForm.Field>
              <Align.Space align="end" size="small">
                <CopyButtons
                  getConfig={() => methods.get("config").value}
                  getName={() => methods.get<string>("name").value}
                  taskKey={task.key}
                />
                <Rack taskKey={task.key} />
              </Align.Space>
            </Align.Space>
            {configured && isSnapshot && <ParentRangeButton taskKey={task.key} />}
            <Align.Space className={CSS.B("task-properties")} x wrap>
              <Properties />
            </Align.Space>
            <Align.Space
              x
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
            isConfiguring={isConfiguring}
            onStartStop={handleStartOrStop}
            onConfigure={handleConfigure}
            isSnapshot={isSnapshot}
            hasBeenConfigured={configured}
          />
        </Align.Space>
      </Align.Space>
    );
  };
  Wrapper.displayName = `Form(${Form.displayName ?? Form.name})`;
  return wrap(Wrapper, { getInitialPayload, configSchema });
};
