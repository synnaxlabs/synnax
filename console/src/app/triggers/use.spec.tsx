// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Triggers as PTriggers } from "@synnaxlabs/pluto";
import { act, fireEvent, renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Triggers } from "@/app/triggers";
import { Session } from "@/session";
import { createConsoleWrapper, type TestStore } from "@/testutil";

const renderTriggers = async (): Promise<{
  overlaid: () => boolean;
  store: TestStore;
}> => {
  const { wrapper: Console, store } = await createConsoleWrapper({ client: null });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <PTriggers.Provider>{children}</PTriggers.Provider>
    </Console>
  );
  Wrapper.displayName = "Wrapper";
  const { result } = renderHook(
    () => {
      Triggers.use();
      return Session.Panel.useSelectOverlaid();
    },
    { wrapper: Wrapper },
  );
  return { overlaid: () => result.current, store };
};

const pressEscape = () => {
  fireEvent.keyDown(window, { key: "Escape", code: "Escape" });
  fireEvent.keyUp(window, { key: "Escape", code: "Escape" });
};

describe("app/triggers", () => {
  describe("escape", () => {
    it("should stop overlaying the focused tab", async () => {
      const { overlaid, store } = await renderTriggers();
      act(() => void store.dispatch(Session.Panel.startOverlaying({})));
      expect(overlaid()).toBe(true);
      act(pressEscape);
      expect(overlaid()).toBe(false);
    });

    it("should stop overlaying when a first press was claimed elsewhere", async () => {
      const { overlaid, store } = await renderTriggers();
      // A code editor claims the first press to blur itself, so the press that reaches
      // the overlay is the second of a pair and arrives as a double trigger.
      act(pressEscape);
      act(() => void store.dispatch(Session.Panel.startOverlaying({})));
      act(pressEscape);
      expect(overlaid()).toBe(false);
    });
  });
});
