import { Text } from "@synnaxlabs/pluto";
import { AiFillDatabase } from "react-icons/ai";

import { useSelectActiveCluster } from "../store";

export const ActiveClusterBadge = (): JSX.Element => {
  const cluster = useSelectActiveCluster();
  return (
    <Text.WithIcon level="p" startIcon={<AiFillDatabase />}>
      {cluster != null ? cluster.name : "No Active Cluster"}
    </Text.WithIcon>
  );
};
