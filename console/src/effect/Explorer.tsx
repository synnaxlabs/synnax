import { Effect } from "@synnaxlabs/pluto";

import { List } from "@/effect/list/List";
import { type Layout } from "@/layout";

export const EXPLORER_LAYOUT_TYPE = "effect_explorer";

export const EXPLORER_LAYOUT: Layout.State = {
  key: EXPLORER_LAYOUT_TYPE,
  windowKey: EXPLORER_LAYOUT_TYPE,
  type: EXPLORER_LAYOUT_TYPE,
  name: "Effect Explorer",
  icon: "Explore",
  location: "mosaic",
};

export const Explorer: Layout.Renderer = () => {
  const { data, getItem, subscribe, retrieve } = Effect.useList({});
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
