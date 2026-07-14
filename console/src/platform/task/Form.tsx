// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/task/Form.css";

import {
  type device,
  panel,
  type rack,
  type Synnax,
  type task,
} from "@synnaxlabs/client";
import {
  Device,
  Flex,
  type Flux,
  Form as PForm,
  Icon,
  Input,
  Panel as PlutoPanel,
  Task as PTask,
  Text,
} from "@synnaxlabs/pluto";
import { id, primitive, TimeStamp, uuid } from "@synnaxlabs/x";
import { type FC, useCallback, useEffect, useState } from "react";
import { z } from "zod";

import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { type Panel } from "@/platform/panel";
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
  schemas,
  type,
  getInitialValues,
  onConfigure,
  showHeader = true,
  showControls = true,
}: WrapFormParams<S>): Panel.Tab => {
  const Content: Panel.Content = () => {
    const { deviceKey, taskKey, rackKey, config } = useFormArgs() ?? {};
    const setView = PlutoPanel.useSetCurrentTabView();
    const initialValues = {
      ...getInitialValues({ deviceKey, config }),
      key: taskKey,
      rackKey: rackKey ?? 0,
    };
    const confirm = Modals.useConfirm();
    const { form, status, save } = PTask.createForm({ schemas, initialValues })({
      query: { key: taskKey },
      beforeSave: async ({ client, ...form }) => {
        const { name, config, rackKey: currentRackKey } = form.value();
        const [newConfig, newRackKey] = await onConfigure(client, config, name);
        const nonZeroRackKey = primitive.isNonZero(newRackKey);
        if (
          nonZeroRackKey &&
          primitive.isNonZero(taskKey) &&
          newRackKey != currentRackKey
        ) {
          const confirmed = await confirm({
            message: "Device has been moved to different driver.",
            description:
              "This means that the task will be moved to the new driver. Do you want to continue?",
            confirm: { label: "Confirm", variant: "error" },
            cancel: { label: "Cancel" },
          });
          if (!confirmed) return false;
        }
        if (nonZeroRackKey) form.set("rackKey", newRackKey);
        form.set("config", newConfig);
        let key = form.value().key;
        if (key == null) {
          key = uuid.create();
          form.set("key", key);
        }
        const status: task.New<S>["status"] = {
          key: id.create(),
          name,
          description: "",
          time: TimeStamp.now(),
          variant: "loading",
          message: "Configuring task",
          details: { task: key, running: true, cmd: "", data: null },
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
  const Name: Panel.TabName = () => {
    const key = useFormArgs()?.taskKey;
    const isPersisted = key != null;
    const [name, setName] = useState("Task");
    const { retrieve } = PTask.useRetrieveObservableName({
      onChange: setName,
      addStatusOnFailure: false,
    });
    const { update } = PTask.useRename({
      beforeUpdate: useCallback(() => isPersisted, [isPersisted]),
    });
    useEffect(() => {
      if (key != null) retrieve({ key });
    }, [key, retrieve]);
    const handleRename = useCallback(
      (name: string) => {
        if (key != null) update({ key, name });
      },
      [key, update],
    );
    return (
      <>
        <Icon.Task />
        <Text.Editable value={name} onChange={handleRename} />
      </>
    );
  };
  return { Content, Name };
};
