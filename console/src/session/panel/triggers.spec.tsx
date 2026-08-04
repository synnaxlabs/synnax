// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Panel as Pluto } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { act, renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Modals } from "@/session/modals";
import { Panel } from "@/session/panel";
import { createConsoleWrapper, type TestStore } from "@/testutil";

const PANEL = uuid.create();
const TAB = uuid.create();
const OTHER_TAB = uuid.create();

const Noop: Modals.Content = () => null;

const setup = async (tabScope: string = TAB) => {
  const { wrapper: Console, store } = await createConsoleWrapper({ client: null });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <Pluto.Scope.Provider value={PANEL}>
        <Pluto.TabScope.Provider value={tabScope}>{children}</Pluto.TabScope.Provider>
      </Pluto.Scope.Provider>
    </Console>
  );
  Wrapper.displayName = "PanelTriggersWrapper";
  const { result } = renderHook(
    () => ({
      isActive: Panel.useGetTabTriggersActive(),
      modals: Modals.useStore("PanelTriggersSpec"),
    }),
    { wrapper: Wrapper },
  );
  return { result, store };
};

const focusTab = (store: TestStore, tabKey: string): void =>
  void act(() => {
    store.dispatch(
      Panel.internalSelectTab({ key: PANEL, tabKey, otherTabKeys: [tabKey] }),
    );
    store.dispatch(Panel.select({ key: PANEL }));
  });

describe("useGetTabTriggersActive", () => {
  it("should be active for the focused tab of the selected panel", async () => {
    const { result, store } = await setup();
    expect(result.current.isActive()).toBe(false);
    focusTab(store, TAB);
    expect(result.current.isActive()).toBe(true);
  });

  it("should be inactive for a background tab", async () => {
    const { result, store } = await setup();
    focusTab(store, OTHER_TAB);
    expect(result.current.isActive()).toBe(false);
  });

  it("should be inactive while a modal is open", async () => {
    const { result, store } = await setup();
    focusTab(store, TAB);
    act(() => result.current.modals.push(Noop, {}, () => {}));
    expect(result.current.isActive()).toBe(false);
    act(() => result.current.modals.closeTop());
    expect(result.current.isActive()).toBe(true);
  });
});
