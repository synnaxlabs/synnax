// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/task/Form.css";

import { type device, type rack, type Synnax, task } from "@synnaxlabs/client";
import {
  Device,
  Flex,
  type Flux,
  Form as PForm,
  Input,
  Task as PTask,
} from "@synnaxlabs/pluto";
import { id, primitive, TimeStamp } from "@synnaxlabs/x";
import { type FC, useCallback } from "react";
import { type z } from "zod";

import { CSS } from "@/platform/css";
import { type Layout } from "@/platform/layout";
import { useConfirm } from "@/platform/modals/useConfirm";
import { Controls } from "@/platform/task/controls";
import { ParentRangeButton } from "@/platform/task/ParentRangeButton";
import { Rack } from "@/platform/task/Rack";
import { useStatus } from "@/platform/task/useStatus";
import { UtilityButtons } from "@/platform/task/UtilityButtons";
import { Session } from "@/session";

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

export interface getInitialValuesArgs {
  deviceKey?: device.Key;
  config?: unknown;
}

export interface GetInitialValues<S extends task.Schemas = task.Schemas> {
  (args: getInitialValuesArgs): PTask.InitialValues<S>;
}

export interface FormProps<
  S extends task.Schemas = task.Schemas,
> extends PForm.UseReturn<PTask.FormSchema<S>> {
  layoutKey: string;
  status: Flux.Result<undefined>["status"];
  onConfigure: () => void;
}

export interface WrapFormArgs<S extends task.Schemas = task.Schemas> {
  Properties?: FC<{}>;
  Form: FC<FormProps<S>>;
  type: z.infer<S["type"]>;
  onConfigure: OnConfigure<S["config"]>;
  schemas: S;
  getInitialValues: GetInitialValues<S>;
  showHeader?: boolean;
  showControls?: boolean;
}

export const useIsRunning = <Schema extends z.ZodType>(
  ctx?: PForm.ContextValue<Schema>,
) => useStatus(ctx)?.details.running ?? false;

export const useIsSnapshot = <Schema extends z.ZodType>(
  ctx?: PForm.ContextValue<Schema>,
) => PForm.useFieldValue<boolean>("snapshot", { ctx });

export interface Layout extends Session.Layout.BaseState<FormLayoutArgs> {}

export const LAYOUT: Omit<Layout, "type"> = {
  name: "Configure",
  icon: "Task",
  location: "mosaic",
  args: {},
};

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

export const wrapForm = <S extends task.Schemas = task.Schemas>({
  Properties,
  Form,
  schemas,
  type,
  getInitialValues,
  onConfigure,
  showHeader = true,
  showControls = true,
}: WrapFormArgs<S>): Layout.Renderer => {
  const Wrapper: Layout.Renderer = ({ layoutKey }) => {
    const store = Session.useStore();
    const { deviceKey, taskKey, rackKey, config } =
      Session.Layout.selectArgs<FormLayoutArgs>(store.getState(), layoutKey);
    const dispatch = Session.useDispatch();
    const handleUnsavedChanges = useCallback(
      (unsavedChanges: boolean) =>
        dispatch(Session.Layout.setUnsavedChanges({ key: layoutKey, unsavedChanges })),
      [dispatch, layoutKey],
    );
    const initialValues = {
      ...getInitialValues({ deviceKey, config }),
      key: taskKey,
      rackKey: rackKey ?? (taskKey == null ? 0 : task.rackKey(taskKey)),
    };
    const confirm = useConfirm();
    const { form, status, save } = PTask.createForm({ schemas, initialValues })({
      query: { key: taskKey },
      onHasTouched: handleUnsavedChanges,
      beforeSave: async ({ client, ...form }) => {
        const { name, config } = form.value();
        const [newConfig, rackKey] = await onConfigure(client, config, name);
        const nonZeroRackKey = primitive.isNonZero(rackKey);
        if (
          nonZeroRackKey &&
          primitive.isNonZero(taskKey) &&
          rackKey != task.rackKey(taskKey)
        ) {
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
        if (nonZeroRackKey) form.set("rackKey", rackKey);
        form.set("config", newConfig);
        const status: task.New<S>["status"] = {
          key: id.create(),
          name,
          description: "",
          time: TimeStamp.now(),
          variant: "loading",
          message: "Configuring task",
          details: { running: true, cmd: "", data: null },
        };
        form.set("status", status);
        return true;
      },
      afterSave: ({ client, ...form }) => {
        const { key, name } = form.value();
        if (key == null) return;
        dispatch(Session.Layout.rename({ key: layoutKey, name }));
        dispatch(Session.Layout.setArgs({ key: layoutKey, args: { taskKey: key } }));
        dispatch(Session.Layout.setAltKey({ key: layoutKey, altKey: key }));
      },
    });
    Device.useRetrieveEffect({
      onChange: (d) => form.set("rackKey", d.data?.rack),
      query: deviceKey == null ? undefined : { key: deviceKey },
    });

    const isSnapshot = useIsSnapshot<PTask.FormSchema<S>>(form);
    return (
      <Flex.Box
        y
        className={CSS(CSS.B("task-configure"), CSS.BM("task-configure", type))}
        grow
        empty
      >
        <Flex.Box grow>
          <PForm.Form<PTask.FormSchema<S>>
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
              <Controls.Controls
                layoutKey={layoutKey}
                formStatus={status}
                onConfigure={save}
              />
            )}
          </PForm.Form>
        </Flex.Box>
      </Flex.Box>
    );
  };
  Wrapper.displayName = `Form(${Form.displayName ?? Form.name})`;
  Wrapper.useName = useName;
  return Wrapper;
};

const useName: Layout.UseName = (layoutKey, onChange) => {
  const args = Session.Layout.useSelectArgs<FormLayoutArgs>(layoutKey);
  const taskKey = args?.taskKey;
  const isPersisted = taskKey != null;
  const { retrieve: baseRetrieve } = PTask.useRetrieveObservableName({
    onChange,
    addStatusOnFailure: false,
  });
  const { update } = PTask.useRename({
    beforeUpdate: useCallback(() => isPersisted, [isPersisted]),
  });
  const onRename = useCallback(
    (name: string) => {
      if (taskKey != null) update({ key: taskKey, name });
    },
    [taskKey, update],
  );
  const retrieve = useCallback(() => {
    if (taskKey != null) baseRetrieve({ key: taskKey });
  }, [taskKey, baseRetrieve]);
  return { retrieve, onRename };
};
