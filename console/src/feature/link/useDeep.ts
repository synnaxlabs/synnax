// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift } from "@synnaxlabs/drift";
import { Status, useAsyncEffect } from "@synnaxlabs/pluto";
import { strings, TimeSpan } from "@synnaxlabs/x";
import { type UnlistenFn } from "@tauri-apps/api/event";
import { getCurrent, onOpenUrl } from "@tauri-apps/plugin-deep-link";
import { useEffect, useRef } from "react";

import { Link } from "@/platform/link";
import { Session } from "@/session";

const BASE_LINK = `${Link.PREFIX}<cluster-key>`;

const INCORRECT_FORMAT_ERROR_MESSAGE = `Links must be of the form ${BASE_LINK} or ${BASE_LINK}/<resource>/<resource-key>`;

const SETTLE_TIMEOUT = TimeSpan.seconds(30);

// Deps are the runtime bindings useDeep relies on. They default to the live Tauri
// deep-link plugin and runtime engine; tests inject fakes to drive links without Tauri.
export interface Deps {
  engine: Session.Runtime.Engine;
  getCurrentURLs: () => Promise<string[] | null>;
  onOpenURL: (handler: (urls: string[]) => void) => Promise<UnlistenFn>;
}

const DEFAULT_DEPS: Deps = {
  engine: Session.Runtime.ENGINE,
  getCurrentURLs: getCurrent,
  onOpenURL: onOpenUrl,
};

export const useDeep = (
  connect: Link.Connect,
  handlers: Record<string, Link.Handler>,
  deps: Deps = DEFAULT_DEPS,
): void => {
  // While early returns are usually bad in hooks, this is fine because the engine is a
  // constant and so the hook will be the exact same for a given runtime.
  if (deps.engine !== "tauri") return;
  const handleError = Status.useErrorHandler();
  const dispatch = Session.useDispatch();
  const store = Session.useStore();
  const settled = Session.useSettled();
  const settledRef = useRef(settled);
  settledRef.current = settled;
  const settledWaitersRef = useRef<(() => void)[]>([]);
  useEffect(() => {
    if (!settled) return;
    settledWaitersRef.current.forEach((resolve) => resolve());
    settledWaitersRef.current = [];
  }, [settled]);
  const awaitSettled = async (): Promise<void> => {
    if (settledRef.current) return;
    await new Promise<void>((resolve, reject) => {
      settledWaitersRef.current.push(resolve);
      setTimeout(
        () => reject(new Error("Timed out waiting for the workspace to settle")),
        SETTLE_TIMEOUT.milliseconds,
      );
    });
  };
  const urlHandler = async (urls: string[]) => {
    try {
      dispatch(Drift.focusWindow({}));

      if (urls.length === 0 || !urls[0].startsWith(Link.PREFIX))
        throw new Error(INCORRECT_FORMAT_ERROR_MESSAGE);
      const urlParts = urls[0].slice(Link.PREFIX.length).split("/");
      if (urlParts.length !== 1 && urlParts.length !== 3)
        throw new Error(INCORRECT_FORMAT_ERROR_MESSAGE);

      const client = await connect(urlParts[0]);
      if (urlParts.length === 1) return;
      const coreKey = Session.Core.selectSelectedKey(store.getState());

      // Handlers resolve their target through the query cache, which the session
      // synchronizers fill after connect. On a cold launch or a Core switch the
      // handler would otherwise read the cache before its first reconcile pass.
      await awaitSettled();
      // A later link that switched Cores supersedes this one. The handler would run
      // against the new Core's session, so drop it.
      if (Session.Core.selectSelectedKey(store.getState()) !== coreKey) return;

      const resource = urlParts[1];
      const resourceKey = urlParts[2];
      const handle = handlers[resource];
      if (handle == null)
        throw new Error(`Resource type "${resource}" is unknown to Synnax`);
      await handle({ client, key: resourceKey });
    } catch (e) {
      handleError(e, `Failed to open ${strings.naturalLanguageJoin(urls, "link")}`);
    }
  };

  // Handles the case where the app is opened from a link
  useAsyncEffect(async (signal) => {
    const urls = await deps.getCurrentURLs();
    // A hard reload re-runs this effect with the same launch link; skip it once.
    if (localStorage.getItem(Link.SHOULD_IGNORE_KEY) === "true") {
      localStorage.setItem(Link.SHOULD_IGNORE_KEY, "false");
      return;
    }
    if (urls == null || signal.aborted) return;
    await urlHandler(urls);
  }, []);

  // Handles the case where the app is open and a link gets called
  useAsyncEffect(async () => await deps.onOpenURL((urls) => void urlHandler(urls)), []);
};
