// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form as PForm } from "@synnaxlabs/pluto";
import { type RenderResult } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { type z } from "zod";

import {
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
