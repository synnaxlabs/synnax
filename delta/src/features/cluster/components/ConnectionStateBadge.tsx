import { Status, StatusVariant } from "@synnaxlabs/pluto";
import { Connectivity } from "@synnaxlabs/client";
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
}: ConnectionStatusProps) => (
  <Status.Text variant={connectionStatusVariants[status]}>
    {message}
  </Status.Text>
);
