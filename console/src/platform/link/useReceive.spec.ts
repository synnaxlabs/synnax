// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { scheduler } from "@synnaxlabs/x";
import { type UnlistenFn } from "@tauri-apps/api/event";
import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { Link } from "@/platform/link";

const FIRST = ["synnax://cluster/c1"];
const SECOND = ["synnax://cluster/c2"];

interface Harness {
  handle: ReturnType<typeof vi.fn<(urls: string[]) => void>>;
  openURL: (urls: string[]) => void;
  /** reload mounts the hook again, as a reload does, holding the given link. */
  reload: (held: string[] | null) => Promise<void>;
}

const mount = async (
  handle: Harness["handle"],
  held: string[] | null,
): Promise<(urls: string[]) => void> => {
  let openURL: (urls: string[]) => void = () => {};
  const onOpenURL = vi.fn(async (h: typeof openURL): Promise<UnlistenFn> => {
    openURL = h;
    return () => {};
  });
  renderHook(() =>
    Link.useReceive(handle, {
      engine: "tauri",
      getCurrentURLs: async () => held,
      onOpenURL,
    }),
  );
  await waitFor(() => expect(onOpenURL).toHaveBeenCalled());
  await act(async () => await scheduler.flushTaskQueue());
  return (urls) => openURL(urls);
};

const setup = async (held: string[] | null): Promise<Harness> => {
  const handle = vi.fn<(urls: string[]) => void>();
  let openURL = await mount(handle, held);
  return {
    handle,
    openURL: (urls) => openURL(urls),
    reload: async (next) => {
      openURL = await mount(handle, next);
    },
  };
};

describe("Link.useReceive", () => {
  beforeEach(() => sessionStorage.clear());

  it("should pass on the link the app holds on load", async () => {
    const h = await setup(FIRST);
    expect(h.handle.mock.calls).toEqual([[FIRST]]);
  });

  it("should not pass on the held link again after a reload", async () => {
    const h = await setup(FIRST);
    await h.reload(FIRST);
    expect(h.handle.mock.calls).toEqual([[FIRST]]);
  });

  it("should pass on a held link that arrived during a reload", async () => {
    const h = await setup(FIRST);
    await h.reload(SECOND);
    expect(h.handle.mock.calls).toEqual([[FIRST], [SECOND]]);
  });

  it("should pass on every link opened while running, repeats included", async () => {
    const h = await setup(null);
    act(() => h.openURL(FIRST));
    act(() => h.openURL(FIRST));
    expect(h.handle.mock.calls).toEqual([[FIRST], [FIRST]]);
  });

  it("should not pass on a link opened while running again after a reload", async () => {
    const h = await setup(null);
    act(() => h.openURL(SECOND));
    await h.reload(SECOND);
    expect(h.handle.mock.calls).toEqual([[SECOND]]);
  });

  it("should do nothing outside Tauri", async () => {
    const getCurrentURLs = vi.fn(async () => FIRST);
    const onOpenURL = vi.fn(async (): Promise<UnlistenFn> => () => {});
    const handle = vi.fn();
    renderHook(() =>
      Link.useReceive(handle, { engine: "web", getCurrentURLs, onOpenURL }),
    );
    await act(async () => await scheduler.flushTaskQueue());
    expect(getCurrentURLs).not.toHaveBeenCalled();
    expect(onOpenURL).not.toHaveBeenCalled();
    expect(handle).not.toHaveBeenCalled();
  });
});
