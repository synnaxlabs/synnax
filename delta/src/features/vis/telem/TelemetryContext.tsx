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

import { useClusterClient } from "@/features/cluster";

export interface TelemetryContextValue {
  client: TelemetryClient | null;
}

const Context = createContext<TelemetryContextValue | null>(null);

export interface TelemetryProviderProps extends PropsWithChildren { }

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
