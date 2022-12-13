import { Synnax, SynnaxProps } from "@synnaxlabs/client";
import { ConnectionState } from "@/features/cluster/types";

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
