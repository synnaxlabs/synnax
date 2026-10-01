// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Analytics } from "@/app/analytics";
import { Analytics as Sink } from "@/platform/analytics";
import { createTestSink } from "@/platform/analytics/testutil";
import { Session } from "@/session";
import { createConsoleWrapper } from "@/testutil";

const renderWatch = async (account: Partial<Session.Account.SliceState>) => {
  const sink = createTestSink();
  const { wrapper: Wrapper, store } = await createConsoleWrapper({
    client: null,
    preloadedState: {
      [Session.Account.SLICE_NAME]: { ...Session.Account.ZERO_SLICE_STATE, ...account },
    },
  });
  render(
    <Wrapper>
      <Sink.Provider sink={sink}>
        <Analytics.Watch />
      </Sink.Provider>
    </Wrapper>,
  );
  return { sink, store };
};

const ACCOUNT = { user: "user_a", email: "a@example.com" };

describe("Analytics.Watch", () => {
  describe("account", () => {
    it("should identify the account the machine is linked to", async () => {
      const { sink } = await renderWatch(ACCOUNT);
      expect(sink.identify).toHaveBeenCalledWith({
        id: "user_a",
        email: "a@example.com",
      });
    });

    it("should leave a machine linked before links carried the user anonymous", async () => {
      const { sink } = await renderWatch({ email: "a@example.com" });
      expect(sink.identify).not.toHaveBeenCalled();
    });

    it("should reset when the machine unlinks", async () => {
      const { sink, store } = await renderWatch(ACCOUNT);
      act(() => {
        store.dispatch(Session.Account.clear());
      });
      expect(sink.reset).toHaveBeenCalledTimes(1);
    });

    it("should reset before identifying a different account", async () => {
      // Without the reset, the second account's first events would carry the first
      // account's identity.
      const { sink, store } = await renderWatch(ACCOUNT);
      act(() => {
        store.dispatch(
          Session.Account.link({
            activation: "act",
            secret: "shh",
            email: "b@example.com",
            user: "user_b",
          }),
        );
      });
      expect(sink.reset).toHaveBeenCalledTimes(1);
      expect(sink.identify).toHaveBeenLastCalledWith({
        id: "user_b",
        email: "b@example.com",
      });
      expect(sink.reset.mock.invocationCallOrder[0]).toBeLessThan(
        sink.identify.mock.invocationCallOrder[1],
      );
    });
  });
});
