import { Connectivity } from "@synnaxlabs/client";
import { Nav } from "@synnaxlabs/pluto";
import ConnectionStatus from "../../cluster/ConnectionStatus";
import { useSelectActiveCluster } from "../../cluster/slice";

export default function BottomNavbar() {
  const activeCluster = useSelectActiveCluster();
  const activeClusterState = activeCluster?.state || {
    status: Connectivity.DISCNNECTED,
    message: "No Active Cluster",
  };

  return (
    <Nav.Bar location="bottom" size={36}>
      <Nav.Bar.End style={{ padding: "0 12px" }}>
        <ConnectionStatus state={activeClusterState} />
      </Nav.Bar.End>
    </Nav.Bar>
  );
}
