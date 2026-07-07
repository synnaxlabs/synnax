import { type ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/pluto";

import { Arc } from "@/platform/arc";
import { Layout } from "@/platform/layout";
import { Search } from "@/platform/search";

const useOpen = () => {
  const placeLayout = Layout.usePlacer();
  return ({ key, name }: ontology.Resource) => placeLayout(Arc.create({ key, name }));
};

export const SearchListItem = Search.createListItem({
  icon: <Icon.Arc />,
  useOnSelect: useOpen,
});

export const SEARCH_LIST_ITEMS: Search.ListItems = { arc: SearchListItem };
