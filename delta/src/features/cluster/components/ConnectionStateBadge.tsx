import { Connectivity } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/pluto";
import type { StatusVariant } from "@synnaxlabs/pluto";

import { ConnectionState } from "../types";

export interface ConnectionStatusProps {
  state: ConnectionState;
}

const connectionStatusVariants: Record<Connectivity, StatusVariant> = {
  [Connectivity.Connected]: "success",
  [Connectivity.Failed]: "error",
  [Connectivity.Connecting]: "info",
  [Connectivity.Disconnected]: "warning",
};

export const ConnectionStateBadge = ({
  state: { message, status },
}: ConnectionStatusProps): JSX.Element => (
  <Status.Text variant={connectionStatusVariants[status]}>{message}</Status.Text>
);
