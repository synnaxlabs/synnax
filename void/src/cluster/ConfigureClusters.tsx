import { createWindow } from "@synnaxlabs/drift";
import { Space, Header } from "@synnaxlabs/pluto";
import { AiFillDatabase, AiOutlinePlus } from "react-icons/ai";
import { useDispatch } from "react-redux";
import { createConnectClusterWindow } from "./ConnectCluster";

function ConfigureClusters() {
  const dispatch = useDispatch();
  return (
    <Space>
      <Header
        level="p"
        divided
        icon={<AiFillDatabase />}
        actions={[
          {
            children: <AiOutlinePlus />,
            onClick: () => createConnectClusterWindow(dispatch),
          },
        ]}
      >
        Clusters
      </Header>
    </Space>
  );
}

const ClustersItem = {
  key: "clusters",
  content: <ConfigureClusters />,
  icon: <AiFillDatabase />,
  minSize: 150,
  maxSize: 500,
  initialSize: 250,
};

export default ClustersItem;
