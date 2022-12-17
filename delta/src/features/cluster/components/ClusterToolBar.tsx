import { Space, Header, List, Text } from "@synnaxlabs/pluto";
import type { ListItemProps, ListEntry } from "@synnaxlabs/pluto";
import { AiFillDatabase, AiOutlinePlus } from "react-icons/ai";
import { useDispatch } from "react-redux";

import { useSelectActiveCluster, useSelectClusters } from "../store";
import { setActiveCluster } from "../store/slice";
import { Cluster } from "../types";

import { Layout, useLayoutPlacer } from "@/features/layout";

const connectClusterWindowLayout: Layout = {
  key: "connectCluster",
  type: "connectCluster",
  title: "Connect a Cluster",
  location: "window",
  window: {
    resizable: false,
    height: 430,
    width: 650,
    navTop: true,
  },
};

const Content = (): JSX.Element => {
  const dispatch = useDispatch();
  const clusters = useSelectClusters();
  const active = useSelectActiveCluster();
  const openWindow = useLayoutPlacer();

  return (
    <Space empty>
      <Header
        level="h4"
        divided
        icon={<AiFillDatabase />}
        actions={[
          {
            children: <AiOutlinePlus />,
            onClick: () => openWindow(connectClusterWindowLayout),
          },
        ]}
      >
        Clusters
      </Header>
      <List
        selectMultiple={false}
        data={Object.values(clusters) as unknown as ListEntry[]}
        selected={active != null ? [active?.key] : []}
        onSelect={(key: string[]) => dispatch(setActiveCluster(key[0]))}
      >
        <List.Core.Virtual itemHeight={30}>{ListItem}</List.Core.Virtual>
      </List>
    </Space>
  );
};

const ListItem = ({
  entry,
  style,
  selected,
  onSelect,
  ...props
}: ListItemProps<Omit<Cluster, "props" | "state">>): JSX.Element => {
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
        backgroundColor: selected ? "var(--pluto-primary-p1-20)" : "",
        borderLeft: selected ? "3px solid var(--pluto-primary-p1)" : "",
      }}
      {...props}
    >
      <Text level="p">{entry.name}</Text>
    </Space>
  );
};

export const ClusterToolBar = {
  key: "clusters",
  content: <Content />,
  icon: <AiFillDatabase />,
  minSize: 150,
  maxSize: 500,
  initialSize: 250,
};
