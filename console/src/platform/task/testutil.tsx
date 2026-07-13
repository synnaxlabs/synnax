// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type framer, panel, type Synnax as Client, task } from "@synnaxlabs/client";
import { Drift } from "@synnaxlabs/drift";
import {
  Flux,
  Form as PForm,
  Panel as PlutoPanel,
  type Status,
} from "@synnaxlabs/pluto";
import { id, type record, TimeStamp, uuid } from "@synnaxlabs/x";
import {
  fireEvent,
  render,
  renderHook,
  type RenderResult,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { act, type FC, type PropsWithChildren, type ReactElement } from "react";
import { type z } from "zod";

import { type Panel } from "@/platform/panel";
import { type FormViewArgs } from "@/platform/task/Form";
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

export interface CreatedPanel {
  /** The flux store the panel doc was seeded into. */
  fluxStore: PlutoPanel.FluxSubStore;
  panelKey: panel.Key;
  /** Keys of the tabs seeded into the panel's single leaf, in order. */
  tabKeys: panel.TabKey[];
}

/**
 * Seeds a single-leaf panel doc holding the given tabs into the wrapper's flux store
 * and selects it in the session store, so Panel.useOpenTab and the tab-scoped panel
 * hooks resolve against it. When a client is given, the panel is also created on the
 * cluster so panel dispatches persist instead of rolling back.
 */
export const createSelectedPanel = async (
  wrapper: FC<PropsWithChildren>,
  store: TestStore,
  client: Client | null,
  tabs: panel.Tab[] = [],
  key: panel.Key = uuid.create(),
): Promise<CreatedPanel> => {
  const doc = panel.panelZ.parse({
    key,
    name: uniqueName("panel"),
    root: { variant: "leaf", tabs },
  });
  if (client != null) await client.panels.create(doc);
  const { result } = renderHook(() => Flux.useStore<PlutoPanel.FluxSubStore>(), {
    wrapper,
  });
  act(() => {
    result.current.panels.set(doc);
    store.dispatch(
      Session.Panel.select({ key: doc.key, windowKey: Drift.MAIN_WINDOW }),
    );
  });
  return {
    fluxStore: result.current,
    panelKey: doc.key,
    tabKeys: tabs.map((t) => t.key),
  };
};

/** Reads the current view args of the seeded tab from the panel doc. */
export const selectViewArgs = (
  { fluxStore, panelKey, tabKeys }: CreatedPanel,
  tabKey: panel.TabKey = tabKeys[0],
): record.Unknown | null => {
  const doc = fluxStore.panels.get(panelKey);
  if (doc == null) return null;
  const tab = panel.findTab(doc.root, tabKey);
  return tab?.variant === "view" ? tab.args : null;
};

export interface RenderInTaskFormOptions extends RenderWithConsoleOptions {
  values?: TaskFormValues;
  mode?: PForm.Mode;
}

export interface RenderInTaskFormResult extends RenderResult {
  store: TestStore;
  form: FormRef;
  tabKey: panel.TabKey;
}

interface PanelScopesProps extends PropsWithChildren {
  panelKey: panel.Key;
  tabKey: panel.TabKey;
}

/**
 * Provides the pluto panel and tab scopes the way the panel mosaic does, so
 * tab-scoped hooks (Session.Panel.useSelectIsFocused, panel view args) resolve.
 */
const PanelScopes = ({ panelKey, tabKey, children }: PanelScopesProps) => (
  <PlutoPanel.Scope.Provider value={panelKey}>
    <PlutoPanel.TabScope.Provider value={tabKey}>
      {children}
    </PlutoPanel.TabScope.Provider>
  </PlutoPanel.Scope.Provider>
);

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
  const tabKey = uuid.create();
  const result = await renderWithConsole(
    <PanelScopes panelKey={uuid.create()} tabKey={tabKey}>
      <TaskFormProvider values={merged} mode={mode} formRef={formRef}>
        {ui}
      </TaskFormProvider>
    </PanelScopes>,
    rest,
  );
  return { ...result, form: formRef, tabKey };
};

export interface RenderInTaskFormWithClientOptions extends RenderInTaskFormOptions {
  client?: Client | null;
}

export interface RenderInTaskFormWithClientResult extends RenderInTaskFormResult {
  /** The console wrapper backing the render, for seeding panel docs. */
  wrapper: FC<PropsWithChildren>;
  /** Key of the panel scope the form is rendered under, for seeding it as selected. */
  panelKey: panel.Key;
}

/**
 * Like renderInTaskForm, but backs the console provider stack with the given client so
 * components that reach for Synnax.use() (e.g. Controls issuing task commands) see it.
 */
export const renderInTaskFormWithClient = async (
  ui: ReactElement,
  options: RenderInTaskFormWithClientOptions = {},
): Promise<RenderInTaskFormWithClientResult> => {
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
  const panelKey = uuid.create();
  const tabKey = uuid.create();
  const result = render(
    <PanelScopes panelKey={panelKey} tabKey={tabKey}>
      <TaskFormProvider values={merged} mode={mode} formRef={formRef}>
        {ui}
      </TaskFormProvider>
    </PanelScopes>,
    { wrapper, ...rest },
  );
  return { ...result, store: resolvedStore, form: formRef, tabKey, panelKey, wrapper };
};

export interface RenderTaskFormViewOptions {
  /** Client backing the console wrapper; null (default) for cluster-free specs. */
  client?: Client | null;
  /** View args the wrapped form reads (deviceKey, taskKey, rackKey, config). */
  args?: FormViewArgs;
  /**
   * When provided, a CaptureStatuses probe is mounted alongside the renderer and this
   * callback receives the notification list on every change.
   */
  onStatuses?: (statuses: Status.NotificationSpec[]) => void;
}

export interface RenderTaskFormViewResult extends RenderResult, CreatedPanel {
  store: TestStore;
  tabKey: panel.TabKey;
}

/**
 * Renders a Task.wrapForm tab the way the panel mosaic does: seeds a panel doc whose
 * single leaf holds a view tab of the given type carrying `args`, then mounts the tab
 * content inside the panel and tab scopes within the full console provider stack.
 */
export const renderTaskFormTab = async (
  Tab: Panel.Tab,
  type: string,
  options: RenderTaskFormViewOptions = {},
): Promise<RenderTaskFormViewResult> => {
  const { client = null, args = {}, onStatuses } = options;
  const store = await createTestStore();
  const { wrapper } = await createConsoleWrapper({ client, store });
  const tab: panel.Tab = {
    variant: "view",
    key: uuid.create(),
    type,
    args,
  };
  const created = await createSelectedPanel(wrapper, store, client, [tab]);
  const result = render(
    <PanelScopes panelKey={created.panelKey} tabKey={tab.key}>
      <Tab.Content />
      {onStatuses != null && <CaptureStatuses onStatuses={onStatuses} />}
    </PanelScopes>,
    { wrapper },
  );
  return { ...result, ...created, store, tabKey: tab.key };
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
 * Polls the panel doc until the save flow writes the created task's key back onto
 * the view tab's args, then returns it.
 */
export const awaitTaskKey = async (created: CreatedPanel): Promise<task.Key> =>
  await waitFor(() => {
    const args = selectViewArgs(created);
    const taskKey = args?.taskKey;
    if (taskKey == null || typeof taskKey !== "string")
      throw new Error("task key not set on view args");
    return taskKey;
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
