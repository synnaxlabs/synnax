// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { Icon, Status, Synnax } from "@synnaxlabs/pluto";

import { retrieveAndPlaceLayout } from "@/feature/task/layouts";
import { Layout } from "@/platform/layout";
import { Search } from "@/platform/search";

const useOpen = () => {
  const client = Synnax.use();
  const placeLayout = Layout.usePlacer();
  const handleError = Status.useErrorHandler();
  return ({ id, name }: ontology.Resource) => {
    if (client == null) return;
    handleError(
      async () => await retrieveAndPlaceLayout(client, id.key, placeLayout),
      `Could not open ${name}`,
    );
  };
};

const SearchListItem = Search.createListItem({
  icon: <Icon.Task />,
  useOnSelect: useOpen,
});

export const SEARCH_LIST_ITEMS: Search.ListItems = { task: SearchListItem };
