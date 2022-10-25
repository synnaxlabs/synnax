import {
  AiFillDatabase,
  AiFillInfoCircle,
  AiFillWarning,
  AiOutlineCheck,
  AiOutlineClose,
  AiOutlineWarning,
} from "react-icons/ai";
import { Text, TextProps, TypographyLevel } from "@synnaxlabs/pluto";
import { ReactElement } from "react";
import { Connectivity } from "@synnaxlabs/client";
import { ConnectionState, DEFAULT_CONNECTION_STATE } from "../../types";
import { useSelectActiveCluster } from "../../store";

export interface ConnectionStatusProps {
  state: ConnectionState;
}

const connectionStatusVariants: Record<Connectivity, StatusVariant> = {
  [Connectivity.CONNECTED]: "success",
  [Connectivity.FAILED]: "error",
  [Connectivity.CONNECTING]: "info",
  [Connectivity.DISCNNECTED]: "warning",
};

export const ActiveClusterBadge = () => {
  const cluster = useSelectActiveCluster();
  return (
    <Text.WithIcon level="p" startIcon={<AiFillDatabase />}>
      {cluster ? cluster.name : "No Active Cluster"}
    </Text.WithIcon>
  );
};

export const ActiveConnectionBadge = () => {
  const cluster = useSelectActiveCluster();
  const connState = cluster?.state || DEFAULT_CONNECTION_STATE;
  return <ConnectionBadge state={connState} />;
};

export const ConnectionBadge = ({
  state: { message, status },
}: ConnectionStatusProps) => (
  <StatusBadge variant={connectionStatusVariants[status]} message={message} />
);
