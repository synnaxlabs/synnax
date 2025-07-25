import { type ranger } from "@synnaxlabs/client";
import { Component, List, Ranger } from "@synnaxlabs/pluto";

import { type Layout } from "@/layout";

export interface ListItemProps extends List.ItemProps<ranger.Key> {}

export const ListItem = (props: ListItemProps) => <Ranger.ListItem {...props} />;

const listItem = Component.renderProp(ListItem);

export const EXPLORER_LAYOUT_TYPE = "range_explorer";

export const EXPLORER_LAYOUT: Layout.State = {
  key: EXPLORER_LAYOUT_TYPE,
  windowKey: EXPLORER_LAYOUT_TYPE,
  type: EXPLORER_LAYOUT_TYPE,
  name: "Range Explorer",
  icon: "Explore",
  location: "mosaic",
};

export const Explorer: Layout.Renderer = () => {
  const { data, getItem, subscribe, retrieve } = Ranger.useList();
  const { fetchMore } = List.usePager({ retrieve });
  return (
    <List.Frame<ranger.Key, ranger.Range>
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      onFetchMore={fetchMore}
    >
      <List.Items>{listItem}</List.Items>
    </List.Frame>
  );
};
