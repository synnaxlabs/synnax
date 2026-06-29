// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";
import { type Synnax as Client } from "@synnaxlabs/client";
import { type UnlistenFn } from "@tauri-apps/api/event";
import { waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { Link } from "@/layered/service/link";
import { renderHookWithConsole } from "@/testUtils";

// useDeep dispatches Drift.focusWindow as a side effect of routing. Routing itself does
// not depend on window state, so an identity-reducer store lets that dispatch no-op
// while we assert on connect/handler calls.
const noopStore = () => configureStore({ reducer: (state = {}) => state });

const client = (): Client => ({}) as Client;

interface Harness {
  connect: ReturnType<typeof vi.fn>;
  handlers: Record<string, ReturnType<typeof vi.fn>>;
  deps: Link.Deps;
  openURL: (urls: string[]) => void;
}

const setup = (overrides: Partial<Link.Deps> = {}): Harness => {
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
  renderHookWithConsole(() => Link.useDeep(connect, handlers, deps), {
    store: noopStore(),
  });
  return { connect, handlers, deps, openURL: (urls) => openURL(urls) };
};

describe("useDeep", () => {
  beforeEach(() => localStorage.clear());

  it("should open a resource link the app was launched from", async () => {
    const { connect, handlers } = setup({
      getCurrentURLs: async () => ["synnax://cluster/c1/schematic/s1"],
    });
    await waitFor(() => expect(connect).toHaveBeenCalledWith("c1"));
    await waitFor(() =>
      expect(handlers.schematic).toHaveBeenCalledWith(
        expect.objectContaining({ key: "s1" }),
      ),
    );
  });

  it("should connect without placing a layout for a cluster-only link", async () => {
    const { connect, handlers } = setup({
      getCurrentURLs: async () => ["synnax://cluster/c1"],
    });
    await waitFor(() => expect(connect).toHaveBeenCalledWith("c1"));
    expect(handlers.schematic).not.toHaveBeenCalled();
    expect(handlers.range).not.toHaveBeenCalled();
  });

  it("should route a link opened while the app is already running", async () => {
    const { connect, handlers, deps, openURL } = setup();
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
    const { connect } = setup({ getCurrentURLs: async () => ["not-a-synnax-link"] });
    await new Promise((resolve) => setTimeout(resolve, 50));
    expect(connect).not.toHaveBeenCalled();
  });

  it("should connect but place nothing for an unknown resource type", async () => {
    const { connect, handlers } = setup({
      getCurrentURLs: async () => ["synnax://cluster/c1/widget/w1"],
    });
    await waitFor(() => expect(connect).toHaveBeenCalledWith("c1"));
    expect(handlers.schematic).not.toHaveBeenCalled();
    expect(handlers.range).not.toHaveBeenCalled();
  });

  it("should ignore the next launch link once when the ignore flag is set", async () => {
    Link.markNextIgnored();
    const first = setup({
      getCurrentURLs: async () => ["synnax://cluster/c1/schematic/s1"],
    });
    await new Promise((resolve) => setTimeout(resolve, 50));
    expect(first.connect).not.toHaveBeenCalled();
    // The flag is single-use: the hook clears it, so a later launch is handled normally.
    const second = setup({
      getCurrentURLs: async () => ["synnax://cluster/c1/schematic/s1"],
    });
    await waitFor(() => expect(second.connect).toHaveBeenCalledWith("c1"));
  });

  it("should be a no-op when the engine is not tauri", async () => {
    const getCurrentURLs = vi.fn(async () => ["synnax://cluster/c1/schematic/s1"]);
    const { connect } = setup({ engine: "web", getCurrentURLs });
    await new Promise((resolve) => setTimeout(resolve, 50));
    expect(getCurrentURLs).not.toHaveBeenCalled();
    expect(connect).not.toHaveBeenCalled();
  });
});
