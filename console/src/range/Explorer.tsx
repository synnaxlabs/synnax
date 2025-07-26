import { Ranger } from "@synnaxlabs/pluto";

import { type Layout } from "@/layout";
import { List } from "@/range/list/List";

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
  const { data, getItem, subscribe, retrieve } = Ranger.useList({});
  return (
    <List
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      retrieve={retrieve}
      enableSearch
    />
  );
};
