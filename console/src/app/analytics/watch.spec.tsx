// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Analytics } from "@/app/analytics";
import { Analytics as Sink } from "@/platform/analytics";
import { reportTaskStopped } from "@/platform/task/testutil";
import { Session } from "@/session";
import { createConsoleWrapper, uniqueName } from "@/testutil";

const createSink = () => ({
  capture: vi.fn<Sink.Sink["capture"]>(),
  screen: vi.fn<Sink.Sink["screen"]>(),
  identify: vi.fn<Sink.Sink["identify"]>(),
  reset: vi.fn<Sink.Sink["reset"]>(),
  describe: vi.fn<Sink.Sink["describe"]>(),
});

interface RenderWatchParams {
  client?: Synnax | null;
  account?: Partial<Session.Account.SliceState>;
}

const renderWatch = async ({ client = null, account = {} }: RenderWatchParams = {}) => {
  const sink = createSink();
  const { wrapper: Wrapper, store } = await createConsoleWrapper({
    client,
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
      const { sink } = await renderWatch({ account: ACCOUNT });
      expect(sink.identify).toHaveBeenCalledWith({
        id: "user_a",
        email: "a@example.com",
      });
    });

    it("should leave a machine linked before links carried the user anonymous", async () => {
      const { sink } = await renderWatch({ account: { email: "a@example.com" } });
      expect(sink.identify).not.toHaveBeenCalled();
    });

    it("should reset when the machine unlinks", async () => {
      const { sink, store } = await renderWatch({ account: ACCOUNT });
      act(() => {
        store.dispatch(Session.Account.clear());
      });
      expect(sink.reset).toHaveBeenCalledTimes(1);
    });

    it("should reset before identifying a different account", async () => {
      // Without the reset, the second account's first events would carry the first
      // account's identity.
      const { sink, store } = await renderWatch({ account: ACCOUNT });
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

  describe("workspace", () => {
    const client = createTestClient();

    // Testing Library's waitFor polls with setInterval, which these specs fake, so they
    // wait with vi.waitFor instead.
    beforeEach(() => {
      vi.useFakeTimers({ toFake: ["setInterval", "clearInterval"] });
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    const renderBaseline = async () => {
      const rendered = await renderWatch({ client });
      await vi.waitFor(() => expect(rendered.sink.describe).toHaveBeenCalledTimes(1));
      const [baseline] = rendered.sink.describe.mock.calls[0];
      return { ...rendered, baseline };
    };

    const poll = async (): Promise<void> => {
      await act(async () => {
        vi.advanceTimersToNextTimer();
      });
    };

    it("should report a channel another client creates, and count it", async () => {
      const { sink, baseline } = await renderBaseline();
      await createTestClient().channels.create({
        name: uniqueName("analytics"),
        dataType: "float32",
        virtual: true,
      });
      await poll();
      await vi.waitFor(() =>
        expect(sink.capture).toHaveBeenCalledWith("resource_created", {
          resource: "channel",
        }),
      );
      const [latest] = sink.describe.mock.calls[sink.describe.mock.calls.length - 1];
      expect(latest.channel_count).toBeGreaterThan(baseline.channel_count);
    });

    it("should report a task once a running instance reports its config", async () => {
      const { sink } = await renderBaseline();
      const draft = await client.tasks.create({
        name: uniqueName("analytics"),
        type: "opc_read",
        config: { device: "", channels: [] },
      });
      // A channel created in the same interval proves the count ran, so the draft's
      // absence is not a count that has yet to happen.
      await client.channels.create({
        name: uniqueName("analytics"),
        dataType: "float32",
        virtual: true,
      });
      await poll();
      await vi.waitFor(() =>
        expect(sink.capture).toHaveBeenCalledWith("resource_created", {
          resource: "channel",
        }),
      );
      expect(sink.capture).not.toHaveBeenCalledWith("resource_created", {
        resource: "task",
      });
      await reportTaskStopped(client, draft.payload);
      await poll();
      await vi.waitFor(() =>
        expect(sink.capture).toHaveBeenCalledWith("resource_created", {
          resource: "task",
        }),
      );
    });
  });
});
