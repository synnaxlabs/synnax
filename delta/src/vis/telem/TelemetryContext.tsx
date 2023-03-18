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

import { FrameCache } from "@synnaxlabs/client";
import { GLDemandCache, useGLContext } from "@synnaxlabs/pluto";

import { TelemetryClient } from "./client";

import { useClusterClient } from "@/cluster";

export interface TelemetryContextValue {
  client: TelemetryClient | null;
}

const Context = createContext<TelemetryContextValue | null>(null);

export interface TelemetryProviderProps extends PropsWithChildren {}

export const useTelemetryClient = (): TelemetryClient | null =>
  useContext(Context)?.client ?? null;

export const TelemetryProvider = ({
  children,
}: TelemetryProviderProps): JSX.Element => {
  const glCtx = useGLContext();
  const clusterClient = useClusterClient();
  const [client, setClient] = useState<TelemetryClient | null>(null);

  useEffect(() => {
    if (clusterClient == null || glCtx == null) return;
    setClient(
      new TelemetryClient(new GLDemandCache(glCtx.gl), clusterClient, new FrameCache())
    );
  }, [clusterClient, glCtx]);

  return <Context.Provider value={{ client }}>{children}</Context.Provider>;
};
