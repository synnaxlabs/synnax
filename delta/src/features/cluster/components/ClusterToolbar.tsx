import { Space, Header, List, Text } from "@synnaxlabs/pluto";
import type { ListItemProps } from "@synnaxlabs/pluto";
import clsx from "clsx";
import { AiOutlinePlus } from "react-icons/ai";
import { useDispatch } from "react-redux";

import { useSelectCluster, useSelectClusters } from "../store";
import { setActiveCluster } from "../store/slice";
import { Cluster } from "../types";

import { ClusterIcon } from "./ClusterIcon";

import { Layout, useLayoutPlacer } from "@/features/layout";

import "./ClusterToolbar.css";

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
  const data = Object.values(useSelectClusters());
  const active = useSelectCluster();
  const openWindow = useLayoutPlacer();

  const selected = active != null ? [active?.key] : [];

  const actions = [
    {
      children: <AiOutlinePlus />,
      onClick: () => openWindow(connectClusterWindowLayout),
    },
  ];

  const handleSelect = ([key]: string[]): void => {
    dispatch(setActiveCluster(key ?? null));
  };

  return (
    <Space empty>
      <Header level="h4" divided icon={<ClusterIcon />} actions={actions}>
        Clusters
      </Header>
      <List<Omit<Cluster, "state" | "props">>
        selectMultiple={false}
        data={data}
        selected={selected}
        onSelect={handleSelect}
      >
        <List.Core.Virtual itemHeight={30}>{ListItem}</List.Core.Virtual>
      </List>
    </Space>
  );
};

const ListItem = ({
  entry: { key, name },
  style,
  selected,
  onSelect,
  ...props
}: ListItemProps<Omit<Cluster, "props" | "state">>): JSX.Element => (
  <Space
    direction="horizontal"
    align="center"
    justify="spaceBetween"
    onDoubleClick={() => onSelect(key)}
    className={clsx(
      "delta-cluster-toolbar-list__item",
      selected && "delta-cluster-toolbar-list__item--selected"
    )}
    {...props}
  >
    <Text level="p">{name}</Text>
  </Space>
);

export const ClusterToolbar = {
  key: "clusters",
  content: <Content />,
  icon: <ClusterIcon />,
  minSize: 150,
  maxSize: 500,
  initialSize: 250,
};
