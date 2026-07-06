// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type framer, type Synnax as Client, task } from "@synnaxlabs/client";
import { Drift } from "@synnaxlabs/drift";
import { Form as PForm, type Status } from "@synnaxlabs/pluto";
import { id, TimeStamp } from "@synnaxlabs/x";
import {
  fireEvent,
  render,
  type RenderResult,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { act, type PropsWithChildren, type ReactElement } from "react";
import { type z } from "zod";

import { type Layout } from "@/platform/layout";
import { type FormLayoutArgs } from "@/platform/task/Form";
import { Session } from "@/session";
import {
  CaptureStatuses,
  createConsoleWrapper,
  createTestStore,
  renderHookWithConsole,
  renderWithConsole,
  type RenderWithConsoleOptions,
  type TestStore,
  uniqueName,
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

export interface RenderTaskFormLayoutOptions {
  /** Client backing the console wrapper; null (default) for cluster-free specs. */
  client?: Client | null;
  /** Layout args the wrapped form reads (deviceKey, taskKey, rackKey, config). */
  args?: FormLayoutArgs;
  /**
   * When provided, a CaptureStatuses probe is mounted alongside the renderer and this
   * callback receives the notification list on every change.
   */
  onStatuses?: (statuses: Status.NotificationSpec[]) => void;
}

export interface RenderTaskFormLayoutResult extends RenderResult {
  store: TestStore;
  layoutKey: string;
}

/**
 * Renders a Task.wrapForm renderer the way the layout mosaic does: places a layout of
 * the given type carrying `args` into a real console store, then mounts the renderer
 * against it inside the full console provider stack.
 */
export const renderTaskFormLayout = async (
  Renderer: Layout.Renderer,
  type: string,
  options: RenderTaskFormLayoutOptions = {},
): Promise<RenderTaskFormLayoutResult> => {
  const { client = null, args = {}, onStatuses } = options;
  const layoutKey = uniqueName("layout");
  const store = await createTestStore();
  act(() => {
    store.dispatch(
      Session.Layout.place({
        key: layoutKey,
        type,
        name: layoutKey,
        location: "mosaic",
        windowKey: Drift.MAIN_WINDOW,
        window: { title: layoutKey },
        args,
      }),
    );
  });
  const { wrapper } = await createConsoleWrapper({ client, store });
  const result = render(
    <>
      <Renderer layoutKey={layoutKey} visible focused={false} onClose={() => {}} />
      {onStatuses != null && <CaptureStatuses onStatuses={onStatuses} />}
    </>,
    { wrapper },
  );
  return { ...result, store, layoutKey };
};

/**
 * Waits for the task form's Configure button to leave its loading/disabled state, then
 * clicks it. Pluto buttons swallow clicks while disabled, so clicking without the wait
 * races the form's initial query.
 */
export const clickConfigure = async (): Promise<void> => {
  const button = await waitFor(() => {
    const b = screen.getByRole("button", { name: /Configure/ });
    if (b.classList.contains("pluto--disabled"))
      throw new Error("configure button is disabled");
    return b;
  });
  fireEvent.click(button);
};

/**
 * Polls the layout args until the save flow writes the created task's key back onto
 * the layout, then returns it.
 */
export const awaitTaskKey = async (
  store: TestStore,
  layoutKey: string,
): Promise<task.Key> =>
  await waitFor(() => {
    const args = Session.Layout.selectArgs<FormLayoutArgs>(store.getState(), layoutKey);
    if (args?.taskKey == null) throw new Error("task key not set on layout args");
    return args.taskKey;
  });

/**
 * Finds the dialog trigger of the mounted select whose current value renders as text.
 * Select triggers expose no accessible name, so this matches on the shown value.
 */
export const findDialogTriggerByText = async (text: string): Promise<HTMLElement> =>
  await waitFor(() => {
    const triggers = Array.from(
      document.querySelectorAll<HTMLElement>(".pluto-dialog__trigger"),
    );
    const match = triggers.find((t) => t.textContent?.includes(text));
    if (match == null) throw new Error(`dialog trigger showing "${text}" not found`);
    return match;
  });

/**
 * Opens the dialog select whose trigger currently shows triggerText and clicks the
 * option showing optionText. Options are queried inside the open dialog, since the
 * option's text may also appear on other closed triggers in the form.
 */
export const selectFromDropdown = async (
  triggerText: string,
  optionText: string,
): Promise<void> => {
  fireEvent.click(await findDialogTriggerByText(triggerText));
  const option = await waitFor(() => {
    const dialogs = screen.getAllByRole("dialog");
    return within(dialogs[dialogs.length - 1]).getByText(optionText);
  });
  fireEvent.click(option);
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
