// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/task/Form.css";

import { type device, panel, type rack, type Synnax, task } from "@synnaxlabs/client";
import {
  Device,
  Flex,
  type Flux,
  Form as PForm,
  Input,
  Panel as PlutoPanel,
  Task as PTask,
  Text,
} from "@synnaxlabs/pluto";
import { id, primitive, TimeStamp } from "@synnaxlabs/x";
import { type FC, useCallback, useEffect, useMemo, useRef } from "react";
import { z } from "zod";

import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Panel } from "@/platform/panel";
import { Controls } from "@/platform/task/controls";
import { ParentRangeButton } from "@/platform/task/ParentRangeButton";
import { Rack } from "@/platform/task/Rack";
import { useStatus } from "@/platform/task/useStatus";
import { UtilityButtons } from "@/platform/task/UtilityButtons";

export interface OnConfigure<Config extends z.ZodType = z.ZodType> {
  (
    client: Synnax,
    config: z.infer<Config>,
    name: string,
  ): Promise<[z.infer<Config>, rack.Key]>;
}

export const formParamsZ = z.object({
  deviceKey: z.string().optional(),
  taskKey: z.string().optional(),
  rackKey: z.number().optional(),
  config: z.unknown().optional(),
  // name is the draft task's semantic name, mirrored from the form's name field so the
  // tab title reflects it before the task is persisted. Absent once taskKey is set: the
  // name is then read from the cluster record.
  name: z.string().optional(),
});

export interface FormViewParams extends z.infer<typeof formParamsZ> {}

const useFormArgs = PlutoPanel.createSelectTabArgs(formParamsZ);

export interface getInitialValuesParams {
  deviceKey?: device.Key;
  config?: unknown;
}

export interface GetInitialValues<S extends task.Schemas = task.Schemas> {
  (params: getInitialValuesParams): PTask.InitialValues<S>;
}

export interface FormProps<
  S extends task.Schemas = task.Schemas,
> extends PForm.UseReturn<PTask.FormSchema<S>> {
  status: Flux.Result<undefined>["status"];
  onConfigure: () => void;
}

export interface WrapFormParams<S extends task.Schemas = task.Schemas> {
  Properties?: FC<{}>;
  Form: FC<FormProps<S>>;
  /** Vendor-specific icon shown on the task's tab name and toolbar button. */
  Icon: Panel.TabIcon;
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
  Icon: TabIcon,
  schemas,
  type,
  getInitialValues,
  onConfigure,
  showHeader = true,
  showControls = true,
}: WrapFormParams<S>): Panel.Tab => {
  const defaultName = getInitialValues({}).name;
  const useSyncName = (
    form: PForm.ContextValue<PTask.FormSchema<S>>,
    { deviceKey, taskKey, rackKey, config, name }: FormViewParams,
  ): void => {
    const { set } = form;
    const setView = PlutoPanel.useSetCurrentTabView();
    const formName = PForm.useFieldValue<string>("name", { ctx: form });
    const prevName = useRef(name);
    useEffect(() => {
      if (taskKey != null) return;
      const argsEdited = name !== prevName.current && name != null;
      prevName.current = name;
      if (formName === name) return;
      if (argsEdited) set("name", name);
      else
        setView(
          panel.viewZ.parse({
            type,
            args: { deviceKey, rackKey, config, name: formName },
          }),
        );
    }, [taskKey, name, formName, deviceKey, rackKey, config, set, setView]);
  };
  const Content: Panel.Content = () => {
    const { deviceKey, taskKey, rackKey, config, name } = useFormArgs();
    const setView = PlutoPanel.useSetCurrentTabView();
    const initialValues = useMemo(() => {
      const base = getInitialValues({ deviceKey, config });
      return {
        ...base,
        name: name ?? base.name,
        key: taskKey,
        rackKey: rackKey ?? (taskKey == null ? 0 : task.rackKey(taskKey)),
      };
    }, [deviceKey, config, name, taskKey, rackKey]);
    const confirm = Modals.useConfirm();
    const { form, status, save } = PTask.createForm({ schemas, initialValues })({
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
      afterSave: (props) => {
        const { key } = props.value();
        if (key == null) return;
        setView(panel.viewZ.parse({ type, args: { taskKey: key } }));
      },
    });
    Device.useRetrieveEffect({
      onChange: (d) => form.set("rackKey", d.data?.rack),
      query: deviceKey == null ? undefined : { key: deviceKey },
    });

    const isSnapshot = useIsSnapshot<PTask.FormSchema<S>>(form);
    useSyncName(form, { deviceKey, taskKey, rackKey, config, name });
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
  const RemoteName = ({ taskKey }: { taskKey: task.Key }) => {
    const tabKey = PlutoPanel.useTabKey();
    const isEditTarget = Panel.useIsNameEditTarget();
    PTask.useEnsureRetrieved({ key: taskKey });
    const name = PTask.useSelectName({ key: taskKey });
    const { update } = PTask.useRename();
    const handleChange = useCallback(
      (name: string) => update({ key: taskKey, name }),
      [taskKey, update],
    );
    return (
      <>
        <TabIcon />
        <Text.Editable
          id={isEditTarget ? Panel.tabNameID(tabKey) : undefined}
          value={name}
          onChange={handleChange}
        />
      </>
    );
  };
  const LocalName = () => {
    const tabKey = PlutoPanel.useTabKey();
    const isEditTarget = Panel.useIsNameEditTarget();
    const { deviceKey, rackKey, config, name } = useFormArgs();
    const setView = PlutoPanel.useSetCurrentTabView();
    const handleChange = useCallback(
      (name: string) =>
        setView(
          panel.viewZ.parse({ type, args: { deviceKey, rackKey, config, name } }),
        ),
      [deviceKey, rackKey, config, setView],
    );
    return (
      <>
        <TabIcon />
        <Text.Editable
          id={isEditTarget ? Panel.tabNameID(tabKey) : undefined}
          value={name ?? defaultName}
          onChange={handleChange}
        />
      </>
    );
  };
  const Name: Panel.TabName = () => {
    const { taskKey } = useFormArgs();
    if (taskKey != null) return <RemoteName taskKey={taskKey} />;
    return <LocalName />;
  };
  return { Content, Name, Icon: TabIcon };
};
