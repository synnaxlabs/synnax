// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Cluster } from "@/cluster/slice";
import { Runtime } from "@/runtime";

export interface ConnectionParams extends Pick<
  Cluster,
  "name" | "host" | "port" | "secure"
> {}

const DEV_CONNECTION: ConnectionParams = {
  name: "Core",
  host: "localhost",
  port: 9090,
  secure: false,
};

export const detectConnection = (): ConnectionParams | null => {
  if (Runtime.ENGINE === "tauri") return null;
  if (IS_DEV) return DEV_CONNECTION;
  const url = new URL(window.location.origin);
  return {
    name: "Core",
    host: url.hostname,
    port: url.port ? parseInt(url.port) : url.protocol === "https:" ? 443 : 80,
    secure: url.protocol === "https:",
  };
};
