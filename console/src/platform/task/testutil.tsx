// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type framer, type Synnax as Client, task } from "@synnaxlabs/client";
import { Form as PForm } from "@synnaxlabs/pluto";
import { id, TimeStamp } from "@synnaxlabs/x";
import { fireEvent, render, type RenderResult } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { type z } from "zod";

import {
  createConsoleWrapper,
  renderHookWithConsole,
  renderWithConsole,
  type RenderWithConsoleOptions,
  type TestStore,
} from "@/testutil";

export type TaskFormValues = Record<string, unknown>;

export type FormRef = { current: PForm.ContextValue<z.ZodType> | null };

/**
 * Renders a real pluto Form context seeded with `values` and invokes `useValue` with
 * that context. Lets form-context task hooks (useKey, useStatus, useIsRunning,
 * useIsSnapshot) be exercised in isolation without mounting the full task Form UI.
 */
export const renderTaskFormHook = async <R,>(
  values: TaskFormValues,
  useValue: (ctx: PForm.ContextValue<z.ZodType>) => R,
) =>
  await renderHookWithConsole(() => {
    const ctx = PForm.use<z.ZodType>({ values });
    return { ctx, value: useValue(ctx) };
  });

export interface TaskFormProviderProps extends PropsWithChildren {
  values: TaskFormValues;
  mode?: PForm.Mode;
  formRef?: FormRef;
}

/**
 * Provides a real pluto Form context to its children so ambient form-context task hooks
 * (e.g. useTare, which reads the "key" and "status" fields without an explicit ctx) can
 * be rendered against seeded form values. When `formRef` is provided, the live form
 * context is exposed through it for reading/writing form state in specs.
 */
export const TaskFormProvider = ({
  values,
  mode,
  formRef,
  children,
}: TaskFormProviderProps): ReactElement => {
  const ctx = PForm.use<z.ZodType>({ values, mode });
  if (formRef != null) formRef.current = ctx;
  return <PForm.Form {...ctx}>{children}</PForm.Form>;
};
TaskFormProvider.displayName = "TaskFormProvider";

/**
 * The default value tree a task form is built around: a persisted-less task with an
 * empty channel list. Specs merge their own top-level fields and `config` over this.
 */
export const DEFAULT_TASK_FORM_VALUES: TaskFormValues = {
  key: undefined,
  name: "Test Task",
  snapshot: false,
  status: undefined,
  config: { channels: [] },
};

export interface RenderInTaskFormOptions extends RenderWithConsoleOptions {
  values?: TaskFormValues;
  mode?: PForm.Mode;
}

export interface RenderInTaskFormResult extends RenderResult {
  store: TestStore;
  form: FormRef;
}

/**
 * Renders `ui` inside a real Pluto Form context wired into the full console provider
 * stack. Composes `renderWithConsole` and `TaskFormProvider`. The returned `form` ref
 * exposes the live form context so specs can read (`form.current!.get(path).value`) and
 * write (`form.current!.set(path, v)`) form state directly. `values` is merged over
 * `DEFAULT_TASK_FORM_VALUES` (top-level fields and `config` shallow-merged).
 */
export const renderInTaskForm = async (
  ui: ReactElement,
  options: RenderInTaskFormOptions = {},
): Promise<RenderInTaskFormResult> => {
  const { values, mode, ...rest } = options;
  const formRef: FormRef = { current: null };
  const defaultConfig = DEFAULT_TASK_FORM_VALUES.config as TaskFormValues;
  const merged: TaskFormValues = {
    ...DEFAULT_TASK_FORM_VALUES,
    ...values,
    config: { ...defaultConfig, ...(values?.config ?? {}) },
  };
  const result = await renderWithConsole(
    <TaskFormProvider values={merged} mode={mode} formRef={formRef}>
      {ui}
    </TaskFormProvider>,
    rest,
  );
  return { ...result, form: formRef };
};

export interface RenderInTaskFormWithClientOptions extends RenderInTaskFormOptions {
  client?: Client | null;
}

/**
 * Like renderInTaskForm, but backs the console provider stack with the given client so
 * components that reach for Synnax.use() (e.g. Controls issuing task commands) see it.
 */
export const renderInTaskFormWithClient = async (
  ui: ReactElement,
  options: RenderInTaskFormWithClientOptions = {},
): Promise<RenderInTaskFormResult> => {
  const { client = null, values, mode, preloadedState, store, ...rest } = options;
  const formRef: FormRef = { current: null };
  const defaultConfig = DEFAULT_TASK_FORM_VALUES.config as TaskFormValues;
  const merged: TaskFormValues = {
    ...DEFAULT_TASK_FORM_VALUES,
    ...values,
    config: { ...defaultConfig, ...(values?.config ?? {}) },
  };
  const { wrapper, store: resolvedStore } = await createConsoleWrapper({
    client,
    preloadedState,
    store,
  });
  const result = render(
    <TaskFormProvider values={merged} mode={mode} formRef={formRef}>
      {ui}
    </TaskFormProvider>,
    { wrapper, ...rest },
  );
  return { ...result, store: resolvedStore, form: formRef };
};

/**
 * Reads command-channel frames until one carries a command for taskKey, so parallel
 * suites writing their own task commands cannot interfere.
 */
export const awaitCommand = async (
  streamer: framer.Streamer,
  taskKey: task.Key,
): Promise<task.Command> => {
  for (;;) {
    const frame = await streamer.read();
    for (const sample of frame.get(task.COMMAND_CHANNEL_NAME)) {
      const cmd = task.commandZ.parse(sample);
      if (cmd.task === taskKey) return cmd;
    }
  }
};

export interface CreateTaskStatusOverrides extends Partial<
  Omit<task.Status, "details">
> {
  details?: Partial<task.Status["details"]>;
}

/** Builds a fully-populated task.Status, merging `overrides` over sane defaults. */
export const createTaskStatus = (
  overrides: CreateTaskStatusOverrides = {},
): task.Status => {
  const { details, ...rest } = overrides;
  return {
    key: id.create(),
    name: "Task Status",
    variant: "success",
    message: "Running smoothly",
    description: "",
    time: TimeStamp.now(),
    ...rest,
    details: { task: "0", running: false, cmd: "", ...details },
  };
};

/** Finds the single non-checkbox input rendered by a task form field. */
export const findFieldInput = (): HTMLInputElement => {
  const input = document.body.querySelector<HTMLInputElement>(
    "input:not([type='checkbox'])",
  );
  if (input == null) throw new Error("form field input not found");
  return input;
};

/** Commits `value` into a text or numeric field input by changing and blurring it. */
export const commitFieldInput = (input: HTMLInputElement, value: string): void => {
  fireEvent.change(input, { target: { value } });
  fireEvent.blur(input);
};
