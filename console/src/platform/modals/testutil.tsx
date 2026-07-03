// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import {
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

import { Modals } from "@/platform/modals";
import { Session } from "@/session";
import {
  type ConsolePreloadedState,
  createConsoleWrapper,
  type TestStore,
} from "@/testutil";
import { createSynnaxWrapper } from "@/testutil/Synnax";

const Base = createSynnaxWrapper({ client: null });

/**
 * The provider stack every modal spec renders within: the proven Pluto-rendering
 * Synnax wrapper (with a null client, since modals never touch the cluster) plus the
 * per-window modal store Provider.
 */
export const Wrapper: FC<PropsWithChildren> = ({ children }): ReactElement => (
  <Base>
    <Session.Modals.Provider>{children}</Session.Modals.Provider>
  </Base>
);
Wrapper.displayName = "Wrapper";

/**
 * Renders ui together with a live {@link Modals.Stack} inside {@link Wrapper}, so modals
 * pushed during the test actually mount. Modals portal to the document body; query them
 * via the returned result's baseElement or testing-library's screen.
 */
export const renderWithModals = (ui: ReactNode): RenderResult =>
  render(
    <>
      {ui}
      <Modals.Stack />
    </>,
    { wrapper: Wrapper },
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
  /** Client backing the console wrapper; null (default) for cluster-free specs. */
  client?: Client | null;
  preloadedState?: ConsolePreloadedState;
  store?: TestStore;
}

export interface ModalOpenerHandle<R> extends RenderResult {
  store: TestStore;
  /** The value returned by the most recent opener invocation. */
  result: () => R | undefined;
  /** Invokes the opener again (e.g. after the modal was closed). */
  reopen: () => void;
}

/**
 * Renders a modal-opener hook inside the full console provider stack with a mounted
 * {@link Modals.Stack}, invokes the opener with args, and returns the render result,
 * backing store, and the opener's return value (a promise for prompt-style hooks).
 */
export const renderModalOpener = async <Args extends unknown[], R>(
  useOpen: () => (...args: Args) => R,
  args: Args,
  options: RenderModalOpenerOptions = {},
): Promise<ModalOpenerHandle<R>> => {
  const { client = null, preloadedState, store } = options;
  const { wrapper, store: resolvedStore } = await createConsoleWrapper({
    client,
    preloadedState,
    store,
  });
  const box: { current?: R } = {};
  const Harness = (): ReactElement => {
    const open = useOpen();
    return <button onClick={() => (box.current = open(...args))}>open modal</button>;
  };
  Harness.displayName = "ModalOpenerHarness";
  const rendered = render(
    <>
      <Harness />
      <Modals.Stack />
    </>,
    { wrapper },
  );
  const reopen = () =>
    fireEvent.click(screen.getByRole("button", { name: "open modal" }));
  reopen();
  return { ...rendered, store: resolvedStore, result: () => box.current, reopen };
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
 * Finds the icon-only dismiss button a modal Header renders (the only button in the
 * document whose subtree contains no text).
 */
export const findDismissButton = (): HTMLButtonElement => {
  const btn = screen
    .getAllByRole("button")
    .find((b) => (b.textContent ?? "").trim() === "");
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
