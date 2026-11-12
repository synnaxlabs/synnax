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
  task,
} from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Status } from "@synnaxlabs/lyra/status";
import { Access, Synnax as PSynnax, Task as PTask } from "@synnaxlabs/pluto";
import { caseconv, deep, narrow, primitive } from "@synnaxlabs/x";
import { type FC, useEffect, useRef } from "react";
import { type z } from "zod";

import { Analytics } from "@/platform/analytics";
import { CSS } from "@/platform/css";
import { Errors } from "@/platform/errors";
import { Controls } from "@/platform/task/controls";
import { ParentRangeButton } from "@/platform/task/ParentRangeButton";
import { Rack } from "@/platform/task/Rack";
import { fieldErrorsZ, formPath } from "@/platform/task/types";
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
  client: Synnax;
  deviceKey?: device.Key;
  config?: unknown;
}

export interface GetInitialValues<S extends task.Schemas = task.Schemas> {
  (
    params: getInitialValuesParams,
  ): PTask.InitialValues<S> | Promise<PTask.InitialValues<S>>;
}

export interface FormTabProps {
  taskKey: task.Key;
}

export interface Forms extends Record<string, FC<FormTabProps>> {}

export interface WrapFormParams<S extends task.Schemas = task.Schemas> {
  Properties?: FC<{}>;
  Form: FC<{}>;
  type: z.infer<S["type"]>;
  onConfigure: OnConfigure<S["config"]>;
  schemas: S;
  /**
   * Checks the config from the first deploy on, as a check the form runs on every edit.
   * A deploy starts only when no field shows an error, so a warning never blocks one.
   * Keep the shape schemas lax: a field whose value fails its shape schema stays
   * unsent.
   */
  deployConfigZ: z.ZodType;
  showHeader?: boolean;
  showControls?: boolean;
}

export const useIsSnapshot = <Schema extends z.ZodType>(
  ctx?: PForm.ContextValue<Schema>,
) => PForm.useFieldValue(formPath.snapshot, { ctx });

const Header = () => (
  <>
    <div className={CSS.BE("task-configure", "header")}>
      <PForm.Field
        path={formPath.name}
        showLabel={false}
        padHelpText={false}
        className={CSS.BE("task-configure", "name")}
      >
        {(p) => (
          <Input.Text
            variant="text"
            level="h2"
            onlyChangeOnBlur
            aria-label="Name"
            {...p}
          />
        )}
      </PForm.Field>
      <div className={CSS.BE("task-configure", "rack")}>
        <Rack />
      </div>
      <div className={CSS.BE("task-configure", "actions")}>
        <UtilityButtons />
      </div>
    </div>
    <ParentRangeButton />
  </>
);

const CLEARED: Status.Crude = { variant: "success", message: "" };

/**
 * Shows each field error of the task status on its config field. Each new status
 * replaces the errors of the last one, and leaves a status that replaced one alone. The
 * Driver names a field by its snake_case config path, with list indexes.
 *
 * @returns a function that clears the errors.
 */
const useFieldErrors = <Z extends z.ZodType>(
  form: PForm.ContextValue<Z>,
  configZ: z.ZodType,
): (() => void) => {
  const status = useStatus(form);
  const shown = useRef<{ target: deep.Target<unknown>; message: string }[]>([]);
  const clear = () => {
    shown.current.forEach(({ target, message }) => {
      if (form.get(target, { optional: true })?.status.message === message)
        form.setStatus(target, CLEARED);
    });
    shown.current = [];
  };
  useEffect(() => {
    clear();
    const data = status?.details.data;
    if (!narrow.isObject(data) || !("errors" in data)) return;
    const values = form.value();
    fieldErrorsZ.parse(data).errors.forEach(({ path, message }) => {
      const converted = caseconv.snakeToCamelPath(path.split(deep.SEPARATOR), {
        schema: configZ,
      });
      const raw = [String(formPath.config), ...converted.path].join(deep.SEPARATOR);
      const target = deep
        .resolvePath(raw, values)
        .split(deep.SEPARATOR)
        .reduce<deep.Path<unknown>>((p, part) => p[part], deep.path<unknown>());
      form.setStatus(target, { variant: "error", message });
      shown.current.push({ target, message });
    });
  }, [status, form, configZ, clear]);
  return clear;
};

export const wrapForm = <S extends task.Schemas = task.Schemas>({
  Properties,
  Form,
  schemas,
  type,
  deployConfigZ,
  onConfigure,
  showHeader = true,
  showControls = true,
}: WrapFormParams<S>): FC<FormTabProps> => {
  const useForm = PTask.createForm({ schemas });
  const Wrapped: FC<FormTabProps> = ({ taskKey }) => {
    const client = PSynnax.use();
    const handleError = Status.useErrorHandler();
    const { capture } = Analytics.use();
    const { form, flush } = useForm({ query: { key: taskKey } });
    const clearFieldErrors = useFieldErrors<PTask.FormSchema<S>>(form, schemas.config);

    // The form saves on every edit, so a subject who cannot write the task gets it
    // read-only rather than fields that revert once the save is refused.
    const canEdit = Access.useUpdateGranted(task.resourceID(taskKey));

    const deployChecked = useRef(false);

    const handleDeploy = () => {
      handleError(async () => {
        if (client == null) throw new DisconnectedError();
        // A start replaces the field errors of the last one.
        clearFieldErrors();
        if (canEdit) {
          if (!(await flush())) return;
          if (!deployChecked.current) {
            form.addCheck(formPath.config, deployConfigZ);
            deployChecked.current = true;
          }
          await form.validateAsync();
          if (form.getStatuses().some(({ variant }) => variant === "error")) return;
          const { config, name, rack } = form.value();
          // The form's values are the cached task, and onConfigure changes its config.
          const [newConfig, newRack] = await onConfigure(
            client,
            deep.copy(config),
            name,
          );
          // Another client can edit the task while onConfigure runs, so only the
          // values onConfigure changed are written.
          form.batch(() => {
            PForm.setChanged(form, formPath.config, config, newConfig);
            if (primitive.isNonZero(newRack))
              PForm.setChanged(form, formPath.rack, rack, newRack);
          });
          if (!(await flush())) return;
          await client.tasks.settled(taskKey);
        }
        // A pinned start runs only the config this form shows. The Driver refuses it
        // when another client changed the config since.
        await client.tasks.executeCommand({
          task: taskKey,
          type: "start",
          configHash: form.value().configHash,
          pinned: true,
        });
        capture("task_started", { type });
      }, "Failed to start task");
    };

    const handleStop = () => {
      handleError(async () => {
        if (client == null) throw new DisconnectedError();
        await client.tasks.executeCommand({ task: taskKey, type: "stop" });
      }, "Failed to stop task");
    };

    const isSnapshot = useIsSnapshot<PTask.FormSchema<S>>(form);
    return (
      <Flex.Box
        y
        className={CSS.cls(CSS.B("task-configure"), CSS.BM("task-configure", type))}
        grow
        empty
      >
        <Flex.Box grow>
          <PForm.Form<PTask.FormSchema<S>>
            {...form}
            mode={isSnapshot || !canEdit ? "preview" : "normal"}
          >
            {showHeader && <Header />}
            {Properties != null && (
              <Flex.Box className={CSS.B("task-properties")} x wrap>
                <Properties />
              </Flex.Box>
            )}
            <Flex.Box
              x
              className={CSS.B("task-channel-form-container")}
              bordered
              grow
              empty
            >
              <Errors.SuspenseBoundary>
                <Form />
              </Errors.SuspenseBoundary>
            </Flex.Box>
            {showControls && (
              <Controls.Controls onDeploy={handleDeploy} onStop={handleStop} />
            )}
          </PForm.Form>
        </Flex.Box>
      </Flex.Box>
    );
  };
  Wrapped.displayName = `Form(${Form.displayName ?? Form.name})`;
  return Wrapped;
};
