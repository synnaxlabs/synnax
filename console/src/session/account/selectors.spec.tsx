// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";
import { act, renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { describe, expect, it } from "vitest";

import { Account } from "@/session/account";

const createStore = () =>
  configureStore({ reducer: { [Account.SLICE_NAME]: Account.reducer } });

describe("account selectors", () => {
  describe("useSelect", () => {
    it("should follow the link", () => {
      const store = createStore();
      const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
        <Provider store={store}>{children}</Provider>
      );
      Wrapper.displayName = "Wrapper";
      const { result } = renderHook(() => Account.useSelect(), { wrapper: Wrapper });
      expect(result.current.email).toBeUndefined();
      act(() => {
        store.dispatch(
          Account.link({ activation: "a", secret: "s", email: "e@example.com" }),
        );
      });
      expect(result.current.email).toBe("e@example.com");
    });
  });
});
