import type { Connectivity } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/pluto";
import type { StatusVariant } from "@synnaxlabs/pluto";

import { ConnectionState } from "../types";

export interface ConnectionStatusProps {
  state: ConnectionState;
}

const connectionStatusVariants: Record<Connectivity, StatusVariant> = {
  connected: "success",
  failed: "error",
  connecting: "info",
  disconnected: "warning",
};

export const ConnectionStateBadge = ({
  state: { message, status },
}: ConnectionStatusProps): JSX.Element => (
  <Status.Text variant={connectionStatusVariants[status]}>{message}</Status.Text>
);
