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

const DEV_CONNECTION: ConnectionParams = {
  key: Session.Core.SERVED_KEY,
  name: "Core",
  host: "localhost",
  port: 9090,
  secure: false,
};

/**
 * Overrides the dev connection's port, for tooling (the docs video studio) that
 * runs the dev Console against a Core on a non-default port.
 */
const DEV_PORT_KEY = "synnax-dev-connection-port";

/**
 * Drops the dev connection, so the dev Console starts on the Cores list like the
 * desktop build. The same tooling uses it to record the connection flow, which
 * the browser build otherwise skips.
 */
const DEV_DETACH_KEY = "synnax-dev-connection-detach";

const devConnection = (): ConnectionParams | null => {
  if (localStorage.getItem(DEV_DETACH_KEY) != null) return null;
  const port = Number(localStorage.getItem(DEV_PORT_KEY));
  if (!Number.isInteger(port) || port <= 0) return DEV_CONNECTION;
  return { ...DEV_CONNECTION, port };
};

export const detectConnection = (): ConnectionParams | null => {
  if (Session.Runtime.ENGINE === "tauri") return null;
  if (new URLSearchParams(window.location.search).has("select-core")) return null;
  if (IS_DEV) return devConnection();
  const url = new URL(window.location.origin);
  return {
    key: Session.Core.SERVED_KEY,
    name: "Core",
    host: url.hostname,
    port: url.port ? parseInt(url.port, 10) : url.protocol === "https:" ? 443 : 80,
    secure: url.protocol === "https:",
  };
};
