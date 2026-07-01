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
import { strings } from "@synnaxlabs/x";
import { type UnlistenFn } from "@tauri-apps/api/event";
import { getCurrent, onOpenUrl } from "@tauri-apps/plugin-deep-link";
import { useDispatch } from "react-redux";

import { type ClusterConnect, type Handler, PREFIX } from "@/service/link/types";
import { Session } from "@/session";

const BASE_LINK = `${PREFIX}<cluster-key>`;

const INCORRECT_FORMAT_ERROR_MESSAGE = `Links must be of the form ${BASE_LINK} or ${BASE_LINK}/<resource>/<resource-key>`;

// The ignore flag lives in localStorage rather than Redux because a hard reload fires
// before Redux finishes persisting, and the flag must survive that reload so the
// relaunched window does not re-open the link it was opened from.
const SHOULD_IGNORE_KEY = "shouldIgnoreLink";

// markNextIgnored flags the next deep link to be skipped. Call it before triggering a
// hard reload so the relaunched window ignores the link that triggered the reload.
export const markNextIgnored = (): void =>
  localStorage.setItem(SHOULD_IGNORE_KEY, "true");

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
  connect: ClusterConnect,
  handlers: Record<string, Handler>,
  deps: Deps = DEFAULT_DEPS,
): void => {
  // While early returns are usually bad in hooks, this is fine because the engine is a
  // constant and so the hook will be the exact same for a given runtime.
  if (deps.engine !== "tauri") return;
  const handleError = Status.useErrorHandler();
  const dispatch = Session.useDispatch();
  const urlHandler = async (urls: string[]) => {
    try {
      dispatch(Drift.focusWindow({}));

      // Processing URL, making sure is has valid form
      if (urls.length === 0 || !urls[0].startsWith(PREFIX))
        throw new Error(INCORRECT_FORMAT_ERROR_MESSAGE);
      const urlParts = urls[0].slice(PREFIX.length).split("/");
      if (urlParts.length !== 1 && urlParts.length !== 3)
        throw new Error(INCORRECT_FORMAT_ERROR_MESSAGE);

      const clusterKey = urlParts[0];
      const client = await connect(clusterKey);
      if (urlParts.length === 1) return;

      const resource = urlParts[1];
      const resourceKey = urlParts[2];
      const handle = handlers[resource];
      if (handle == null)
        throw new Error(`Resource type "${resource}" is unknown to Synnax`);
      await handle({ client, key: resourceKey });
    } catch (e) {
      handleError(e, `Failed to open ${(strings.naturalLanguageJoin(urls), "link")}`);
    }
  };

  // Handles the case where the app is opened from a link
  useAsyncEffect(async (signal) => {
    const urls = await deps.getCurrentURLs();
    // A hard reload re-runs this effect with the same launch link; skip it once.
    if (localStorage.getItem(SHOULD_IGNORE_KEY) === "true") {
      localStorage.setItem(SHOULD_IGNORE_KEY, "false");
      return;
    }
    if (urls == null || signal.aborted) return;
    await urlHandler(urls);
  }, []);

  // Handles the case where the app is open and a link gets called
  useAsyncEffect(async () => await deps.onOpenURL((urls) => void urlHandler(urls)), []);
};
