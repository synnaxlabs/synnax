import { Connectivity } from "@synnaxlabs/client";
import type { SynnaxProps } from "@synnaxlabs/client";

export interface Cluster {
  key: string;
  name: string;
  props: SynnaxProps;
  state: ConnectionState;
}

export interface ConnectionState {
  status: Connectivity;
  message?: string;
}

export const DEFAULT_CONNECTION_STATE: ConnectionState = {
  status: Connectivity.Disconnected,
  message: "Disconnected",
};
