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

import { Layout } from "@/platform/layout";
import { LinePlot } from "@/platform/lineplot";
import { Search } from "@/platform/search";
import { Session } from "@/session";

const useOpen = () => {
  const client = Synnax.use();
  const store = Session.useStore();
  const placeLayout = Layout.usePlacer();
  const handleError = Status.useErrorHandler();
  return (resource: ontology.Resource) => {
    if (client == null) return;
    const state = store.getState();
    const layout = Session.Layout.selectActiveMosaicLayout(state);
    const nonVirtualSelection = [resource]
      .filter((s) => s.data?.virtual !== true || s.data.expression != "")
      .map((s) => Number(s.id.key));
    if (nonVirtualSelection.length === 0) return;
    switch (layout?.type) {
      case LinePlot.LAYOUT_TYPE: {
        handleError(
          () =>
            LinePlot.addChannelsToActivePlot(client, layout.key, nonVirtualSelection),
          "Failed to add channels to plot",
        );
        break;
      }
      default: {
        const project = Session.Project.selectSelected(state);
        const activeRange =
          Session.Range.selectSelectedKey(state) ?? Session.Range.RECENT_KEY;
        handleError(async () => {
          const { key, name } = await client.lineplots.create(project, {
            name: "Line Plot",
            channels: { y1: nonVirtualSelection },
            ranges: { x1: [activeRange] },
          });
          placeLayout(LinePlot.create({ key, name }));
        }, "Failed to create plot");
      }
    }
  };
};

export const SearchListItem = Search.createListItem({
  icon: <Icon.Channel />,
  useOnSelect: useOpen,
});

export const SEARCH_LIST_ITEMS: Search.ListItems = { channel: SearchListItem };
