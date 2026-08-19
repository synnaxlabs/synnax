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
  type status,
  type Synnax,
  task,
} from "@synnaxlabs/client";
import {
  Access,
  Flex,
  Form as PForm,
  Input,
  Status,
  Synnax as PSynnax,
  Task as PTask,
} from "@synnaxlabs/pluto";
import { primitive, TimeSpan } from "@synnaxlabs/x";
import { type FC, useCallback } from "react";
import { type z } from "zod";

import { CSS } from "@/platform/css";
import { Errors } from "@/platform/errors";
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

export interface getInitialValuesParams {
  deviceKey?: device.Key;
  config?: unknown;
}

export interface GetInitialValues<S extends task.Schemas = task.Schemas> {
  (params: getInitialValuesParams): PTask.InitialValues<S>;
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
   * Validates the config when the user deploys. Failures render as field
   * errors and block the start command; warning-variant issues render but
   * don't block. Shape schemas stay lax so drafts persist through autosave.
   */
  deployConfigZ: z.ZodType;
  getInitialValues: GetInitialValues<S>;
  showHeader?: boolean;
  showControls?: boolean;
}

export const useIsRunning = <Schema extends z.ZodType>(
  ctx?: PForm.ContextValue<Schema>,
) => useStatus(ctx).details.running;

export const useIsSnapshot = <Schema extends z.ZodType>(
  ctx?: PForm.ContextValue<Schema>,
) => PForm.useFieldValue<boolean>("snapshot", { ctx });

/**
 * Whether the surrounding task form renders read-only: the task is a snapshot or the
 * subject holds no update grant. Edit affordances gate on this, not on useIsSnapshot.
 */
export const useIsPreview = <Schema extends z.ZodType>(
  ctx?: PForm.ContextValue<Schema>,
) => PForm.useContext<Schema>(ctx).mode === "preview";

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

// The deploy pipeline saves once at the end; notifying would fire autosave first.
const SKIP_AUTOSAVE: PForm.SetOptions = { notifyOnChange: false };

// Numeric fields have drag handles that emit a change per pixel.
const AUTO_SAVE_DEBOUNCE = TimeSpan.milliseconds(200);

const issueVariant = (issue: z.core.$ZodIssue): status.Variant =>
  issue.code === "custom" && issue.params != null && "variant" in issue.params
    ? (issue.params.variant as status.Variant)
    : "error";

export const wrapForm = <S extends task.Schemas = task.Schemas>({
  Properties,
  Form,
  schemas,
  type,
  deployConfigZ,
  getInitialValues,
  onConfigure,
  showHeader = true,
  showControls = true,
}: WrapFormParams<S>): FC<FormTabProps> => {
  const useForm = PTask.createForm({ schemas, initialValues: getInitialValues({}) });
  const Wrapped: FC<FormTabProps> = ({ taskKey }) => {
    const client = PSynnax.use();
    const handleError = Status.useErrorHandler();
    const { form, saveAsync } = useForm({
      query: { key: taskKey },
      autoSave: true,
      autoSaveDebounce: AUTO_SAVE_DEBOUNCE,
    });

    // The form saves on every edit, so a subject who cannot write the task gets it
    // read-only rather than fields that revert once the save is refused.
    const canEdit = Access.useUpdateGranted(task.ontologyID(taskKey));

    const handleDeploy = useCallback(() => {
      handleError(async () => {
        if (client == null) throw new DisconnectedError();
        if (canEdit) {
          const { config, name } = form.value();
          const result = deployConfigZ.safeParse(config);
          if (!result.success) {
            let blocked = false;
            result.error.issues.forEach((issue) => {
              const variant = issueVariant(issue);
              if (variant !== "warning") blocked = true;
              const path = ["config", ...issue.path].join(".");
              form.setStatus(path, { key: path, variant, message: issue.message });
            });
            if (blocked) return;
          }
          const [newConfig, newRack] = await onConfigure(client, config, name);
          form.set("config", newConfig, SKIP_AUTOSAVE);
          if (primitive.isNonZero(newRack)) form.set("rack", newRack, SKIP_AUTOSAVE);
          if (!(await saveAsync())) return;
        }
        await client.tasks.executeCommand({ task: taskKey, type: "start" });
      }, "Failed to start task");
    }, [client, form, saveAsync, taskKey, handleError, canEdit]);

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
        className={CSS.cls(CSS.B("task-configure"), CSS.BM("task-configure", type))}
        grow
        empty
      >
        <Flex.Box grow>
          <PForm.Form<PTask.FormSchema<S>>
            {...form}
            mode={isSnapshot || !canEdit ? "preview" : "normal"}
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
