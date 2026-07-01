// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  render,
  renderHook,
  type RenderHookResult,
  type RenderResult,
} from "@testing-library/react";
import {
  type FC,
  type PropsWithChildren,
  type ReactElement,
  type ReactNode,
} from "react";

import { Modals } from "@/platform/modals";
import { Session } from "@/session";
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
