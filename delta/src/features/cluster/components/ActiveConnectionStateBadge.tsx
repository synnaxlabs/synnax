import { useSelectActiveCluster } from "../store";
import { DEFAULT_CONNECTION_STATE } from "../types";

import { ConnectionStateBadge } from "./ConnectionStateBadge";

export const ActiveConnectionBadge = (): JSX.Element => {
  const cluster = useSelectActiveCluster();
  return <ConnectionStateBadge state={cluster?.state ?? DEFAULT_CONNECTION_STATE} />;
};
