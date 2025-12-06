// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/common/task/Form.css";

import { type device, type rack, type Synnax, task } from "@synnaxlabs/client";
import { Device, Flex, type Flux, Form as PForm, Input, Task } from "@synnaxlabs/pluto";
import { primitive, TimeStamp } from "@synnaxlabs/x";
import { type FC, useCallback } from "react";
import { useDispatch, useStore } from "react-redux";
import { type z } from "zod";

import { CSS } from "@/css";
import { Controls } from "@/hardware/common/task/Controls";
import { ParentRangeButton } from "@/hardware/common/task/ParentRangeButton";
import { Rack } from "@/hardware/common/task/Rack";
import { useStatus } from "@/hardware/common/task/useStatus";
import { UtilityButtons } from "@/hardware/common/task/UtilityButtons";
import { Layout } from "@/layout";
import { useConfirm } from "@/modals/Confirm";
import { type RootState } from "@/store";

export interface OnConfigure<Config extends z.ZodType = z.ZodType> {
  (
    client: Synnax,
    config: z.infer<Config>,
    name: string,
  ): Promise<[z.infer<Config>, rack.Key]>;
}
export interface FormLayoutArgs {
  deviceKey?: device.Key;
  taskKey?: task.Key;
  rackKey?: rack.Key;
  config?: unknown;
}

export interface Layout extends Layout.BaseState<FormLayoutArgs> {}

export const LAYOUT: Omit<Layout, "type"> = {
  name: "Configure",
  icon: "Task",
  location: "mosaic",
  args: {},
};

export interface getInitialValuesArgs {
  deviceKey?: device.Key;
  config?: unknown;
}

export interface GetInitialValues<
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
> {
  (args: getInitialValuesArgs): Task.InitialValues<Type, Config, StatusData>;
}

export interface FormProps<
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
> extends PForm.UseReturn<Task.FormSchema<Type, Config, StatusData>> {
  layoutKey: string;
  status: Flux.Result<undefined>["status"];
  onConfigure: () => void;
}

export interface WrapFormArgs<
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
> {
  Properties?: FC<{}>;
  Form: FC<FormProps<Type, Config, StatusData>>;
  type: z.infer<Type>;
  onConfigure: OnConfigure<Config>;
  schemas: task.Schemas<Type, Config, StatusData>;
  getInitialValues: GetInitialValues<Type, Config, StatusData>;
  showHeader?: boolean;
  showControls?: boolean;
}

export const useIsRunning = <Schema extends z.ZodType>(
  ctx?: PForm.ContextValue<Schema>,
) => useStatus(ctx)?.details.running ?? false;
export const useIsSnapshot = <Schema extends z.ZodType>(
  ctx?: PForm.ContextValue<Schema>,
) => PForm.useFieldValue<boolean>("snapshot", { ctx });

interface HeaderProps {
  isSnapshot: boolean;
}

const Header = ({ isSnapshot }: HeaderProps) => (
  <>
    <Flex.Box x justify="between">
      <PForm.Field<string> path="name">
        {(p) => <Input.Text variant="text" level="h2" onlyChangeOnBlur {...p} />}
      </PForm.Field>
      <Flex.Box align="end" gap="small">
        <UtilityButtons />
        <Rack />
      </Flex.Box>
    </Flex.Box>
    {!isSnapshot && <ParentRangeButton />}
  </>
);

export const wrapForm = <
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
>({
  Properties,
  Form,
  schemas,
  type,
  getInitialValues,
  onConfigure,
  showHeader = true,
  showControls = true,
}: WrapFormArgs<Type, Config, StatusData>): Layout.Renderer => {
  const Wrapper: Layout.Renderer = ({ layoutKey }) => {
    const store = useStore<RootState>();
    const { deviceKey, taskKey, rackKey, config } = Layout.selectArgs<FormLayoutArgs>(
      store.getState(),
      layoutKey,
    );
    const dispatch = useDispatch();
    const handleUnsavedChanges = useCallback(
      (unsavedChanges: boolean) =>
        dispatch(Layout.setUnsavedChanges({ key: layoutKey, unsavedChanges })),
      [dispatch, layoutKey],
    );
    const initialValues = {
      ...getInitialValues({ deviceKey, config }),
      key: taskKey,
      rackKey: (rackKey ?? taskKey == null) ? 0 : task.rackKey(taskKey),
    };
    const confirm = useConfirm();
    const { form, status, save } = Task.createForm({ schemas, initialValues })({
      query: { key: taskKey },
      onHasTouched: handleUnsavedChanges,
      beforeSave: async ({ client, ...form }) => {
        const { name, config } = form.value();
        const [newConfig, rackKey] = await onConfigure(client, config, name);
        if (primitive.isNonZero(taskKey) && rackKey != task.rackKey(taskKey)) {
          const confirmed = await confirm({
            message: "Device has been moved to different driver.",
            description:
              "This means that the task will need to be deleted and recreated on the new driver. Do you want to continue?",
            confirm: { label: "Confirm", variant: "error" },
            cancel: { label: "Cancel" },
          });
          if (!confirmed) return false;
          await client.tasks.delete(taskKey);
        }
        form.set("rackKey", rackKey);
        form.set("config", newConfig);
        const status: task.NewStatus = {
          name,
          time: TimeStamp.now(),
          variant: "loading",
          message: "Configuring task",
          details: { running: true, data: null },
        };
        form.set("status", status);
        return true;
      },
      afterSave: ({ client, ...form }) => {
        const { key, name } = form.value();
        if (key == null) return;
        dispatch(Layout.rename({ key: layoutKey, name }));
        dispatch(Layout.setArgs({ key: layoutKey, args: { taskKey: key } }));
        dispatch(Layout.setAltKey({ key: layoutKey, altKey: key }));
      },
    });
    Device.useRetrieveEffect({
      onChange: (d) => form.set("rackKey", d.data?.rack),
      query: deviceKey == null ? undefined : { key: deviceKey },
    });
    const name = PForm.useFieldValue<
      string,
      string,
      Task.FormSchema<Type, Config, StatusData>
    >("name", { ctx: form });
    const handleLayoutNameChange = useCallback(
      (name: string) => {
        if (status.variant !== "success") return;
        form.set("name", name);
      },
      [form.set, status?.variant],
    );
    Layout.useSyncName(layoutKey, name, handleLayoutNameChange);

    const isSnapshot = useIsSnapshot<Task.FormSchema<Type, Config, StatusData>>(form);
    return (
      <Flex.Box
        y
        className={CSS(CSS.B("task-configure"), CSS.BM("task-configure", type))}
        grow
        empty
      >
        <Flex.Box grow>
          <PForm.Form<Task.FormSchema<Type, Config, StatusData>>
            {...form}
            mode={isSnapshot ? "preview" : "normal"}
          >
            {showHeader && <Header isSnapshot={isSnapshot} />}
            {Properties != null && (
              <Flex.Box className={CSS.B("task-properties")} x wrap>
                <Properties />
              </Flex.Box>
            )}
            <Flex.Box
              x
              className={CSS.B("task-channel-form-container")}
              bordered
              rounded
              grow
              empty
            >
              <Form
                layoutKey={layoutKey}
                status={status}
                onConfigure={save}
                {...form}
              />
            </Flex.Box>
            {showControls && (
              <Controls layoutKey={layoutKey} formStatus={status} onConfigure={save} />
            )}
          </PForm.Form>
        </Flex.Box>
      </Flex.Box>
    );
  };
  Wrapper.displayName = `Form(${Form.displayName ?? Form.name})`;
  return Wrapper;
};
