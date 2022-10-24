import { Connectivity } from "@synnaxlabs/client";
import { Nav, Text, Divider } from "@synnaxlabs/pluto";
import { AiFillDatabase } from "react-icons/ai";
import ConnectionStatus from "../../cluster/ConnectionStatus";
import { useSelectActiveCluster } from "../../cluster/slice";

export default function BottomNavbar() {
  const activeCluster = useSelectActiveCluster();
  const activeClusterState = activeCluster?.state || {
    status: Connectivity.DISCNNECTED,
    message: "Disconnected",
  };

  return (
    <Nav.Bar location="bottom" size={32}>
      <Nav.Bar.End style={{ padding: "0 12px" }}>
        {activeCluster && (
          <>
            <Divider direction="vertical" />
            <Text.WithIcon level="p" startIcon={<AiFillDatabase />}>
              {activeCluster.name}
            </Text.WithIcon>
          </>
        )}
        <Divider direction="vertical" />
        <ConnectionStatus state={activeClusterState} />
      </Nav.Bar.End>
    </Nav.Bar>
  );
}
