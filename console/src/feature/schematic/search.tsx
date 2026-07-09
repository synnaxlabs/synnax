// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/pluto";

import { Layout } from "@/platform/layout";
import { Schematic } from "@/platform/schematic";
import { Search } from "@/platform/search";

const useOpen = () => {
  const placeLayout = Layout.usePlacer();
  return ({ id, name }: ontology.Resource) =>
    placeLayout(Schematic.create({ key: id.key, name }));
};

const SearchListItem = Search.createListItem({
  icon: <Icon.Schematic />,
  useOnSelect: useOpen,
});

export const SEARCH_LIST_ITEMS: Search.ListItems = { schematic: SearchListItem };
