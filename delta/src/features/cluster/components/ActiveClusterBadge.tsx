import { AiFillDatabase } from "react-icons/ai";
import { Text } from "@synnaxlabs/pluto";
import { useSelectActiveCluster } from "../store";

export const ActiveClusterBadge = () => {
  const cluster = useSelectActiveCluster();
  return (
    <Text.WithIcon level="p" startIcon={<AiFillDatabase />}>
      {cluster ? cluster.name : "No Active Cluster"}
    </Text.WithIcon>
  );
};
