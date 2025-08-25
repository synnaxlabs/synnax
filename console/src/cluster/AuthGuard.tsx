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

import { detectServingConnection } from "@/cluster/autoConnect";
import { LoginScreen } from "@/cluster/LoginScreen";
import { useSelect } from "@/cluster/selectors";
import { set, setActive } from "@/cluster/slice";
import { Cluster } from "@/cluster/types";
import z from "zod";

interface AuthGuardProps {
  children: ReactElement;
}

export const AuthGuard = ({ children }: AuthGuardProps): ReactElement | null => {
  const dispatch = useDispatch();
  const activeCluster = useSelect();
  const serving = detectServingConnection();
  const initialAuthState = serving != null && activeCluster == null;
  const handleConnect = (cluster: Cluster): void => {
    dispatch(set(cluster));
    dispatch(setActive(cluster.key));
  };
  if (serving != null && initialAuthState)
    return <LoginScreen connection={serving} onSuccess={handleConnect} />;
  return children;
};
