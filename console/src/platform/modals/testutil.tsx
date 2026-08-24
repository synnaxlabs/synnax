// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";
import { type connection, type Synnax as Client } from "@synnaxlabs/client";
import { Triggers } from "@synnaxlabs/pluto";
import { type aether } from "@synnaxlabs/pluto/ether";
import { deep } from "@synnaxlabs/x";
import {
  act,
  fireEvent,
  render,
  renderHook,
  type RenderHookResult,
  type RenderResult,
  screen,
} from "@testing-library/react";
import {
  type FC,
  type PropsWithChildren,
  type ReactElement,
  type ReactNode,
} from "react";
import { Provider } from "react-redux";

import { Modals } from "@/platform/modals";
import { Session } from "@/session";
import {
  type ConsolePreloadedState,
  createConsoleWrapper,
  createSynnaxWrapper,
  type TestStore,
} from "@/testutil";

// Modal content mounts inside an error Boundary that reads the layout slice, so the
// stack needs a Redux Provider even though modals themselves live in a separate store.
const store = configureStore({
  reducer: Session.reducer,
  preloadedState: deep.copy(Session.ZERO_STATE),
});

const createWrapper = (connectionStatus?: connection.Status): FC<PropsWithChildren> => {
  const Base = createSynnaxWrapper({ client: null, connectionStatus });
  const InnerWrapper: FC<PropsWithChildren> = ({ children }): ReactElement => (
    <Base>
      <Provider store={store}>
        <Session.Modals.Context>{children}</Session.Modals.Context>
      </Provider>
    </Base>
  );
  InnerWrapper.displayName = "Wrapper";
  return InnerWrapper;
};

/**
 * The provider stack every modal spec renders within: the proven Pluto-rendering Synnax
 * wrapper (with a null client, since modals never touch the Core), a Redux Provider
 * backing the error Boundary, and the per-window modal store Provider.
 */
export const Wrapper = createWrapper();

export interface RenderWithModalsOptions {
  /** Connection status the Synnax context reports; defaults to disconnected. */
  connectionStatus?: connection.Status;
}

/**
 * Renders ui together with a live {@link Modals.Stack} inside {@link Wrapper}, so modals
 * pushed during the test actually mount. Modals portal to the document body; query them
 * via the returned result's baseElement or testing-library's screen.
 */
export const renderWithModals = (
  ui: ReactNode,
  { connectionStatus }: RenderWithModalsOptions = {},
): RenderResult =>
  render(
    <>
      {ui}
      <Modals.Stack />
    </>,
    { wrapper: connectionStatus == null ? Wrapper : createWrapper(connectionStatus) },
  );

/** The value a modal hook yields, paired with the live store backing it. */
export interface ModalHookHandle<T> {
  hook: T;
  store: Session.Modals.Store;
}

/**
 * Renders a modal hook (an opener or prompt hook) within {@link Wrapper} and returns its
 * value alongside the window's modal store, letting a spec invoke the opener and inspect
 * or settle the resulting stack directly.
 */
export const renderModalHook = <T,>(
  useHook: () => T,
): RenderHookResult<ModalHookHandle<T>, undefined> =>
  renderHook(() => ({ hook: useHook(), store: Session.Modals.useStore("test") }), {
    wrapper: Wrapper,
  });

/** Pulls the close callback the store bound into the topmost modal's rendered content. */
export const closeOf = (
  store: Session.Modals.Store,
): Session.Modals.ContentProps["close"] =>
  (store.getState().at(-1)?.render() as ReactElement<Session.Modals.ContentProps>).props
    .close;

export interface RenderModalOpenerOptions {
  /** Client backing the console wrapper; null (default) for Core-free specs. */
  client?: Client | null;
  preloadedState?: ConsolePreloadedState;
  store?: TestStore;
  /** Extra aether components merged over the default console test registry. */
  additionalRegistry?: aether.ComponentRegistry;
}

export interface ModalOpenerHandle<R> {
  store: TestStore;
  /** The DOM root modals portal into. */
  baseElement: HTMLElement;
  /** The value returned by the most recent opener invocation. */
  result: () => R | undefined;
  /** Invokes the opener again (e.g. after the modal was closed). */
  reopen: () => void;
  unmount: () => void;
}

/**
 * Mounts a modal-opener hook inside the full console provider stack with a live
 * {@link Modals.Stack}, invokes the opener with args, and returns the backing store
 * plus the opener's return value (a promise for prompt-style hooks).
 */
export const renderModalOpener = async <Args extends unknown[], R>(
  useOpen: () => (...args: Args) => R,
  args: Args,
  options: RenderModalOpenerOptions = {},
): Promise<ModalOpenerHandle<R>> => {
  const { client = null, preloadedState, store, additionalRegistry } = options;
  const { wrapper: Console, store: resolvedStore } = await createConsoleWrapper({
    client,
    preloadedState,
    store,
    additionalRegistry,
  });
  const wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <Triggers.Provider>
        {children}
        <Modals.Stack />
      </Triggers.Provider>
    </Console>
  );
  const { result, unmount } = renderHook(useOpen, { wrapper });
  const box: { current?: R } = {};
  const reopen = () => {
    act(() => {
      box.current = result.current(...args);
    });
  };
  // Modal content that suspends is discarded when it opens inside a synchronous act
  // scope, so the first open needs an awaited one.
  await act(async () => {
    box.current = result.current(...args);
  });
  return {
    store: resolvedStore,
    baseElement: document.body,
    result: () => box.current,
    reopen,
    unmount,
  };
};

export interface OpenModalOptions<P> {
  client?: Client | null;
  params?: P;
}

/**
 * Opens the given modal-opener hook inside the full console provider stack with a
 * mounted modal stack, and returns the render result plus the console store. Pass a
 * real client to exercise the enabled/save path, or omit it (null) to exercise the
 * no-Core branch.
 */
export const openModal = async <P,>(
  useOpen: () => Modals.Opener<P>,
  { client = null, params }: OpenModalOptions<P> = {},
): Promise<ModalOpenerHandle<void>> =>
  await renderModalOpener(useOpen as () => (params?: P) => void, [params], {
    client,
  });

/**
 * Presses the Ctrl+Enter save shortcut through the triggers provider. The provider
 * identifies keys by KeyboardEvent.code and treats a modifier as a held key rather than
 * an event flag, and a button's trigger fires on release.
 */
export const pressSaveTrigger = (): void => {
  fireEvent.keyDown(window, { key: "Control", code: "ControlLeft" });
  fireEvent.keyDown(window, { code: "Enter" });
  fireEvent.keyUp(window, { code: "Enter" });
  fireEvent.keyUp(window, { key: "Control", code: "ControlLeft" });
};

/**
 * Finds the rendered button whose subtree contains the given text. Pluto buttons nest
 * their label, so role-name queries often miss them.
 */
export const findButton = (text: string): HTMLButtonElement => {
  const btn = screen
    .getAllByText(text)
    .map((el) => el.closest<HTMLButtonElement>("button"))
    .find((b) => b != null);
  if (btn == null) throw new Error(`button with text ${text} not found`);
  return btn;
};

/**
 * Like {@link findButton}, but returns the last matching button in document order. Use
 * when a modal's button label also appears in UI mounted before the modal (e.g. a
 * statically rendered context menu item with the same text).
 */
export const findLastButton = (text: string): HTMLButtonElement => {
  const buttons = screen
    .getAllByText(text)
    .map((el) => el.closest<HTMLButtonElement>("button"))
    .filter((b) => b != null);
  const btn = buttons[buttons.length - 1];
  if (btn == null) throw new Error(`button with text ${text} not found`);
  return btn;
};

/** Finds the icon-only dismiss button a modal Header renders. */
export const findDismissButton = (): HTMLButtonElement => {
  const btn = screen
    .getAllByRole("button")
    .find((b) => b.getAttribute("aria-label") === "Close");
  if (btn == null) throw new Error("modal dismiss button not found");
  return btn as HTMLButtonElement;
};

/** Finds the checkbox input backing the pluto switch field with the given label. */
export const getSwitch = (label: string): HTMLInputElement => {
  const input = screen
    .getByText(label)
    .closest("*")
    ?.parentElement?.querySelector<HTMLInputElement>("input[type='checkbox']");
  if (input == null) throw new Error(`switch ${label} not found`);
  return input;
};
