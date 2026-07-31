// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type EnhancedStore } from "@reduxjs/toolkit";
import {
  type access,
  type ontology,
  panel,
  type Synnax as Client,
  type SynnaxParams,
} from "@synnaxlabs/client";
import { Drift } from "@synnaxlabs/drift";
import { Access, Status, Synnax } from "@synnaxlabs/pluto";
import { type aether, eraser } from "@synnaxlabs/pluto/ether";
import { deep, id } from "@synnaxlabs/x";
import {
  render,
  renderHook,
  type RenderHookOptions,
  type RenderHookResult,
  type RenderOptions,
  type RenderResult,
  screen,
  waitFor,
} from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement, useEffect } from "react";
import { Provider } from "react-redux";
import { assert } from "vitest";

import { Session } from "@/session";
import { createAsyncSynnaxWrapper, createSynnaxWrapper } from "@/testutil/Synnax";

/**
 * Generates a unique, cluster-safe resource name: letters, digits, and underscores
 * only, prefixed so failures are traceable back to tests.
 */
export const uniqueName = (prefix: string = "test"): string =>
  `${prefix}_${id.create().replace(/-/g, "_")}`;

/**
 * Throws when value is null or undefined, narrowing it for subsequent use.
 * Excludes void as well as null and undefined: NonNullable leaves `void & {}`
 * behind, which drops every property off a `T | void` return.
 */
export function assertDefined<T>(
  value: T,
  message = "expected value to be defined",
): asserts value is Exclude<T, null | undefined | void> {
  assert(value != null, message);
}

/**
 * Waits for and returns the trigger button of the (single) mounted pluto dialog-based
 * select, which renders no accessible role or stable text.
 */
export const findDialogTrigger = async (): Promise<HTMLElement> =>
  await waitFor(() => {
    const el = document.querySelector<HTMLElement>(".pluto-dialog__trigger");
    assertDefined(el, "dialog trigger not found");
    return el;
  });

/** Finds the close button of the rendered pluto tag whose text matches name. */
export const findTagCloseButton = (name: string): HTMLElement => {
  const btn = screen
    .getByText(name)
    .closest(".pluto-tag")
    ?.querySelector<HTMLElement>(".pluto-tag__close");
  assertDefined(btn, `close button for tag ${name} not found`);
  return btn;
};

/**
 * Polls the store until the active window has a focused tab (the definite session
 * effect of opening a tab via Panel.useOpenTab) and returns its key.
 */
export const waitForFocusedTab = async (store: TestStore): Promise<string> =>
  await waitFor(() => {
    const state = store.getState();
    const panelKey = Session.Panel.selectSelected(state);
    const focused =
      panelKey == null
        ? undefined
        : Session.Panel.selectSelectedTabs(state, panelKey)[0];
    assertDefined(focused, "no tab focused");
    return focused;
  });

/**
 * Polls until the active window's focused tab resolves to a tab in the selected
 * panel's document and returns it. Bridges the session focus state (a tab key)
 * and the panel document (fetched from the cluster) that together define what a
 * Panel.useOpenTab call produced.
 */
export const resolveFocusedTab = async (
  store: TestStore,
  client: Client,
  match?: (tab: panel.Tab) => boolean,
): Promise<panel.Tab> =>
  await waitFor(async () => {
    const state = store.getState();
    const panelKey = Session.Panel.selectSelected(state);
    const focused =
      panelKey == null
        ? undefined
        : Session.Panel.selectSelectedTabs(state, panelKey)[0];
    assertDefined(focused, "no tab focused");
    assertDefined(panelKey, "no panel selected");
    const doc = await client.panels.retrieve({ key: panelKey });
    const tab = panel.findTab(doc.root, focused);
    assertDefined(tab, "focused tab not found in panel");
    // Placement lands focus one dispatch after the click, so a test opening a
    // second tab must poll until focus reaches the expected one rather than
    // reading the still-focused prior tab.
    assert(match == null || match(tab), "focused tab does not match");
    return tab;
  });

/**
 * Creates a project on the cluster and marks it active in the session. Panel
 * placement (Panel.useOpenTab) auto-creates panels under the active project, so
 * any spec that exercises opening a tab must establish one first.
 */
export const selectTestProject = async (
  store: TestStore,
  client: Client,
): Promise<string> => {
  const proj = await client.projects.create({ name: id.create(), layout: {} });
  store.dispatch(Session.Project.select(proj.key));
  return proj.key;
};

export interface CaptureStatusesProps {
  onStatuses: (statuses: Status.NotificationSpec[]) => void;
}

/**
 * Reports the current status notifications to onStatuses on every change. Mount
 * alongside the UI under test to observe statuses raised during a component render.
 */
export const CaptureStatuses = ({ onStatuses }: CaptureStatusesProps): null => {
  const { statuses } = Status.useNotifications();
  useEffect(() => {
    onStatuses(statuses);
  }, [onStatuses, statuses]);
  return null;
};

export type ConsolePreloadedState = Partial<Session.State>;

export interface ConsoleTestProviderOptions {
  preloadedState?: ConsolePreloadedState;
}

/**
 * Builds the console test store the same way the app builds its store: the full root
 * reducer wired through drift's middleware, but with a NoopRuntime in place of the Tauri
 * runtime. Window actions (Drift.focusWindow, createWindow) resolve their labels exactly
 * as they do in production. Async because Drift.configureStore awaits runtime setup. This
 * is the one console store — every render helper below is backed by it.
 */
export const createTestStore = async (options: ConsoleTestProviderOptions = {}) => {
  const { preloadedState } = options;
  return await Drift.configureStore({
    runtime: new Drift.NoopRuntime(),
    reducer: Session.reducer,
    preloadedState: deep.copy({ ...Session.ZERO_STATE, ...preloadedState }),
    middleware: (getDefault) => getDefault().concat(...Session.BASE_MIDDLEWARE),
    enablePrerender: false,
  });
};

export type TestStore = Awaited<ReturnType<typeof createTestStore>>;

const ADDITIONAL_REGISTRY = eraser.REGISTRY;

const composeConsole = (
  SynnaxWrapper: FC<PropsWithChildren>,
  store: EnhancedStore,
): FC<PropsWithChildren> => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <SynnaxWrapper>
      <Provider store={store}>
        <Session.Modals.Context>{children}</Session.Modals.Context>
      </Provider>
    </SynnaxWrapper>
  );
  return Wrapper;
};

export interface RenderWithConsoleOptions extends RenderOptions {
  preloadedState?: ConsolePreloadedState;
  store?: TestStore;
}

export const renderWithConsole = async (
  ui: ReactElement,
  options: RenderWithConsoleOptions = {},
): Promise<RenderResult & { store: TestStore }> => {
  const { preloadedState, store, ...rest } = options;
  const resolvedStore = store ?? (await createTestStore({ preloadedState }));
  const Wrapper = composeConsole(
    createSynnaxWrapper({ client: null, additionalRegistry: ADDITIONAL_REGISTRY }),
    resolvedStore,
  );
  return { ...render(ui, { wrapper: Wrapper, ...rest }), store: resolvedStore };
};

export interface RenderLinkHookOptions {
  client?: Client | null;
  preloadedState?: ConsolePreloadedState;
}

export const renderLinkHook = async <H,>(
  useHook: () => H,
  options: RenderLinkHookOptions = {},
): Promise<{ handler: H; store: TestStore; modals: Session.Modals.Store }> => {
  const { client = null, preloadedState } = options;
  const store = await createTestStore({ preloadedState });
  const Wrapper = composeConsole(
    createSynnaxWrapper({ client, additionalRegistry: ADDITIONAL_REGISTRY }),
    store,
  );
  const { result } = renderHook(
    () => ({ handler: useHook(), modals: Session.Modals.useStore("renderLinkHook") }),
    { wrapper: Wrapper },
  );
  return { handler: result.current.handler, store, modals: result.current.modals };
};

export interface RenderHookWithConsoleOptions<Props> extends RenderHookOptions<Props> {
  preloadedState?: ConsolePreloadedState;
  store?: TestStore;
  client?: Client | null;
}

export const renderHookWithConsole = async <Result, Props>(
  cb: (props: Props) => Result,
  options: RenderHookWithConsoleOptions<Props> = {},
): Promise<RenderHookResult<Result, Props> & { store: TestStore }> => {
  const { preloadedState, store, client = null, ...rest } = options;
  const resolvedStore = store ?? (await createTestStore({ preloadedState }));
  const Wrapper = composeConsole(
    createSynnaxWrapper({ client, additionalRegistry: ADDITIONAL_REGISTRY }),
    resolvedStore,
  );
  return { ...renderHook(cb, { wrapper: Wrapper, ...rest }), store: resolvedStore };
};

export interface CreateConsoleWrapperParams {
  client: Client | null;
  preloadedState?: ConsolePreloadedState;
  store?: TestStore;
  /** Extra aether components merged over the default (eraser) test registry. */
  additionalRegistry?: aether.ComponentRegistry;
}

export const createConsoleWrapper = async ({
  client,
  preloadedState,
  store,
  additionalRegistry,
}: CreateConsoleWrapperParams): Promise<{
  wrapper: FC<PropsWithChildren>;
  store: TestStore;
}> => {
  const resolvedStore = store ?? (await createTestStore({ preloadedState }));
  const SynnaxWrapper = await createAsyncSynnaxWrapper({
    client,
    additionalRegistry: { ...ADDITIONAL_REGISTRY, ...additionalRegistry },
  });
  return {
    wrapper: composeConsole(SynnaxWrapper, resolvedStore),
    store: resolvedStore,
  };
};

/**
 * Warms the client's permission cache with a resolved grant for `action` on `id`, so
 * synchronous Access checks (createGranted, updateGranted) used by file ingesters pass
 * exactly as they do in a running app.
 */
export const awaitGranted = async (
  client: Client,
  id: ontology.ID,
  action: access.Action = "create",
): Promise<void> => {
  const { wrapper } = await createConsoleWrapper({ client });
  const { result } = renderHook(() => Access.useGranted({ objects: id, action }), {
    wrapper,
  });
  await waitFor(() => {
    if (!result.current) throw new Error(`${action} grant did not resolve`);
  });
};

export interface CreateConnectedConsoleWrapperParams extends CreateConsoleWrapperParams {
  /** Connection parameters handed to the production pluto Synnax.Provider. */
  connParams: SynnaxParams;
}

/**
 * Like createConsoleWrapper, but nests the production pluto Synnax.Provider inside the
 * stack, so Synnax.useConnectionStatus reflects a live connection to the cluster at
 * connParams, the same wiring the app uses in production.
 */
export const createConnectedConsoleWrapper = async ({
  connParams,
  ...args
}: CreateConnectedConsoleWrapperParams): Promise<{
  wrapper: FC<PropsWithChildren>;
  store: TestStore;
}> => {
  const { wrapper: Console, store } = await createConsoleWrapper(args);
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <Synnax.Provider connParams={connParams}>{children}</Synnax.Provider>
    </Console>
  );
  return { wrapper: Wrapper, store };
};

const SessionSynnaxProvider = ({ children }: PropsWithChildren): ReactElement => {
  const cluster = Session.Cluster.useSelectState();
  return <Synnax.Provider connParams={cluster}>{children}</Synnax.Provider>;
};
SessionSynnaxProvider.displayName = "SessionSynnaxProvider";

/**
 * Like createConnectedConsoleWrapper, but derives the provider's connection params from
 * the store's selected cluster, exactly as the production Pluto.Context does. Use for
 * tests that exercise the login -> select -> connect flow.
 */
export const createSessionConsoleWrapper = async (
  args: CreateConsoleWrapperParams,
): Promise<{ wrapper: FC<PropsWithChildren>; store: TestStore }> => {
  const { wrapper: Console, store } = await createConsoleWrapper(args);
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <SessionSynnaxProvider>{children}</SessionSynnaxProvider>
    </Console>
  );
  Wrapper.displayName = "SessionConsoleWrapper";
  return { wrapper: Wrapper, store };
};
