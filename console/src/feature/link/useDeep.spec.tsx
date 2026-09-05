// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Status } from "@synnaxlabs/pluto";
import { TimeSpan } from "@synnaxlabs/x";
import { type UnlistenFn } from "@tauri-apps/api/event";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it, type Mock, vi } from "vitest";

import { Link } from "@/feature/link";
import { Link as PlatformLink } from "@/platform/link";
import { Session } from "@/session";
import {
  createSessionConsoleWrapper,
  renderHookWithConsole,
  type TestStore,
} from "@/testutil";

const client = (): Client => createTestClient();

interface Harness {
  connect: ReturnType<typeof vi.fn>;
  handlers: Record<string, ReturnType<typeof vi.fn>>;
  deps: Link.Deps;
  openURL: (urls: string[]) => void;
}

const setup = async (overrides: Partial<Link.Deps> = {}): Promise<Harness> => {
  const resolved = client();
  const connect = vi.fn(async () => resolved);
  const handlers = {
    schematic: vi.fn(async () => {}),
    range: vi.fn(async () => {}),
  };
  let openURL: (urls: string[]) => void = () => {};
  const onOpenURL = vi.fn(
    async (handler: (urls: string[]) => void): Promise<UnlistenFn> => {
      openURL = handler;
      return () => {};
    },
  );
  const deps: Link.Deps = {
    engine: "tauri",
    getCurrentURLs: async () => null,
    onOpenURL,
    ...overrides,
  };
  await renderHookWithConsole(() => Link.useDeep(connect, handlers, deps));
  return { connect, handlers, deps, openURL: (urls) => openURL(urls) };
};

const CORE_KEY = "local";
const OTHER_CORE_KEY = "other";
const THIRD_CORE_KEY = "third";

// All records reach the same live Core, so a Core switch in a spec connects for real
// while still changing the session's selected Core key.
const createCoreState = (): Session.Core.SliceState => ({
  ...Session.Core.ZERO_SLICE_STATE,
  cores: {
    [CORE_KEY]: {
      key: CORE_KEY,
      name: "Local",
      host: "localhost",
      port: 9090,
      username: "synnax",
      password: "seldon",
      secure: false,
    },
    [OTHER_CORE_KEY]: {
      key: OTHER_CORE_KEY,
      name: "Other",
      host: "127.0.0.1",
      port: 9090,
      username: "synnax",
      password: "seldon",
      secure: false,
    },
    [THIRD_CORE_KEY]: {
      key: THIRD_CORE_KEY,
      name: "Third",
      host: "127.0.0.1",
      port: 9090,
      username: "synnax",
      password: "seldon",
      secure: false,
    },
  },
  selected: CORE_KEY,
});

interface SettledHarness extends Omit<Harness, "connect" | "handlers"> {
  connect: Mock<PlatformLink.Connect>;
  handlers: Record<string, Mock<PlatformLink.Handler>>;
  store: TestStore;
  settled: () => boolean;
  statuses: () => Status.NotificationSpec[];
}

/**
 * Renders useDeep under a live Core with the real synchronizers, so settled reflects
 * production wiring. preMount runs against the store before the hook mounts, letting a
 * spec hold the workspace unsettled while the launch link is handled.
 */
const setupSettled = async (
  overrides: Partial<Link.Deps> = {},
  preMount?: (store: TestStore) => void,
): Promise<SettledHarness> => {
  const resolved = client();
  const connect: Mock<PlatformLink.Connect> = vi.fn(async () => resolved);
  const handlers: Record<string, Mock<PlatformLink.Handler>> = {
    schematic: vi.fn(async () => {}),
    range: vi.fn(async () => {}),
  };
  let openURL: (urls: string[]) => void = () => {};
  const onOpenURL = vi.fn(
    async (handler: (urls: string[]) => void): Promise<UnlistenFn> => {
      openURL = handler;
      return () => {};
    },
  );
  const deps: Link.Deps = {
    engine: "tauri",
    getCurrentURLs: async () => null,
    onOpenURL,
    ...overrides,
  };
  const { wrapper: Console, store } = await createSessionConsoleWrapper({
    client: null,
    preloadedState: { [Session.Core.SLICE_NAME]: createCoreState() },
  });
  preMount?.(store);
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <Session.SettledProvider>{children}</Session.SettledProvider>
    </Console>
  );
  Wrapper.displayName = "SettledDeepWrapper";
  const { result } = renderHook(
    () => {
      Link.useDeep(connect, handlers, deps);
      return {
        settled: Session.useSettled(),
        statuses: Status.useNotifications().statuses,
      };
    },
    { wrapper: Wrapper },
  );
  return {
    connect,
    handlers,
    deps,
    openURL: (urls) => openURL(urls),
    store,
    settled: () => result.current.settled,
    statuses: () => result.current.statuses,
  };
};

describe("useDeep", () => {
  beforeEach(() => localStorage.clear());
  afterEach(() => vi.useRealTimers());

  it("should open a resource link the app was launched from", async () => {
    const { connect, handlers } = await setup({
      getCurrentURLs: async () => ["synnax://cluster/c1/schematic/s1"],
    });
    await waitFor(() => expect(connect).toHaveBeenCalledWith("c1"));
    await waitFor(() =>
      expect(handlers.schematic).toHaveBeenCalledWith(
        expect.objectContaining({ key: "s1" }),
      ),
    );
  });

  it("should connect without placing a layout for a Core-only link", async () => {
    const { connect, handlers } = await setup({
      getCurrentURLs: async () => ["synnax://cluster/c1"],
    });
    await waitFor(() => expect(connect).toHaveBeenCalledWith("c1"));
    expect(handlers.schematic).not.toHaveBeenCalled();
    expect(handlers.range).not.toHaveBeenCalled();
  });

  it("should route a link opened while the app is already running", async () => {
    const { connect, handlers, deps, openURL } = await setup();
    await waitFor(() => expect(deps.onOpenURL).toHaveBeenCalled());
    expect(connect).not.toHaveBeenCalled();
    openURL(["synnax://cluster/c2/range/r9"]);
    await waitFor(() => expect(connect).toHaveBeenCalledWith("c2"));
    await waitFor(() =>
      expect(handlers.range).toHaveBeenCalledWith(
        expect.objectContaining({ key: "r9" }),
      ),
    );
  });

  it("should not route a malformed link", async () => {
    const { connect } = await setup({
      getCurrentURLs: async () => ["not-a-synnax-link"],
    });
    await act(async () => {});
    expect(connect).not.toHaveBeenCalled();
  });

  it("should connect but place nothing for an unknown resource type", async () => {
    const { connect, handlers } = await setup({
      getCurrentURLs: async () => ["synnax://cluster/c1/widget/w1"],
    });
    await waitFor(() => expect(connect).toHaveBeenCalledWith("c1"));
    expect(handlers.schematic).not.toHaveBeenCalled();
    expect(handlers.range).not.toHaveBeenCalled();
  });

  it("should ignore the next launch link once when the ignore flag is set", async () => {
    PlatformLink.markNextIgnored();
    const first = await setup({
      getCurrentURLs: async () => ["synnax://cluster/c1/schematic/s1"],
    });
    await act(async () => {});
    expect(first.connect).not.toHaveBeenCalled();
    // The flag is single-use: the hook clears it, so a later launch is handled normally.
    const second = await setup({
      getCurrentURLs: async () => ["synnax://cluster/c1/schematic/s1"],
    });
    await waitFor(() => expect(second.connect).toHaveBeenCalledWith("c1"));
  });

  it("should be a no-op when the engine is not tauri", async () => {
    const getCurrentURLs = vi.fn(async () => ["synnax://cluster/c1/schematic/s1"]);
    const { connect } = await setup({ engine: "web", getCurrentURLs });
    expect(getCurrentURLs).not.toHaveBeenCalled();
    expect(connect).not.toHaveBeenCalled();
  });

  it("should hold a runtime link until the workspace settles", async () => {
    const { store, settled, openURL, connect, handlers } = await setupSettled();
    await waitFor(() => expect(settled()).toBe(true));
    act(() => {
      store.dispatch(Session.Persist.beginSwap());
    });
    openURL(["synnax://cluster/c1/range/r1"]);
    await waitFor(() => expect(connect).toHaveBeenCalledWith("c1"));
    await act(async () => {});
    expect(handlers.range).not.toHaveBeenCalled();
    act(() => {
      store.dispatch(Session.Persist.endSwap());
    });
    await waitFor(() =>
      expect(handlers.range).toHaveBeenCalledWith(
        expect.objectContaining({ key: "r1" }),
      ),
    );
  });

  it("should hold a launch link until the workspace settles", async () => {
    const { store, connect, handlers } = await setupSettled(
      { getCurrentURLs: async () => ["synnax://cluster/c1/schematic/s1"] },
      (pre) => pre.dispatch(Session.Persist.beginSwap()),
    );
    await waitFor(() => expect(connect).toHaveBeenCalledWith("c1"));
    await act(async () => {});
    expect(handlers.schematic).not.toHaveBeenCalled();
    act(() => {
      store.dispatch(Session.Persist.endSwap());
    });
    await waitFor(() =>
      expect(handlers.schematic).toHaveBeenCalledWith(
        expect.objectContaining({ key: "s1" }),
      ),
    );
  });

  it("should invoke the handler only once the workspace has settled", async () => {
    const h = await setupSettled();
    await waitFor(() => expect(h.settled()).toBe(true));
    let settledAtInvoke: boolean | null = null;
    h.handlers.range.mockImplementation(async () => {
      settledAtInvoke = h.settled();
    });
    act(() => {
      h.store.dispatch(Session.Persist.beginSwap());
    });
    h.openURL(["synnax://cluster/c1/range/r1"]);
    await waitFor(() => expect(h.connect).toHaveBeenCalledWith("c1"));
    act(() => {
      h.store.dispatch(Session.Persist.endSwap());
    });
    await waitFor(() => expect(h.handlers.range).toHaveBeenCalled());
    expect(settledAtInvoke).toBe(true);
  });

  it("should hold every link fired while unsettled and release them together", async () => {
    const h = await setupSettled();
    await waitFor(() => expect(h.settled()).toBe(true));
    act(() => {
      h.store.dispatch(Session.Persist.beginSwap());
    });
    h.openURL(["synnax://cluster/c1/range/r1"]);
    h.openURL(["synnax://cluster/c1/schematic/s1"]);
    await waitFor(() => expect(h.connect).toHaveBeenCalledTimes(2));
    await act(async () => {});
    expect(h.handlers.range).not.toHaveBeenCalled();
    expect(h.handlers.schematic).not.toHaveBeenCalled();
    act(() => {
      h.store.dispatch(Session.Persist.endSwap());
    });
    await waitFor(() => {
      expect(h.handlers.range).toHaveBeenCalledWith(
        expect.objectContaining({ key: "r1" }),
      );
      expect(h.handlers.schematic).toHaveBeenCalledWith(
        expect.objectContaining({ key: "s1" }),
      );
    });
  });

  it("should hold a link when connecting itself unsettles the workspace", async () => {
    const h = await setupSettled();
    await waitFor(() => expect(h.settled()).toBe(true));
    // Models a Core switch: the client swap unsettles before connect resolves.
    h.connect.mockImplementation(async () => {
      h.store.dispatch(Session.Persist.beginSwap());
      return client();
    });
    h.openURL(["synnax://cluster/c2/schematic/s2"]);
    await waitFor(() => expect(h.connect).toHaveBeenCalledWith("c2"));
    await act(async () => {});
    expect(h.handlers.schematic).not.toHaveBeenCalled();
    act(() => {
      h.store.dispatch(Session.Persist.endSwap());
    });
    await waitFor(() =>
      expect(h.handlers.schematic).toHaveBeenCalledWith(
        expect.objectContaining({ key: "s2" }),
      ),
    );
  });

  it("should drop a held link superseded by a Core switch", async () => {
    const h = await setupSettled();
    await waitFor(() => expect(h.settled()).toBe(true));
    act(() => {
      h.store.dispatch(Session.Persist.beginSwap());
    });
    h.openURL(["synnax://cluster/c1/range/r1"]);
    await waitFor(() => expect(h.connect).toHaveBeenCalledWith("c1"));
    // The second link switches Cores, so the provider swaps to a fresh client.
    h.connect.mockImplementation(async () => {
      h.store.dispatch(Session.Core.select(OTHER_CORE_KEY));
      return client();
    });
    h.openURL(["synnax://cluster/c2/schematic/s2"]);
    await waitFor(() => expect(h.connect).toHaveBeenCalledWith("c2"));
    await act(async () => {});
    act(() => {
      h.store.dispatch(Session.Persist.endSwap());
    });
    await waitFor(() =>
      expect(h.handlers.schematic).toHaveBeenCalledWith(
        expect.objectContaining({ key: "s2" }),
      ),
    );
    await act(async () => {});
    expect(h.handlers.range).not.toHaveBeenCalled();
  });

  it("should run only the latest of two links racing to different Cores", async () => {
    const h = await setupSettled();
    await waitFor(() => expect(h.settled()).toBe(true));
    act(() => {
      h.store.dispatch(Session.Persist.beginSwap());
    });
    h.openURL(["synnax://cluster/c1/range/r1"]);
    await waitFor(() => expect(h.connect).toHaveBeenCalledWith("c1"));
    h.connect.mockImplementation(async () => {
      h.store.dispatch(Session.Core.select(OTHER_CORE_KEY));
      return client();
    });
    h.openURL(["synnax://cluster/c2/range/r2"]);
    await waitFor(() => expect(h.connect).toHaveBeenCalledWith("c2"));
    await act(async () => {});
    act(() => {
      h.store.dispatch(Session.Persist.endSwap());
    });
    await waitFor(() =>
      expect(h.handlers.range).toHaveBeenCalledWith(
        expect.objectContaining({ key: "r2" }),
      ),
    );
    await act(async () => {});
    expect(h.handlers.range).toHaveBeenCalledTimes(1);
  });

  it("should drop a held launch link superseded by a runtime Core switch", async () => {
    const h = await setupSettled(
      { getCurrentURLs: async () => ["synnax://cluster/c1/range/r1"] },
      (pre) => pre.dispatch(Session.Persist.beginSwap()),
    );
    await waitFor(() => expect(h.connect).toHaveBeenCalledWith("c1"));
    h.connect.mockImplementation(async () => {
      h.store.dispatch(Session.Core.select(OTHER_CORE_KEY));
      return client();
    });
    h.openURL(["synnax://cluster/c2/schematic/s2"]);
    await waitFor(() => expect(h.connect).toHaveBeenCalledWith("c2"));
    await act(async () => {});
    act(() => {
      h.store.dispatch(Session.Persist.endSwap());
    });
    await waitFor(() =>
      expect(h.handlers.schematic).toHaveBeenCalledWith(
        expect.objectContaining({ key: "s2" }),
      ),
    );
    await act(async () => {});
    expect(h.handlers.range).not.toHaveBeenCalled();
  });

  it("should drop every link superseded by a later Core switch", async () => {
    const h = await setupSettled();
    await waitFor(() => expect(h.settled()).toBe(true));
    const switchTo = (key: string) => async () => {
      h.store.dispatch(Session.Core.select(key));
      return client();
    };
    act(() => {
      h.store.dispatch(Session.Persist.beginSwap());
    });
    h.openURL(["synnax://cluster/c1/range/r1"]);
    await waitFor(() => expect(h.connect).toHaveBeenCalledWith("c1"));
    h.connect.mockImplementation(switchTo(OTHER_CORE_KEY));
    h.openURL(["synnax://cluster/c2/schematic/s2"]);
    await waitFor(() => expect(h.connect).toHaveBeenCalledWith("c2"));
    h.connect.mockImplementation(switchTo(THIRD_CORE_KEY));
    h.openURL(["synnax://cluster/c3/range/r3"]);
    await waitFor(() => expect(h.connect).toHaveBeenCalledWith("c3"));
    await act(async () => {});
    act(() => {
      h.store.dispatch(Session.Persist.endSwap());
    });
    await waitFor(() =>
      expect(h.handlers.range).toHaveBeenCalledWith(
        expect.objectContaining({ key: "r3" }),
      ),
    );
    await act(async () => {});
    expect(h.handlers.range).toHaveBeenCalledTimes(1);
    expect(h.handlers.schematic).not.toHaveBeenCalled();
  });

  it("should not run a timed-out link when the workspace settles later", async () => {
    const h = await setupSettled();
    await waitFor(() => expect(h.settled()).toBe(true));
    act(() => {
      h.store.dispatch(Session.Persist.beginSwap());
    });
    vi.useFakeTimers();
    h.openURL(["synnax://cluster/c1/range/r1"]);
    await act(async () => {});
    expect(h.connect).toHaveBeenCalledWith("c1");
    await act(async () => {
      vi.advanceTimersByTime(Number(TimeSpan.seconds(31).milliseconds));
    });
    vi.useRealTimers();
    await act(async () => {});
    act(() => {
      h.store.dispatch(Session.Persist.endSwap());
    });
    await waitFor(() => expect(h.settled()).toBe(true));
    await act(async () => {});
    expect(h.handlers.range).not.toHaveBeenCalled();
  });

  it("should fail a link with an error status when the workspace never settles", async () => {
    const h = await setupSettled();
    await waitFor(() => expect(h.settled()).toBe(true));
    act(() => {
      h.store.dispatch(Session.Persist.beginSwap());
    });
    vi.useFakeTimers();
    h.openURL(["synnax://cluster/c1/range/r1"]);
    await act(async () => {});
    expect(h.connect).toHaveBeenCalledWith("c1");
    await act(async () => {
      vi.advanceTimersByTime(Number(TimeSpan.seconds(31).milliseconds));
    });
    vi.useRealTimers();
    await waitFor(() =>
      expect(h.statuses().some((s) => s.message.includes("Failed to open"))).toBe(true),
    );
    expect(h.handlers.range).not.toHaveBeenCalled();
  });

  it("should hold a link fired after a settle cycle re-arms the wait", async () => {
    const h = await setupSettled();
    await waitFor(() => expect(h.settled()).toBe(true));
    act(() => {
      h.store.dispatch(Session.Persist.beginSwap());
    });
    act(() => {
      h.store.dispatch(Session.Persist.endSwap());
    });
    await waitFor(() => expect(h.settled()).toBe(true));
    act(() => {
      h.store.dispatch(Session.Persist.beginSwap());
    });
    h.openURL(["synnax://cluster/c1/range/r2"]);
    await waitFor(() => expect(h.connect).toHaveBeenCalledWith("c1"));
    await act(async () => {});
    expect(h.handlers.range).not.toHaveBeenCalled();
    act(() => {
      h.store.dispatch(Session.Persist.endSwap());
    });
    await waitFor(() =>
      expect(h.handlers.range).toHaveBeenCalledWith(
        expect.objectContaining({ key: "r2" }),
      ),
    );
  });
});
