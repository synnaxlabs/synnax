import { createWindow } from "@synnaxlabs/drift";
import { Space, Header, List, Text, ListItemProps } from "@synnaxlabs/pluto";
import { AiFillDatabase, AiOutlinePlus } from "react-icons/ai";
import { useDispatch } from "react-redux";
import { createConnectClusterWindow } from "./ConnectCluster";
import { useSelector } from "react-redux";
import { Cluster, setActiveCluster } from "./slice";

function ConfigureClusters() {
  const dispatch = useDispatch();
  const clusters = useSelector((state: any) => state.cluster.clusters);
  return (
    <Space empty>
      <Header
        level="h4"
        divided
        icon={<AiFillDatabase />}
        actions={[
          {
            children: <AiOutlinePlus />,
            onClick: () => dispatch(createConnectClusterWindow()),
          },
        ]}
      >
        Clusters
      </Header>
      <List
        selectMultiple={false}
        data={clusters}
        selected={[clusters.find((c: Cluster) => c.active)?.key]}
        onSelect={(key: string[]) => {
          if (key.length > 0) {
            dispatch(setActiveCluster(key[0]));
          } else {
            dispatch(setActiveCluster(undefined));
          }
        }}
      >
        <List.Core.Virtual itemHeight={30}>{ClusterListItem}</List.Core.Virtual>
      </List>
    </Space>
  );
}

interface ClusterListItemProps extends ListItemProps<string, Cluster> {}

const ClusterListItem = ({
  entry,
  style,
  selected,
  onSelect,
  ...props
}: ClusterListItemProps) => {
  return (
    <Space
      direction="horizontal"
      align="center"
      justify="spaceBetween"
      onDoubleClick={(e) => {
        e.preventDefault();
        onSelect(entry.key);
      }}
      style={{
        padding: "0 0 0 10px",
        width: "100%",
        height: 30,
        cursor: "pointer",
        userSelect: "none",
        backgroundColor: selected && "var(--pluto-primary-p1-20)",
        borderLeft: selected && "3px solid var(--pluto-primary-p1)",
        ...style,
      }}
      {...props}
    >
      <Text level="p">{entry.name}</Text>
    </Space>
  );
};

const ClustersItem = {
  key: "clusters",
  content: <ConfigureClusters />,
  icon: <AiFillDatabase />,
  minSize: 150,
  maxSize: 500,
  initialSize: 250,
};

export default ClustersItem;
