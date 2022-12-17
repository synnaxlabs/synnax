import {
  createContext,
  PropsWithChildren,
  useContext,
  useEffect,
  useState,
} from "react";

import { Synnax, TimeSpan } from "@synnaxlabs/client";
import {
  completeProcess,
  registerProcess,
  useWindowLifecycle,
} from "@synnaxlabs/drift";
import { useDispatch } from "react-redux";

import {
  useSelectActiveCluster,
  setClusterConnectionState,
  useSelectActiveClusterKey,
} from "../store";

export interface ClusterContextProps extends PropsWithChildren<any> {}

export interface ClusterContextValue {
  client: Synnax | null;
  trigger: boolean;
}

export const ClusterContext = createContext<ClusterContextValue>({
  client: null,
  trigger: false,
});

export const useClusterClient = (): Synnax | null => useContext(ClusterContext).client;

export const ClusterProvider = ({ children }: ClusterContextProps): JSX.Element => {
  const [s, setClient] = useState<{
    client: Synnax | null;
    trigger: boolean;
  }>({ client: null, trigger: false });
  const dispatch = useDispatch();
  const activeCluster = useSelectActiveCluster();
  const activeClusterKey = useSelectActiveClusterKey();

  useEffect(() => {
    if (activeCluster == null) return;

    const { key, props } = activeCluster;

    const c = new Synnax({
      ...props,
      connectivityPollFrequency: TimeSpan.Seconds(5),
    });

    c.connectivity.onChange((status, error, message) => {
      dispatch(
        setClusterConnectionState({
          key,
          state: { status, message },
        })
      );
    });

    setClient((prev) => ({ client: c, trigger: !prev.trigger }));

    return () => {
      if (s.client != null) s.client.close();
      setClient({ client: null, trigger: false });
    };
  }, [activeClusterKey]);

  useWindowLifecycle(() => {
    dispatch(registerProcess());
    return () => {
      if (s.client != null) s.client.close();
      dispatch(completeProcess());
    };
  });

  return <ClusterContext.Provider value={s}>{children}</ClusterContext.Provider>;
};
