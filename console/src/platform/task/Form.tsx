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
  status,
  type Synnax,
  task,
} from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Status } from "@synnaxlabs/lyra/status";
import { Access, Synnax as PSynnax, Task as PTask } from "@synnaxlabs/pluto";
import { deep, narrow, primitive } from "@synnaxlabs/x";
import { createContext, type FC, useContext, useEffect, useRef, useState } from "react";
import type { z } from "zod";

import { Analytics } from "@/platform/analytics";
import { CSS } from "@/platform/css";
import { Errors } from "@/platform/errors";
import { Section } from "@/platform/section";
import { Controls } from "@/platform/task/controls";
import { Rack } from "@/platform/task/Rack";
import { formPath } from "@/platform/task/types";
import { UtilityButtons } from "@/platform/task/UtilityButtons";

export interface OnConfigure<Config extends z.ZodType = z.ZodType> {
  (
    client: Synnax,
    config: z.infer<Config>,
    name: string,
  ): Promise<[z.infer<Config>, rack.Key]>;
}

/** Sends the edits that a form holds. Resolves false when an edit stays unsent. */
type Flush = () => Promise<boolean>;

const FlushesContext = createContext<Set<Flush> | null>(null);
FlushesContext.displayName = "Task.FlushesContext";

/**
 * Sends the edits of a form nested in a task form before each save and deploy of the
 * task, so the Core holds what the task names. An edit that stays unsent stops the
 * deploy.
 */
export const useFlushFirst = (flush: Flush): void => {
  const flushes = useContext(FlushesContext);
  if (flushes == null)
    throw new Error("useFlushFirst must be called inside a task form");
  useEffect(() => {
    flushes.add(flush);
    return () => {
      flushes.delete(flush);
    };
  }, [flushes, flush]);
};

export interface getInitialValuesParams {
  client: Synnax;
  deviceKey?: device.Key;
  /** The devices the task starts on, deviceKey included. */
  deviceKeys: device.Key[];
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

// Puts each class instance of original, such as a DataType, back in copy, where
// structuredClone made it a plain object.
const keepInstances = (original: unknown, copy: unknown): void => {
  const keep = (v: unknown, set: (v: unknown) => void, next: unknown): void => {
    if (narrow.isObject(v) && !narrow.isPlainObject(v)) set(v);
    else keepInstances(v, next);
  };
  if (Array.isArray(original) && Array.isArray(copy))
    original.forEach((v, i) => keep(v, (c) => (copy[i] = c), copy[i]));
  else if (narrow.isPlainObject(original) && narrow.isPlainObject(copy))
    Object.entries(original).forEach(([k, v]) => {
      if (k === "__proto__" || k === "constructor" || k === "prototype") return;
      keep(v, (c) => (copy[k] = c), copy[k]);
    });
};

const copyConfig = <T,>(config: T): T => {
  const copy = deep.copy(config);
  keepInstances(config, copy);
  return copy;
};

const STILL_RUNS = "The task still runs its last deployed config.";

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

const Header = () => (
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
);

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
    const [nested] = useState(() => new Set<Flush>());
    const { form, flush } = useForm({
      query: { key: taskKey },
      beforeSend: async () => await Promise.all([...nested].map((f) => f())),
    });

    // The form saves on every edit, so a subject who cannot write the task gets it
    // read-only rather than fields that revert once the save is refused.
    const canEdit = Access.useUpdateGranted(task.resourceID(taskKey));

    const deployChecked = useRef(false);
    const [configuring, setConfiguring] = useState(false);
    const [failure, setFailure] = useState<Controls.DeployFailure>();
    // A Stop while a start configures cancels the start.
    const startCancelled = useRef(false);

    const handleDeploy = async () => {
      startCancelled.current = false;
      setFailure(undefined);
      setConfiguring(true);
      try {
        await deploy();
      } catch (e) {
        const { name, status: current } = form.value();
        const running = current?.details.running === true;
        const heading = {
          name,
          message: running ? "Failed to deploy changes" : "Failed to start task",
        };
        handleError(e, heading);
        const failed = status.fromException(e, heading);
        if (running) failed.description = `${STILL_RUNS} ${failed.description}`;
        setFailure({ status: failed, taskStatusTime: current?.time });
      }
      setConfiguring(false);
    };

    const deploy = async () => {
      if (client == null) throw new DisconnectedError();
      if (canEdit) {
        const flushed = await Promise.all([flush(), ...[...nested].map((f) => f())]);
        if (!flushed.every(Boolean)) return;
        if (!deployChecked.current) {
          form.addCheck(formPath.config, deployConfigZ);
          deployChecked.current = true;
        }
        await form.validateAsync();
        const first = form.getStatuses().find(({ variant }) => variant === "error");
        if (first != null) {
          // The part of the form that shows the error takes focus. With none, the toast
          // shows it.
          const target = first.key
            ?.split(".")
            .reduce<deep.Path<unknown>>((p, k) => p[k], deep.path<unknown>());
          if (target == null || !form.focus(target, { optional: true }))
            throw new Error(first.message);
          return;
        }
        const { config, name, rack } = form.value();
        // The form's values are the cached task, and onConfigure changes its config.
        const [newConfig, newRack] = await onConfigure(
          client,
          copyConfig(config),
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
      if (startCancelled.current) return;
      // A pinned start runs only the config this form shows. The Driver refuses it
      // when another client changed the config since.
      await client.tasks.executeCommand({
        task: taskKey,
        type: "start",
        configHash: form.value().configHash,
        pinned: true,
      });
      capture("task_started", { type });
    };

    const handleStop = () => {
      startCancelled.current = true;
      handleError(async () => {
        if (client == null) throw new DisconnectedError();
        await client.tasks.executeCommand({ task: taskKey, type: "stop" });
      }, "Failed to stop task");
    };

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
            mode={canEdit ? "normal" : "preview"}
          >
            {showHeader && <Header />}
            {Properties != null && (
              <Flex.Box className={CSS.B("task-properties")} x wrap>
                <Properties />
              </Flex.Box>
            )}
            <Section.Section
              x
              className={CSS.B("task-channel-form-container")}
              grow
              empty
            >
              <Errors.SuspenseBoundary>
                <FlushesContext value={nested}>
                  <Form />
                </FlushesContext>
              </Errors.SuspenseBoundary>
            </Section.Section>
            {showControls && (
              <Controls.Controls
                configuring={configuring}
                failure={failure}
                onDeploy={() => void handleDeploy()}
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
