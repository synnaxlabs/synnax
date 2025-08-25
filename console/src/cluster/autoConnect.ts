// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Cluster } from "@/cluster/slice";
import { RUNTIME } from "@/runtime";

export interface ConnectionParams extends Pick<Cluster, "host" | "port" | "secure"> {}

export const detectServingConnection = (): ConnectionParams | null => {
  if (RUNTIME === "tauri") return null;
  let host = "localhost";
  let port = 9090;
  let secure = false;
  console.log(IS_DEV);
  if (!IS_DEV) {
    const url = new URL(window.location.origin);
    host = url.hostname;
    port = url.port ? parseInt(url.port) : url.protocol === "https:" ? 443 : 80;
    secure = url.protocol === "https:";
  }
  return { host, port, secure };
};
