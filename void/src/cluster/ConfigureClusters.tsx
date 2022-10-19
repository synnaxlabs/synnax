import { Space, Header } from "@synnaxlabs/pluto";
import { AiFillDatabase, AiOutlinePlus } from "react-icons/ai";

function ConfigureClusters() {
  return (
    <Space>
      <Header
        level="p"
        text="Clusters"
        icon={<AiFillDatabase />}
        actions={[
          {
            icon: <AiOutlinePlus />,
          },
        ]}
      />
    </Space>
  );
}

const ClustersItem = {
  key: "clusters",
  icon: <AiFillDatabase />,
  content: <ConfigureClusters />,
  minSize: 150,
  maxSize: 500,
  initialSize: 250,
};

export default ClustersItem;
