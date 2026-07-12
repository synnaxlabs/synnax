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
  type Synnax as Client,
  type SynnaxParams,
} from "@synnaxlabs/client";
import { Drift } from "@synnaxlabs/drift";
import { Access, Flux, type Pluto, Status, Synnax } from "@synnaxlabs/pluto";
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

import { Session } from "@/session";
import { createAsyncSynnaxWrapper, createSynnaxWrapper } from "@/testutil/Synnax";

/**
 * Generates a unique, cluster-safe resource name: letters, digits, and underscores
 * only, prefixed so failures are traceable back to tests.
 */
export const uniqueName = (prefix: string = "test"): string =>
  `${prefix}_${id.create().replace(/-/g, "_")}`;

/** Throws when value is null or undefined, narrowing it for subsequent use. */
export function assertDefined<T>(
  value: T,
  message = "expected value to be defined",
): asserts value is NonNullable<T> {
  if (value == null) throw new Error(message);
}

/**
 * Waits for and returns the trigger button of the (single) mounted pluto dialog-based
 * select, which renders no accessible role or stable text.
 */
export const findDialogTrigger = async (): Promise<HTMLElement> =>
  await waitFor(() => {
    const el = document.querySelector<HTMLElement>(".pluto-dialog__trigger");
    if (el == null) throw new Error("dialog trigger not found");
    return el;
  });

/** Finds the close button of the rendered pluto tag whose text matches name. */
export const findTagCloseButton = (name: string): HTMLElement => {
  const btn = screen
    .getByText(name)
    .closest(".pluto-tag")
    ?.querySelector<HTMLElement>(".pluto-tag__close");
  if (btn == null) throw new Error(`close button for tag ${name} not found`);
  return btn;
};

/**
 * Polls the store until a layout of the given type has been placed and returns its
 * key.
 */
export const waitForPlacedLayout = async (
  store: TestStore,
  type: string,
): Promise<string> =>
  await waitFor(() => {
    const placed = Session.Layout.selectByFilter(
      store.getState(),
      (l) => l.type === type,
    );
    if (placed == null) throw new Error(`no ${type} layout placed`);
    return placed.key;
  });

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
    // The layout middleware is omitted: it drives real Tauri window creation, which
    // cannot run in jsdom. Everything else matches the production store.
    middleware: (getDefault) =>
      getDefault().concat(...Session.Nav.MIDDLEWARE, ...Session.Panel.MIDDLEWARE),
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
        <Session.Modals.Provider>{children}</Session.Modals.Provider>
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

export const renderLinkHook = async <H,>(
  useHook: () => H,
): Promise<{ handler: H; store: TestStore; modals: Session.Modals.Store }> => {
  const store = await createTestStore();
  const Wrapper = ({ children }: PropsWithChildren) => (
    <Provider store={store}>
      <Session.Modals.Provider>{children}</Session.Modals.Provider>
    </Provider>
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

export interface CreateConsoleWrapperArgs {
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
}: CreateConsoleWrapperArgs): Promise<{
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
 * Builds a flux store whose permission cache has resolved a grant for `action` on `id`
 * against the given client, so synchronous Access checks (createGranted,
 * updateGranted) used by file ingesters pass exactly as they do in a running app.
 */
export const createGrantedFluxStore = async (
  client: Client,
  id: ontology.ID,
  action: access.Action = "create",
): Promise<Pluto.FluxStore> => {
  const { wrapper } = await createConsoleWrapper({ client });
  const { result } = renderHook(
    () => ({
      store: Flux.useStore<Pluto.FluxStore>(),
      granted: Access.useGranted({ objects: id, action }),
    }),
    { wrapper },
  );
  await waitFor(() => {
    if (!result.current.granted) throw new Error(`${action} grant did not resolve`);
  });
  return result.current.store;
};

export interface CreateConnectedConsoleWrapperArgs extends CreateConsoleWrapperArgs {
  /** Connection parameters handed to the production pluto Synnax.Provider. */
  connParams: SynnaxParams;
}

/**
 * Like createConsoleWrapper, but nests the production pluto Synnax.Provider inside the
 * stack, so Synnax.useConnectionState reflects a live connection to the cluster at
 * connParams, the same wiring the app uses in production.
 */
export const createConnectedConsoleWrapper = async ({
  connParams,
  ...args
}: CreateConnectedConsoleWrapperArgs): Promise<{
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
