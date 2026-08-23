// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  channel,
  device,
  type framer,
  type ontology,
  panel,
  project,
  query,
  rack,
  type Synnax as Client,
  task,
} from "@synnaxlabs/client";
import { createPanelParent, createTestClient } from "@synnaxlabs/client/testutil";
import { Drift } from "@synnaxlabs/drift";
import { Form as PForm, Panel as PlutoPanel, type Status } from "@synnaxlabs/pluto";
import { id, TimeSpan, TimeStamp, uuid } from "@synnaxlabs/x";
import {
  fireEvent,
  type RenderResult,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { act, type FC, type PropsWithChildren, type ReactElement } from "react";
import { onTestFinished } from "vitest";
import { type z } from "zod";

import { CSS } from "@/platform/css";
import { type FormTabProps } from "@/platform/task/Form";
import { Session } from "@/session";
import {
  assertDefined,
  CaptureStatuses,
  createConsoleWrapper,
  createTestClientWithGrants,
  createTestStore,
  getIconButton,
  renderHookWithConsole,
  renderSuspended,
  renderWithConsole,
  type RenderWithConsoleOptions,
  type TestStore,
  uniqueName,
} from "@/testutil";

const defaultClient = createTestClient();

/**
 * Creates a client that may read and update tasks but only read channels, so specs can
 * check the channel-rename gates on task context menus.
 */
export const createChannelReadOnlyClient = async (client: Client): Promise<Client> =>
  await createTestClientWithGrants(client, {
    retrieve: [
      task.TYPE_ONTOLOGY_ID,
      device.TYPE_ONTOLOGY_ID,
      rack.TYPE_ONTOLOGY_ID,
      channel.TYPE_ONTOLOGY_ID,
      panel.TYPE_ONTOLOGY_ID,
      project.TYPE_ONTOLOGY_ID,
    ],
    update: [task.TYPE_ONTOLOGY_ID],
  });

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
    details: {
      task: "0",
      running: false,
      cmd: "",
      configHash: "",
      rack: 0,
      ...details,
    },
  };
};

/**
 * The default value tree a task form is built around: a persisted-less task with an
 * empty channel list. Specs merge their own top-level fields and `config` over this.
 * The status mirrors what core writes for a task that has never been deployed.
 */
export const DEFAULT_TASK_FORM_VALUES: TaskFormValues = {
  key: undefined,
  name: "Test Task",
  rack: 0,
  snapshot: false,
  configHash: "",
  status: createTaskStatus({
    variant: "disabled",
    message: "Test Task has not been deployed",
  }),
  config: { channels: [] },
};

export interface CreatedPanel {
  /** The client whose cache holds the created panel doc. */
  client: Client;
  panelKey: panel.Key;
  /** Keys of the tabs seeded into the panel's single leaf, in order. */
  tabKeys: panel.TabKey[];
}

/**
 * Creates a single-leaf panel doc holding the given tabs on the cluster and selects it
 * in the session store, so Panel.useOpenTab and the tab-scoped panel hooks resolve
 * against it through the client's cache.
 */
export const createSelectedPanel = async (
  store: TestStore,
  client: Client,
  tabs: panel.Tab[] = [],
  key: panel.Key = uuid.create(),
): Promise<CreatedPanel> => {
  const doc = panel.panelZ.parse({
    key,
    name: uniqueName("panel"),
    root: { variant: "leaf", tabs },
  });
  await client.panels.create({ ...doc, parent: await createPanelParent(client) });
  // Prime the query cache the way the mosaic's retrieve does, and keep the
  // answer live for the test: unsubscribed answers go stale by design.
  onTestFinished(client.panels.onChange(doc.key, () => {}));
  await client.panels.retrieve(doc.key);
  act(() => {
    store.dispatch(
      Session.Panel.select({ key: doc.key, windowKey: Drift.MAIN_WINDOW }),
    );
  });
  return { client, panelKey: doc.key, tabKeys: tabs.map((t) => t.key) };
};

/** Reads the resource ID of a tab from the cached panel doc, or null for none. */
export const selectTabResource = (
  { client, panelKey, tabKeys }: CreatedPanel,
  tabKey: panel.TabKey = tabKeys[0],
): ontology.ID | null => {
  const cached = client.panels.getCached(panelKey);
  if (!query.isLive(cached)) return null;
  const tab = panel.findTab(cached.root, tabKey);
  return tab?.variant === "resource" ? tab.resource : null;
};

/**
 * Polls the cached panel doc until a resource tab for a task appears anywhere in it,
 * then returns the task key it points at. Create-then-open flows insert a new tab
 * rather than mutating the seeded one, so every leaf tab is scanned.
 */
export const awaitTaskResourceTab = async ({
  client,
  panelKey,
}: CreatedPanel): Promise<task.Key> =>
  await waitFor(() => {
    const cached = client.panels.getCached(panelKey);
    if (!query.isLive(cached)) throw new Error("panel doc not found");
    const doc = cached;
    const tabs: panel.Tab[] = [];
    const visit = (node: panel.Node): void => {
      if (node.variant === "leaf") tabs.push(...node.tabs);
      else {
        visit(node.first);
        visit(node.last);
      }
    };
    visit(doc.root);
    const match = tabs.find(
      (t) => t.variant === "resource" && t.resource.type === task.TYPE_ONTOLOGY_ID.type,
    );
    if (match == null || match.variant !== "resource")
      throw new Error("task resource tab not found");
    return match.resource.key;
  });

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
  const result = await renderSuspended(
    <PanelScopes panelKey={panelKey} tabKey={tabKey}>
      <TaskFormProvider values={merged} mode={mode} formRef={formRef}>
        {ui}
      </TaskFormProvider>
    </PanelScopes>,
    { wrapper, ...rest },
  );
  return { ...result, store: resolvedStore, form: formRef, tabKey, panelKey, wrapper };
};

export interface RenderTaskFormTabOptions {
  /** Client backing the console wrapper; falls back to a shared test client. */
  client?: Client | null;
  /** Client the console renders as; the panel and task are created with `client`. */
  as?: Client;
  /** Key of the task row the form edits. */
  taskKey?: task.Key;
  /** Row to create and open when `taskKey` is omitted. */
  task?: task.New;
  /**
   * When provided, a CaptureStatuses probe is mounted alongside the renderer and this
   * callback receives the notification list on every change.
   */
  onStatuses?: (statuses: Status.NotificationSpec[]) => void;
}

export interface RenderTaskFormTabResult extends RenderResult, CreatedPanel {
  store: TestStore;
  tabKey: panel.TabKey;
}

/**
 * Renders a Task.wrapForm tab the way the task panel entry does: seeds a panel doc
 * whose single leaf holds a resource tab for the task, then mounts the form inside
 * the panel and tab scopes within the full console provider stack.
 */
export const renderTaskFormTab = async (
  Form: FC<FormTabProps>,
  options: RenderTaskFormTabOptions = {},
): Promise<RenderTaskFormTabResult> => {
  const { onStatuses } = options;
  const client = options.client ?? defaultClient;
  const as = options.as ?? client;
  const taskKey =
    options.taskKey ??
    (options.task == null ? "" : (await client.tasks.create(options.task)).key);
  const store = await createTestStore();
  const { wrapper } = await createConsoleWrapper({ client: as, store });
  const tab: panel.Tab = {
    variant: "resource",
    key: uuid.create(),
    resource: task.ontologyID(taskKey),
  };
  const created = await createSelectedPanel(store, client, [tab]);
  if (as !== client) {
    // The panel scopes resolve against the rendering client's cache, so prime it
    // the same way createSelectedPanel primes the creating client's.
    onTestFinished(as.panels.onChange(created.panelKey, () => {}));
    await as.panels.retrieve(created.panelKey);
  }
  const result = await renderSuspended(
    <PanelScopes panelKey={created.panelKey} tabKey={tab.key}>
      <Form taskKey={taskKey} />
      {onStatuses != null && <CaptureStatuses onStatuses={onStatuses} />}
    </PanelScopes>,
    { wrapper },
  );
  return { ...result, ...created, store, tabKey: tab.key };
};

/**
 * Waits for the task form's start button to leave its loading/disabled state, then
 * clicks it to run the deploy pipeline. Pluto buttons swallow clicks while disabled,
 * so clicking without the wait races the form's initial query.
 */
export const clickDeploy = async (container: ParentNode): Promise<void> => {
  const button = await waitFor(() => {
    const b = getIconButton(container, "play");
    if (b.classList.contains("pluto--disabled"))
      throw new Error("start button is disabled");
    return b;
  });
  fireEvent.click(button);
};

/**
 * Finds the channel list row showing the given port. The details form for the selected
 * channel shows the same port, so a plain text query is ambiguous.
 */
export const findChannelListItem = async (port: string): Promise<HTMLElement> =>
  await waitFor(() => {
    const match = screen
      .getAllByText(port)
      .find((el) => el.closest(`.${CSS.B("channel-item")}`) != null);
    assertDefined(match, `channel list item for port "${port}" not found`);
    return match;
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
    assertDefined(match, `dialog trigger showing "${text}" not found`);
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
 * Finds the input rendered inside the Input.Item labeled by label. Item labels carry no
 * htmlFor, so this walks the item container instead of using getByLabelText.
 * @throws if no item or input renders for the label.
 */
export const getLabeledInput = (label: string): HTMLInputElement => {
  const item = screen.getByText(label).closest(".pluto-input__item");
  const input = item?.querySelector("input");
  if (input == null) throw new Error(`no input found for label "${label}"`);
  return input;
};

/**
 * Reads command-channel frames until one carries a command for taskKey, so parallel
 * suites writing their own task commands cannot interfere. Rejects after 10 seconds so
 * a command that never arrives names itself instead of hitting the test timeout.
 */
export const awaitCommand = async (
  streamer: framer.Streamer,
  taskKey: task.Key,
): Promise<task.Command> => {
  const read = async (): Promise<task.Command> => {
    for (;;) {
      const frame = await streamer.read();
      for (const sample of frame.get(task.COMMAND_CHANNEL_NAME)) {
        const cmd = task.commandZ.parse(sample);
        if (cmd.task === taskKey) return cmd;
      }
    }
  };
  let timer: ReturnType<typeof setTimeout> | undefined;
  const deadline = new Promise<never>((_, reject) => {
    timer = setTimeout(
      () => reject(new Error(`no command for task ${taskKey} received within 10s`)),
      TimeSpan.seconds(10).milliseconds,
    );
  });
  try {
    return await Promise.race([read(), deadline]);
  } finally {
    clearTimeout(timer);
  }
};

/**
 * Clicks the form's deploy button, waits for the resulting start command, then returns
 * the persisted task. The streamer opens before the click because deploy saves the task
 * and issues the command in one pass.
 */
export const deployAndAwaitTask = async <S extends task.Schemas = task.Schemas>(
  client: Client,
  container: ParentNode,
  key: task.Key,
  schemas?: S,
): Promise<task.Task<S>> => {
  const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
  try {
    await clickDeploy(container);
    await awaitCommand(streamer, key);
  } finally {
    streamer.close();
  }
  return await client.tasks.retrieve({ key, schemas });
};

/**
 * Writes the status a driver reports once a task has stopped. Specs run without a
 * driver, so a start command otherwise leaves the task mid-start forever and the
 * form offers no way to deploy again.
 */
export const reportTaskStopped = async (
  client: Client,
  tsk: task.Payload,
): Promise<void> => {
  await client.tasks.create({
    ...tsk,
    status: createTaskStatus({
      message: "Task stopped",
      details: { task: tsk.key, running: false, configHash: tsk.configHash },
    }),
  });
};

/** Finds the single non-checkbox input rendered by a task form field. */
export const findFieldInput = (): HTMLInputElement => {
  const input = document.body.querySelector<HTMLInputElement>(
    "input:not([type='checkbox'])",
  );
  assertDefined(input, "form field input not found");
  return input;
};

/** Commits `value` into a text or numeric field input by changing and blurring it. */
export const commitFieldInput = (input: HTMLInputElement, value: string): void => {
  fireEvent.change(input, { target: { value } });
  fireEvent.blur(input);
};

/** Whether the redeploy button is collapsed rather than revealed. */
export const isRedeployHidden = (): boolean =>
  screen.getByText("Redeploy").closest("[aria-hidden='true']") != null;

/**
 * Waits for the redeploy button to be revealed and enabled, then clicks it. The button
 * stays mounted while hidden and Pluto buttons swallow clicks while disabled, so
 * clicking without the wait races the drift computation and the form's initial query.
 */
export const clickRedeploy = async (): Promise<void> => {
  const button = await waitFor(() => {
    if (isRedeployHidden()) throw new Error("redeploy button is hidden");
    const b = screen.getByText("Redeploy").closest("button");
    assertDefined(b, "redeploy button not found");
    if (b.classList.contains("pluto--disabled"))
      throw new Error("redeploy button is disabled");
    return b;
  });
  fireEvent.click(button);
};
