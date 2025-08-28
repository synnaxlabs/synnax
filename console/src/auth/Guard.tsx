// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PropsWithChildren, type ReactNode } from "react";
import { useDispatch } from "react-redux";

import { Login } from "@/auth/Login";
import { Cluster } from "@/cluster";

interface GuardProps extends PropsWithChildren {}

export const Guard = ({ children }: GuardProps): ReactNode => {
  const dispatch = useDispatch();
  const activeCluster = Cluster.useSelect();
  const serving = Cluster.detectConnection();
  const initialAuthState = serving != null && activeCluster == null;
  const handleConnect = (cluster: Cluster.Cluster): void => {
    dispatch(Cluster.set(cluster));
    dispatch(Cluster.setActive(cluster.key));
  };
  if (serving != null && initialAuthState)
    return <Login connection={serving} onSuccess={handleConnect} />;
  return children;
};
