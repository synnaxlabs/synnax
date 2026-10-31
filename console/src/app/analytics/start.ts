// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { getVersion } from "@tauri-apps/api/app";
import { v7 } from "uuid";

import { watchAccount } from "@/app/analytics/account";
import { follow } from "@/app/analytics/core";
import { type Install, retrieveInstall } from "@/app/analytics/install";
import { create, type Params, type SharedSession } from "@/app/analytics/posthog";
import { Analytics } from "@/platform/analytics";
import { Session } from "@/session";

/**
 * Starts analytics for this window and returns the sink the app reports through. The
 * vendor loads only while the store says the user allows analytics, and stops when the
 * user refuses. The Console build returns the sink that discards everything, and the
 * dead branch takes every vendor with it.
 */
export const start = (store: Session.Store): Analytics.Sink => {
  if (!DESKTOP || Session.Runtime.ENGINE !== "tauri") return Analytics.NOOP;
  const install = retrieveInstall();
  const transport = create(identify(install), storeSession(store));
  const sink = Analytics.createSink(transport);
  watchAccount({ store, sink, transport });
  // Every window reports its own screens and actions, but a launch opens the app once,
  // and runs one embedded Core.
  if (Session.Runtime.isMainWindow()) {
    void open(sink, install);
    void follow(sink);
  }
  return sink;
};

/** Names the install that PostHog reports as. */
const identify = async (install: Promise<Install>): Promise<Params> => ({
  installID: (await install).id,
});

/**
 * Keeps the PostHog session in the store, where every window sees it. The main window
 * starts a new session each launch, and every other window waits for it.
 */
const storeSession = (store: Session.Store): SharedSession => {
  const current = (): string | undefined =>
    Session.Analytics.selectSessionID(store.getState());
  const share = (id: string): void => {
    if (id !== current()) store.dispatch(Session.Analytics.setSessionID(id));
  };
  const subscribe = (listener: (id: string) => void): (() => void) => {
    let last = current();
    return store.subscribe(() => {
      const next = current();
      if (next == null || next === last) return;
      last = next;
      listener(next);
    });
  };
  const join = async (): Promise<string> => {
    if (Session.Runtime.isMainWindow()) {
      // PostHog takes only a UUIDv7, since it reads the start of the session from it.
      const id = v7();
      share(id);
      return id;
    }
    const id = current();
    if (id != null) return id;
    return await new Promise((resolve) => {
      const unsubscribe = subscribe((next) => {
        unsubscribe();
        resolve(next);
      });
    });
  };
  return { join, current, share, subscribe };
};

/**
 * Reports the launch, and a reset that ran before it, once the install and the version
 * of the app are known.
 */
const open = async (sink: Analytics.Sink, install: Promise<Install>): Promise<void> => {
  try {
    const [record, version] = await Promise.all([install, getVersion()]);
    sink.capture("app_opened", {
      version,
      os: record.os,
      arch: record.arch,
      first_launch: record.firstLaunch,
      hours_since_last_launch: record.hoursSinceLastLaunch,
    });
    // A reset restarts the app before PostHog can send, so the launch after it reports.
    if (record.erasedBytes != null)
      sink.capture("core_reset", { data_size_bytes: record.erasedBytes });
  } catch (err) {
    console.error("failed to report the launch", err);
  }
};
