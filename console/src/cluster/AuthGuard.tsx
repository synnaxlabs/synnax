// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useEffect, useState } from "react";
import { useDispatch } from "react-redux";

import { detectServingCluster } from "@/cluster/autoConnect";
import { LoginScreen } from "@/cluster/LoginScreen";
import { useSelect } from "@/cluster/selectors";
import { type Cluster, set, setActive } from "@/cluster/slice";

interface AuthGuardProps {
  children: ReactElement;
}

export const AuthGuard = ({ children }: AuthGuardProps): ReactElement => {
  const dispatch = useDispatch();
  const activeCluster = useSelect();
  const [servingCluster, setServingCluster] = useState<Cluster | null>(null);
  const [isAuthenticating, setIsAuthenticating] = useState(true);

  useEffect(() => {
    // Check if we're being served by a cluster
    const serving = detectServingCluster();
    if (serving != null) {
      setServingCluster(serving);

      // Check if we already have credentials for this cluster
      if (
        activeCluster != null &&
        activeCluster.host === serving.host &&
        activeCluster.port === serving.port &&
        activeCluster.username != null &&
        activeCluster.password != null
      )
        // We already have credentials, no need to show login
        setIsAuthenticating(false);
      else
        // Need to authenticate
        setIsAuthenticating(true);
    } else
      // Not being served by a cluster, normal console mode
      setIsAuthenticating(false);
  }, [activeCluster]);

  const handleLoginSuccess = (credentials: {
    username: string;
    password: string;
  }): void => {
    if (servingCluster == null) return;
    const clusterKey = `${servingCluster.host}:${servingCluster.port}`;
    const cluster = {
      key: clusterKey,
      name: `${servingCluster.host}:${servingCluster.port}`,
      host: servingCluster.host,
      port: servingCluster.port,
      username: credentials.username,
      password: credentials.password,
      secure: servingCluster.secure,
    };
    dispatch(set(cluster));
    dispatch(setActive(clusterKey));
    setIsAuthenticating(false);
  };

  // If we're being served by a cluster and not authenticated, show login
  if (servingCluster != null && isAuthenticating)
    return (
      <LoginScreen
        host={servingCluster.host}
        port={servingCluster.port}
        secure={servingCluster.secure}
        onSuccess={handleLoginSuccess}
      />
    );

  // Otherwise, show the normal console
  return children;
};
