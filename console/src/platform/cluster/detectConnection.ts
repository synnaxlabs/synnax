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
  Session.Cluster.Cluster,
  "name" | "host" | "port" | "secure"
> {}

const DEV_CONNECTION: ConnectionParams = {
  name: "Core",
  host: "localhost",
  port: 9090,
  secure: false,
};

/**
 * DEV_PORT_KEY is a localStorage key that overrides the dev connection's port,
 * for tooling (e.g. the docs video studio) that runs the dev Console against a
 * core on a non-default port.
 */
export const DEV_PORT_KEY = "synnax-dev-connection-port";

const devConnection = (): ConnectionParams => {
  const port = Number(localStorage.getItem(DEV_PORT_KEY));
  if (!Number.isInteger(port) || port <= 0) return DEV_CONNECTION;
  return { ...DEV_CONNECTION, port };
};

export const detectConnection = (): ConnectionParams | null => {
  if (Session.Runtime.ENGINE === "tauri") return null;
  if (IS_DEV) return devConnection();
  const url = new URL(window.location.origin);
  return {
    name: "Core",
    host: url.hostname,
    port: url.port ? parseInt(url.port, 10) : url.protocol === "https:" ? 443 : 80,
    secure: url.protocol === "https:",
  };
};
