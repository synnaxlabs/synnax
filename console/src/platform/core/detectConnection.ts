// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Session } from "@/session";

export interface ConnectionParams extends Pick<
  Session.Core.Core,
  "key" | "name" | "host" | "port" | "secure"
> {}

/**
 * Pins the dev Console to the Core on this port, as the browser build pins to the Core
 * that serves it. Test tooling (the integration tests and the docs video studio) uses
 * it to run single-Core flows on the dev Console, which otherwise starts on the Cores
 * list like the desktop build.
 */
const DEV_PORT_KEY = "synnax-dev-connection-port";

/** Drops the pin, so the same tooling can record the connection flow. */
const DEV_DETACH_KEY = "synnax-dev-connection-detach";

const devConnection = (): ConnectionParams | null => {
  if (localStorage.getItem(DEV_DETACH_KEY) != null) return null;
  const port = Number(localStorage.getItem(DEV_PORT_KEY));
  if (!Number.isInteger(port) || port <= 0) return null;
  return {
    key: Session.Core.SERVED_KEY,
    name: "Core",
    host: "localhost",
    port,
    secure: false,
  };
};

const pageConnection = (origin: string): ConnectionParams => {
  const url = new URL(origin);
  return {
    key: Session.Core.SERVED_KEY,
    name: "Core",
    host: url.hostname,
    port: url.port ? parseInt(url.port, 10) : url.protocol === "https:" ? 443 : 80,
    secure: url.protocol === "https:",
  };
};

export const detectConnection = (): ConnectionParams | null => {
  if (Session.Runtime.ENGINE === "tauri") return null;
  if (new URLSearchParams(window.location.search).has("select-core")) return null;
  if (IS_DEV) return devConnection();
  return pageConnection(window.location.origin);
};

/** Returns true when the browser page at the origin came from the Core. */
export const servesPage = (
  { host, port }: Pick<ConnectionParams, "host" | "port">,
  origin: string,
): boolean => {
  if (Session.Runtime.ENGINE === "tauri") return false;
  const page = pageConnection(origin);
  return page.host === host && page.port === port;
};
