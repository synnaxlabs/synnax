// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useAsyncEffect, useSyncedRef } from "@synnaxlabs/lyra/hooks";

import { DEFAULT_DEPS, type Deps } from "@/platform/link/deps";

// Tauri returns the last link the app received until the app quits. Session storage
// lasts as long: a reload keeps it, and a restart clears it.
const RECEIVED_KEY = "link.received";

/**
 * useReceive calls handle with each link the app receives: the link it holds on load,
 * then every link opened while it runs. A reload passes a link on again only if handle
 * had not finished with it, so a link still waiting on the user survives the reload.
 * handle reports its own errors. Mount one per window: callers share one record, so a
 * second would miss the held link.
 */
export const useReceive = (
  handle: (urls: string[]) => void | Promise<void>,
  deps: Deps = DEFAULT_DEPS,
): void => {
  const enabled = deps.engine === "tauri";
  const handleRef = useSyncedRef(handle);
  const receive = async (urls: string[]): Promise<void> => {
    await handleRef.current(urls);
    sessionStorage.setItem(RECEIVED_KEY, JSON.stringify(urls));
  };

  useAsyncEffect(
    async (signal) => {
      if (!enabled) return;
      const urls = await deps.getCurrentURLs();
      if (urls == null || signal.aborted) return;
      if (sessionStorage.getItem(RECEIVED_KEY) === JSON.stringify(urls)) return;
      await receive(urls);
    },
    [enabled],
  );

  useAsyncEffect(async () => {
    if (!enabled) return;
    return await deps.onOpenURL((urls) => void receive(urls));
  }, [enabled]);
};
