// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift } from "@synnaxlabs/drift";
import { Status, Synnax, useAsyncEffect, useSyncedRef } from "@synnaxlabs/pluto";
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

// A link outlives the renders that turn its preconditions true, so callers park here
// and an effect releases them when met transitions. A timeout rejects parked calls.
const useWaitFor = (
  met: boolean,
  timeout?: TimeSpan,
  timeoutMessage?: string,
): (() => Promise<void>) => {
  const metRef = useRef(met);
  metRef.current = met;
  const waitersRef = useRef<(() => void)[]>([]);
  useEffect(() => {
    if (!met) return;
    waitersRef.current.forEach((resolve) => resolve());
    waitersRef.current = [];
  }, [met]);
  return async (): Promise<void> => {
    if (metRef.current) return;
    await new Promise<void>((resolve, reject) => {
      let timer: ReturnType<typeof setTimeout> | undefined;
      if (timeout != null)
        timer = setTimeout(
          () => reject(new Error(timeoutMessage)),
          timeout.milliseconds,
        );
      waitersRef.current.push(() => {
        clearTimeout(timer);
        resolve();
      });
    });
  };
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
  const awaitSettled = useWaitFor(
    Session.useSettled(),
    SETTLE_TIMEOUT,
    "Timed out waiting for the workspace to settle",
  );
  const waitProject = useWaitFor(Session.Project.useSelectIsAnySelected());
  const awaitProject = async (): Promise<void> => {
    dispatch(Session.Link.beginProjectWait());
    try {
      await waitProject();
    } finally {
      dispatch(Session.Link.endProjectWait());
    }
  };
  const awaitPanel = useWaitFor(Session.Panel.useSelectSelected() != null);
  // A parked link outlives the handlers and client of the render that received it.
  // Synced refs resolve both at invoke time.
  const handlersRef = useSyncedRef(handlers);
  const clientRef = useSyncedRef(Synnax.use());
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

      // A link opens only into a ready workspace. It waits for the workspace to settle,
      // then for a selected project, then for a selected panel to place the tab in.
      await awaitSettled();
      await awaitProject();
      await awaitPanel();
      // The workspace fills the panel cache lazily, so a selection the synchronizers
      // repaired can stay cold. Retrieve it before placement.
      const panelKey = Session.Panel.selectSelected(store.getState());
      if (panelKey != null) await clientRef.current?.panels.retrieve(panelKey);
      // A later link that switched Cores supersedes this one. The handler would run
      // against the new Core's session, so drop it.
      if (Session.Core.selectSelectedKey(store.getState()) !== coreKey) return;

      const resource = urlParts[1];
      const resourceKey = urlParts[2];
      const handle = handlersRef.current[resource];
      if (handle == null)
        throw new Error(`Resource type "${resource}" is unknown to Synnax`);
      await handle({ client: clientRef.current ?? client, key: resourceKey });
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
