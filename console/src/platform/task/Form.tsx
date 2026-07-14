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
  DisconnectedError,
  type rack,
  type Synnax,
  type task,
} from "@synnaxlabs/client";
import {
  Flex,
  type Flux,
  Form as PForm,
  Input,
  Status,
  Synnax as PSynnax,
  Task as PTask,
} from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { type FC, useCallback } from "react";
import { type z } from "zod";

import { CSS } from "@/platform/css";
import { Controls } from "@/platform/task/controls";
import { DriftBadge } from "@/platform/task/DriftBadge";
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
}

export interface FormTabProps {
  taskKey: task.Key;
}

export interface Forms extends Record<string, FC<FormTabProps>> {}

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
        <Flex.Box x align="center" gap="small">
          <DriftBadge />
          <Rack />
        </Flex.Box>
      </Flex.Box>
    </Flex.Box>
    {!isSnapshot && <ParentRangeButton />}
  </>
);

const SET_OPTIONS: PForm.SetOptions = { notifyOnChange: false };

export const wrapForm = <S extends task.Schemas = task.Schemas>({
  Properties,
  Form,
  schemas,
  type,
  getInitialValues,
  onConfigure,
  showHeader = true,
  showControls = true,
}: WrapFormParams<S>): FC<FormTabProps> => {
  const Wrapped: FC<FormTabProps> = ({ taskKey }) => {
    const client = PSynnax.use();
    const handleError = Status.useErrorHandler();
    const { form, status, saveAsync } = PTask.createForm({
      schemas,
      initialValues: getInitialValues({}),
    })({ query: { key: taskKey }, autoSave: true });

    // Deploy pipeline: resolve channels and rack through onConfigure, persist
    // the row, then issue the start command so the driver picks it up.
    const handleDeploy = useCallback(() => {
      handleError(async () => {
        if (client == null) throw new DisconnectedError();
        const { config, name } = form.value();
        const [newConfig, newRack] = await onConfigure(client, config, name);
        form.set("config", newConfig, SET_OPTIONS);
        if (primitive.isNonZero(newRack)) form.set("rack", newRack, SET_OPTIONS);
        if (!(await saveAsync())) return;
        await client.tasks.executeCommand({ task: taskKey, type: "start" });
      }, "Failed to start task");
    }, [client, form, saveAsync, taskKey, handleError]);

    const handleStop = useCallback(() => {
      handleError(async () => {
        if (client == null) throw new DisconnectedError();
        await client.tasks.executeCommand({ task: taskKey, type: "stop" });
      }, "Failed to stop task");
    }, [client, taskKey, handleError]);

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
              <Form status={status} {...form} />
            </Flex.Box>
            {showControls && (
              <Controls.Controls
                formStatus={status}
                onDeploy={handleDeploy}
                onStop={handleStop}
              />
            )}
          </PForm.Form>
        </Flex.Box>
      </Flex.Box>
    );
  };
  Wrapped.displayName = `Form(${Form.displayName ?? Form.name})`;
  return Wrapped;
};
