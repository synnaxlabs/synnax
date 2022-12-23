import { Synnax } from "@synnaxlabs/client";
import type { SynnaxProps } from "@synnaxlabs/client";

import type { ConnectionState } from "../types";

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
