import { Connectivity, Synnax, TimeSpan } from "@synnaxlabs/client";
import { useEffect } from "react";
import { useDispatch } from "react-redux";
import clientPool from "./ClientPool";
import { setClusterConnectionState, useSelectActiveCluster } from "./slice";

export const useActiveClient = (): Synnax | undefined => {
  const activeCluster = useSelectActiveCluster();
  if (!activeCluster) return undefined;
  return clientPool.acquire(activeCluster.key);
};

export const useClientConnector = () => {
  const dispatch = useDispatch();
  const activeCluster = useSelectActiveCluster();
  useEffect(() => {
    if (!activeCluster) return clientPool.closeAll();
    const { key, props } = activeCluster;

    const existingC = clientPool.acquire(key);
    if (existingC && existingC.connectivity.status() === Connectivity.CONNECTED)
      return;

    let newC = new Synnax({
      ...props,
      connectivityPollFrequency: TimeSpan.Seconds(5),
    });
    clientPool.set(key, newC);
    newC.connectivity.onChange((status, error, message) => {
      dispatch(
        setClusterConnectionState({
          key,
          state: { status, message },
        })
      );
    });
  }, [dispatch, activeCluster]);
};
