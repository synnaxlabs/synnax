import { Connectivity, SynnaxProps } from "@synnaxlabs/client";

export type Cluster = {
  key: string;
  name: string;
  props: SynnaxProps;
  active: boolean;
  state: ConnectionState;
};

export type ConnectionState = {
  status: Connectivity;
  message?: string;
};

export const DEFAULT_CONNECTION_STATE: ConnectionState = {
  status: Connectivity.DISCNNECTED,
  message: "Disconnected",
};
