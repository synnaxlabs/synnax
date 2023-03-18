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

import type { ConnectionState } from "@/cluster/core";

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
  const client = new Synnax(props);
  await client.connectivity.check();
  client.close();
  return {
    clusterKey: client.connectivity.clusterKey,
    state: {
      status: client.connectivity.status(),
      message: client.connectivity.statusMessage(),
    },
  };
};
