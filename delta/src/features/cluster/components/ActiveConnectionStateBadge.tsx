import { useSelectActiveCluster } from "../store";
import { DEFAULT_CONNECTION_STATE } from "../types";
import { ConnectionStateBadge } from "./ConnectionStateBadge";

export const ActiveConnectionBadge = () => {
  const cluster = useSelectActiveCluster();
  const connState = cluster?.state || DEFAULT_CONNECTION_STATE;
  return <ConnectionStateBadge state={connState} />;
};
