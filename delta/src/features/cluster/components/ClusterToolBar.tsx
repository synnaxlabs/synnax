import { Space, Header, List, Text } from "@synnaxlabs/pluto";
import type { ListItemProps, RenderableRecord } from "@synnaxlabs/pluto";
import { AiOutlinePlus } from "react-icons/ai";
import { useDispatch } from "react-redux";

import { useSelectActiveCluster, useSelectClusters } from "../store";
import { setActiveCluster } from "../store/slice";
import { Cluster } from "../types";

import { ClusterIcon } from "./Icon";

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
        icon={<ClusterIcon />}
        actions={[
          {
            children: <AiOutlinePlus />,
            onClick: () => openWindow(connectClusterWindowLayout),
          },
        ]}
      >
        Clusters
      </Header>
      <List<Omit<Cluster, "state" | "props">>
        selectMultiple={false}
        data={Object.values(clusters)}
        selected={active != null ? [active?.key] : []}
        onSelect={([key]) => dispatch(setActiveCluster(key))}
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
  icon: <ClusterIcon />,
  minSize: 150,
  maxSize: 500,
  initialSize: 250,
};
