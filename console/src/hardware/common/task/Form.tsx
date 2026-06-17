// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/common/task/Form.css";

import { device, panel, rack, type Synnax, task } from "@synnaxlabs/client";
import {
  Device,
  Flex,
  type Flux,
  Form as PForm,
  type Icon,
  Input,
  Panel,
  Task,
} from "@synnaxlabs/pluto";
import { primitive, record, TimeStamp } from "@synnaxlabs/x";
import { type FC, useCallback } from "react";
import { z } from "zod";

import { CSS } from "@/css";
import { Controls } from "@/hardware/common/task/controls";
import { ParentRangeButton } from "@/hardware/common/task/ParentRangeButton";
import { Rack } from "@/hardware/common/task/Rack";
import { useStatus } from "@/hardware/common/task/useStatus";
import { UtilityButtons } from "@/hardware/common/task/UtilityButtons";
import { useConfirm } from "@/modals/Confirm";
import { type Tabs } from "@/panel/tabs/index";

export interface OnConfigure<Config extends z.ZodType = z.ZodType> {
  (
    client: Synnax,
    config: z.infer<Config>,
    name: string,
  ): Promise<[z.infer<Config>, rack.Key]>;
}

export interface getInitialValuesArgs {
  deviceKey?: device.Key;
  config?: unknown;
}

export interface GetInitialValues<S extends task.Schemas = task.Schemas> {
  (args: getInitialValuesArgs): Task.InitialValues<S>;
}

export interface FormProps<
  S extends task.Schemas = task.Schemas,
> extends PForm.UseReturn<Task.FormSchema<S>> {
  status: Flux.Result<undefined>["status"];
  onConfigure: () => void;
}

export interface WrapFormArgs<S extends task.Schemas = task.Schemas> {
  Properties?: FC<{}>;
  Form: FC<FormProps<S>>;
  type: z.infer<S["type"]>;
  icon: Icon.ReactElement;
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

const formArgsZ = z
  .object({
    deviceKey: device.keyZ.optional(),
    taskKey: task.keyZ.optional(),
    rackKey: rack.keyZ.optional(),
    config: record.unknownZ().optional(),
    name: z.string().default("New Task"),
  })
  .prefault({});

const useArgs = Panel.createSelectContextTabArgs(formArgsZ);

export const wrapForm = <S extends task.Schemas = task.Schemas>({
  Properties,
  Form,
  schemas,
  type,
  getInitialValues,
  onConfigure,
  icon,
  showHeader = true,
  showControls = true,
}: WrapFormArgs<S>): Tabs.Renderer => {
  const Content: Tabs.Content = () => {
    const panelDispatch = Panel.useSingleDispatch();
    const tabKey = Panel.useTabKey("");
    const { deviceKey, taskKey, rackKey, config } = useArgs();
    const initialValues = {
      ...getInitialValues({ deviceKey, config }),
      key: taskKey,
      rackKey: (rackKey ?? taskKey == null) ? 0 : task.rackKey(taskKey),
    };
    const confirm = useConfirm();
    const { form, status, save } = Task.createForm({ schemas, initialValues })({
      query: { key: taskKey },
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
        panelDispatch([
          panel.setTabArgs({ key: tabKey, args: { name, taskKey: key } }),
        ]);
      },
    });
    Device.useRetrieveEffect({
      onChange: (d) => form.set("rackKey", d.data?.rack),
      query: deviceKey == null ? undefined : { key: deviceKey },
    });

    const isSnapshot = useIsSnapshot<Task.FormSchema<S>>(form);
    return (
      <Flex.Box
        y
        className={CSS(CSS.B("task-configure"), CSS.BM("task-configure", type))}
        grow
        empty
      >
        <Flex.Box grow>
          <PForm.Form<Task.FormSchema<S>>
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
              <Form status={status} onConfigure={save} {...form} />
            </Flex.Box>
            {showControls && (
              <Controls.Controls formStatus={status} onConfigure={save} />
            )}
          </PForm.Form>
        </Flex.Box>
      </Flex.Box>
    );
  };
  Content.displayName = `Form(${Form.displayName ?? Form.name})`;
  const PersistedName = ({
    taskKey: key,
    ...rest
  }: Panel.MosaicTabNameRenderProps & { taskKey: task.Key }) => {
    const name = Task.useRetrieveName({ key });
    const { update } = Task.useRename();
    const onRename = useCallback(
      (name: string) => update({ key, name }),
      [key, update],
    );
    return (
      <Panel.DefaultTabName icon={icon} name={name} onRename={onRename} {...rest} />
    );
  };
  const Name: Tabs.Name = (props) => {
    const { taskKey, name } = useArgs();
    if (taskKey != null) return <PersistedName {...props} taskKey={taskKey} />;
    return <Panel.DefaultTabName icon={icon} name={name} {...props} />;
  };
  Name.displayName = `Name(${Form.displayName ?? Form.name}}`;
  return { Content, Name };
};
