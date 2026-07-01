// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/service/task/Form.css";

import { task } from "@synnaxlabs/client";
import {
  Device,
  Flex,
  Form as PForm,
  Input,
  Task as PTask,
} from "@synnaxlabs/pluto";
import { id, primitive, TimeStamp } from "@synnaxlabs/x";
import { useCallback } from "react";

import { CSS } from "@/component/css";
import { Layout } from "@/component/layout";
import { useConfirm } from "@/component/modals/useConfirm";
import { Task } from "@/component/task";
import { type FormLayoutArgs, useIsSnapshot } from "@/component/task/Form";
import { UtilityButtons } from "@/service/task/UtilityButtons";
import { Session } from "@/session";

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
        <Task.Rack />
      </Flex.Box>
    </Flex.Box>
    {!isSnapshot && <Task.ParentRangeButton />}
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
}: Task.WrapFormArgs<S>): Layout.Renderer => {
  const Wrapper: Layout.Renderer = ({ layoutKey }) => {
    const store = Session.useStore();
    const { deviceKey, taskKey, rackKey, config } = Session.Layout.selectArgs<FormLayoutArgs>(
      store.getState(),
      layoutKey,
    );
    const dispatch = Session.useDispatch();
    const handleUnsavedChanges = useCallback(
      (unsavedChanges: boolean) =>
        dispatch(Session.Layout.setUnsavedChanges({ key: layoutKey, unsavedChanges })),
      [dispatch, layoutKey],
    );
    const initialValues = {
      ...getInitialValues({ deviceKey, config }),
      key: taskKey,
      rackKey: (rackKey ?? taskKey == null) ? 0 : task.rackKey(taskKey),
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
              <Task.Controls.Controls
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
