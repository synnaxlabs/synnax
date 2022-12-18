import type { Connectivity, SynnaxProps } from "@synnaxlabs/client";

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
  status: "disconnected",
  message: "Disconnected",
};
