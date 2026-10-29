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
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Status } from "@synnaxlabs/lyra/status";
import { Access, Synnax as PSynnax, Task as PTask } from "@synnaxlabs/pluto";
import { deep, primitive } from "@synnaxlabs/x";
import { type FC } from "react";
import { type z } from "zod";

import { Analytics } from "@/platform/analytics";
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
   * Validates the config when the user deploys. Errors show on their fields and block
   * the start. Warnings show but do not block. Keep the shape schemas lax: a field
   * whose value fails its shape schema stays unsent.
   */
  deployConfigZ: z.ZodType;
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
      <PForm.Field<string> path="name" showLabel={false}>
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
      <Flex.Box align="end" gap="small">
        <UtilityButtons />
        <Rack />
      </Flex.Box>
    </Flex.Box>
    {!isSnapshot && <ParentRangeButton />}
  </>
);

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

    // The form saves on every edit, so a subject who cannot write the task gets it
    // read-only rather than fields that revert once the save is refused.
    const canEdit = Access.useUpdateGranted(task.ontologyID(taskKey));

    const handleDeploy = () => {
      handleError(async () => {
        if (client == null) throw new DisconnectedError();
        if (canEdit) {
          if (!(await flush())) return;
          const { config, name, rack } = form.value();
          form.clearStatuses();
          const result = deployConfigZ.safeParse(config);
          if (!result.success) {
            let blocked = false;
            result.error.issues.forEach((issue) => {
              const variant = issueVariant(issue);
              if (variant !== "warning") blocked = true;
              // Fields address array items by their key, not by their index.
              const path = deep.resolvePath(
                ["config", ...issue.path].join("."),
                form.value(),
              );
              form.setStatus(path, { key: path, variant, message: issue.message });
            });
            if (blocked) return;
          }
          // The form's values are the cached task, and onConfigure changes its config.
          const [newConfig, newRack] = await onConfigure(
            client,
            deep.copy(config),
            name,
          );
          // Another client can edit the task while onConfigure runs, so only the
          // values onConfigure changed are written.
          form.batch(() => {
            PForm.setChanged(form, "config", config, newConfig);
            if (primitive.isNonZero(newRack))
              PForm.setChanged(form, "rack", rack, newRack);
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
