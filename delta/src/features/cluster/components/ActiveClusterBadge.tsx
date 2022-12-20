import { Text } from "@synnaxlabs/pluto";

import { useSelectActiveCluster } from "../store";

import { ClusterIcon } from "./Icon";

export const ActiveClusterBadge = (): JSX.Element => {
  const cluster = useSelectActiveCluster();
  return (
    <Text.WithIcon level="p" startIcon={<ClusterIcon />}>
      {cluster != null ? cluster.name : "No Active Cluster"}
    </Text.WithIcon>
  );
};
