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

import { Search } from "@/platform/search";
import { Session } from "@/session";

const useOpen = () => {
  const client = Synnax.use();
  const store = Session.useStore();
  const handleError = Status.useErrorHandler();
  return ({ id, name }: Tree.Entry) => {
    if (client == null) return;
    handleError(async () => {
      const proj = await client.projects.retrieve(id.key);
      store.dispatch(Session.Project.select(proj.key));
      store.dispatch(
        Session.Layout.setProject({
          slice: Session.Layout.migrateLayout(proj.layout),
        }),
      );
    }, `Failed to select ${name}`);
  };
};

const SearchListItem = Search.createListItem({
  icon: <Icon.Project />,
  useOnSelect: useOpen,
});

export const SEARCH_LIST_ITEMS: Search.ListItems = { project: SearchListItem };
