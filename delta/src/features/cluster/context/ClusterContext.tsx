// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  createContext,
  PropsWithChildren,
  useContext,
  useEffect,
  useState,
} from "react";

import { Synnax, TimeSpan } from "@synnaxlabs/client";
import { useDispatch } from "react-redux";

import {
  setClusterConnectionState,
  useSelectActiveClusterKey,
  useSelectCluster,
} from "../store";

interface ClusterContextValue {
  client: Synnax | null;
}

const ClusterContext = createContext<ClusterContextValue>({ client: null });

/**
 * @returns the connection to the active cluster or null if there is no active cluster.
 * It's important to note that this client is not guaranteed to have a stable connection.
 */
export const useClusterClient = (): Synnax | null => useContext(ClusterContext).client;

/** The props for the ClusterProvider component. */
export interface ClusterProviderProps extends PropsWithChildren<any> {}

/**
 * Provides the connection to the active cluster to all of its children. We're using a
 * context provider instead of a redux store because the client is not serializable.
 *
 * @param props - The props of the component.
 * @param props.children - The children of the component.
 */
export const ClusterProvider = ({ children }: ClusterProviderProps): JSX.Element => {
  const [state, setState] = useState<{ client: Synnax | null }>({ client: null });
  const dispatch = useDispatch();
  const activeCluster = useSelectCluster();
  const activeClusterKey = useSelectActiveClusterKey();

  useEffect(() => {
    if (activeCluster == null) return;

    const { key, props } = activeCluster;

    const client = new Synnax({
      ...props,
      connectivityPollFrequency: TimeSpan.seconds(5),
    });

    client.connectivity.onChange((status, _, message) =>
      dispatch(setClusterConnectionState({ key, state: { status, message } }))
    );

    setState({ client });

    return () => {
      client.close();
      setState({ client: null });
    };
  }, [activeClusterKey]);

  return <ClusterContext.Provider value={state}>{children}</ClusterContext.Provider>;
};
