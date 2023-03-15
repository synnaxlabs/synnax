// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Synnax } from "@synnaxlabs/client";
import type { SynnaxProps } from "@synnaxlabs/client";

import type { ConnectionState } from "@/cluster/types";

/**
 * Tests the connection to the cluster with the given props.
 *
 * @param props - The connection properties to test.
 * @returns The cluster key and connection state. If unsuccessful, the cluster key
 * will be undefined.
 */
export const testConnection = async (
  props: SynnaxProps
): Promise<{ clusterKey: string | undefined; state: ConnectionState }> => {
  const conn = new Synnax(props);
  await conn.connectivity.check();
  conn.close();
  return {
    clusterKey: conn.connectivity.clusterKey,
    state: {
      status: conn.connectivity.status(),
      message: conn.connectivity.statusMessage(),
    },
  };
};
