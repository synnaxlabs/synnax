// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/common/task/Form.css";

import {
  DisconnectedError,
  type rack,
  type Synnax,
  task,
  UnexpectedError,
} from "@synnaxlabs/client";
import {
  Align,
  Form as PForm,
  Input,
  Status,
  Synnax as PSynnax,
} from "@synnaxlabs/pluto";
import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { type UseMutateFunction, useMutation } from "@tanstack/react-query";
import { type FC, useCallback, useEffect, useState as useReactState } from "react";
import { useDispatch } from "react-redux";
import { z } from "zod";

import { CSS } from "@/css";
import { Controls } from "@/hardware/common/task/Controls";
import { ParentRangeButton } from "@/hardware/common/task/ParentRangeButton";
import { Rack } from "@/hardware/common/task/Rack";
import { type TaskProps, wrap, type WrapOptions } from "@/hardware/common/task/Task";
import { type Command } from "@/hardware/common/task/types";
import { useCreate } from "@/hardware/common/task/useCreate";
import { useStatus } from "@/hardware/common/task/useStatus";
import { UtilityButtons } from "@/hardware/common/task/UtilityButtons";
import { Layout } from "@/layout";
import { useConfirm } from "@/modals/Confirm";

export type FormSchema<Config extends z.ZodType = z.ZodType> = z.ZodObject<{
  name: z.ZodString;
  config: Config;
}>;

export type FormProps<
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
> = { methods: PForm.ContextValue<FormSchema<Config>> } & (
  | {
      configured: false;
      task: task.Payload<Type, Config, StatusData>;
      isSnapshot: false;
      isRunning: false;
    }
  | ({ configured: true; task: task.Task<Type, Config, StatusData> } & (
      | { isSnapshot: false; isRunning: boolean }
      | { isSnapshot: true; isRunning: false }
    ))
);

const COMMAND_MESSAGES: Record<Command, string> = {
  start: "Starting task",
  stop: "Stopping task",
};

export interface OnConfigure<Config extends z.ZodType = z.ZodType> {
  (
    client: Synnax,
    config: z.infer<Config>,
    name: string,
  ): Promise<[z.infer<Config>, rack.Key]>;
}

export interface WrapFormArgs<
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
> extends WrapOptions<Type, Config, StatusData> {
  Properties: FC<{}>;
  Form: FC<FormProps<Type, Config, StatusData>>;
  type: z.infer<Type>;
  onConfigure: OnConfigure<Config>;
}

export interface UseFormArgs<
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
> extends TaskProps<Type, Config, StatusData>,
    Pick<WrapFormArgs<Type, Config, StatusData>, "schemas" | "onConfigure" | "type"> {}

export interface UseFormReturn<
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
> {
  formProps: FormProps<Type, Config, StatusData>;
  handleConfigure: UseMutateFunction<void, Error, void, unknown>;
  handleStartOrStop: UseMutateFunction<void, Error, Command, unknown>;
  status: task.Status<StatusData>;
  isConfiguring: boolean;
}

const nameZ = z.string().min(1, "Name is required");

const DEFAULT_STATUS: task.Status<z.ZodType> = {
  key: "",
  variant: "disabled",
  message: "Task is not configured",
  time: TimeStamp.now(),
  details: { running: false, task: "", data: {} },
};

export const useForm = <
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
>({
  task: initialTask,
  layoutKey,
  onConfigure,
  type,
  schemas,
}: UseFormArgs<Type, Config, StatusData>): UseFormReturn<Type, Config, StatusData> => {
  const schema = z.object({ name: nameZ, config: schemas.configSchema });
  const client = PSynnax.use();
  const handleError_ = Status.useErrorHandler();
  const dispatch = useDispatch();
  const handleUnsavedChanges = useCallback(
    (hasUnsavedChanges: boolean) => {
      dispatch(
        Layout.setUnsavedChanges({ key: layoutKey, unsavedChanges: hasUnsavedChanges }),
      );
    },
    [dispatch, layoutKey],
  );
  const methods = PForm.use<FormSchema<Config>>({
    schema,
    values: {
      name: initialTask.name,
      config: initialTask.config,
    } as z.infer<FormSchema<Config>>,
    onHasTouched: handleUnsavedChanges,
  });
  const create = useCreate<Type, Config, StatusData>(layoutKey, schemas);
  const name = Layout.useSelectName(layoutKey);
  useEffect(() => {
    if (name != null) methods.set("name", name);
  }, [name]);
  const [task_, setTask_] = useReactState(initialTask);
  const configured = task_.key.length > 0;
  const { status, triggerError, triggerLoading } = useStatus<StatusData>(
    task_.key,
    initialTask.status ?? (DEFAULT_STATUS as task.Status<StatusData>),
    COMMAND_MESSAGES,
  );
  const handleError = (e: Error, action: string) => {
    triggerError(e.message);
    handleError_(e, `Failed to ${action} ${methods.get<string>("name").value}`);
  };

  const confirm = useConfirm();

  const { mutate: handleConfigure, isPending: isConfiguring } = useMutation({
    mutationFn: async () => {
      if (client == null) throw new DisconnectedError();
      if (initialTask.snapshot) return;
      if (!(await methods.validateAsync())) return;
      const { name, config } = methods.value() as {
        name: string;
        config: z.infer<Config>;
      };
      if (config == null) throw new Error("Config is required");
      const [newConfig, rackKey] = await onConfigure(
        client,
        config as z.infer<Config>,
        name,
      );
      if (task_.key != "" && rackKey != task.getRackKey(task_.key)) {
        const confirmed = await confirm({
          message: "Device has been moved to different driver.",
          description:
            "This means that the task will need to be deleted and recreated on the new driver. Do you want to continue?",
          confirm: { label: "Confirm", variant: "error" },
          cancel: { label: "Cancel" },
        });
        if (!confirmed) return;
        await client.hardware.tasks.delete(BigInt(task_.key));
      }

      methods.setCurrentStateAsInitialValues();
      methods.set("config", newConfig);
      // current work around for Pluto form issues (Issue: SY-1465)
      if ("channels" in (newConfig as { channels: any }))
        methods.set("config.channels", (newConfig as { channels: any }).channels);
      dispatch(Layout.rename({ key: layoutKey, name }));
      const t = await create(
        { key: task_.key, name, type, config: newConfig as z.infer<Config> },
        rackKey,
      );
      setTask_(t);
    },
    onError: (e: Error) => handleError(e, "configure"),
  });
  const { mutate: handleStartOrStop } = useMutation({
    mutationFn: async (command: Command) => {
      if (!configured) throw new UnexpectedError("Task has not been configured");
      triggerLoading(COMMAND_MESSAGES[command]);
      const sugaredTask = client?.hardware.tasks.sugar({
        ...initialTask,
        key: task_.key,
      });
      await sugaredTask?.executeCommandSync(command, TimeSpan.fromSeconds(10));
    },
    onError: handleError,
  });
  const isSnapshot = configured ? (initialTask.snapshot ?? false) : false;
  const isRunning =
    configured && !isSnapshot ? (status?.details.running ?? false) : false;
  const formProps = {
    methods,
    configured,
    task: task_,
    isSnapshot,
    isRunning,
  } as FormProps<Type, Config, StatusData>;
  return { formProps, handleConfigure, handleStartOrStop, status, isConfiguring };
};

export const wrapForm = <
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
>({
  Properties,
  Form,
  schemas,
  type,
  getInitialPayload,
  onConfigure,
}: WrapFormArgs<Type, Config, StatusData>): Layout.Renderer => {
  const Wrapper = ({ layoutKey, ...rest }: TaskProps<Type, Config, StatusData>) => {
    const { formProps, handleConfigure, handleStartOrStop, status, isConfiguring } =
      useForm({ ...rest, layoutKey, schemas, type, onConfigure });
    const { isSnapshot, methods, configured, task } = formProps;
    return (
      <Align.Space
        y
        className={CSS(CSS.B("task-configure"), CSS.BM("task-configure", type))}
        grow
        empty
      >
        <Align.Space grow>
          <PForm.Form<FormSchema<Config>>
            {...methods}
            mode={isSnapshot ? "preview" : "normal"}
          >
            <Align.Space x justify="spaceBetween">
              <PForm.Field<string> path="name">
                {(p) => <Input.Text variant="natural" level="h2" {...p} />}
              </PForm.Field>
              <Align.Space align="end" gap="small">
                <UtilityButtons
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
            status={status}
            isConfiguring={isConfiguring}
            onCommand={handleStartOrStop}
            onConfigure={handleConfigure}
            isSnapshot={isSnapshot}
            hasBeenConfigured={configured}
          />
        </Align.Space>
      </Align.Space>
    );
  };
  Wrapper.displayName = `Form(${Form.displayName ?? Form.name})`;
  return wrap(Wrapper, { getInitialPayload, schemas });
};
