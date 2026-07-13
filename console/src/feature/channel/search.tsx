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
  return (entry: Tree.Entry) => {
    if (client == null) return;
    handleError(async () => {
      const channelKey = Number(entry.id.key);
      const ch = await client.channels.retrieve(channelKey);
      if (ch.virtual && ch.expression == "") return;
      const state = store.getState();
      const layout = Session.Layout.selectActiveMosaicLayout(state);
      if (layout?.type === LinePlot.LAYOUT_TYPE) {
        await LinePlot.addChannelsToActivePlot(client, layout.key, [channelKey]);
        return;
      }
      const project = Session.Project.selectSelected(state);
      const activeRange =
        Session.Range.selectSelectedKey(state) ?? Session.Range.RECENT_KEY;
      const { key, name } = await client.lineplots.create(project, {
        name: "Line Plot",
        channels: { y1: [channelKey] },
        ranges: { x1: [activeRange] },
      });
      placeLayout(LinePlot.create({ key, name }));
    }, "Failed to plot channel");
  };
};

const SearchListItem = Search.createListItem({
  icon: <Icon.Channel />,
  useOnSelect: useOpen,
});

export const SEARCH_LIST_ITEMS: Search.ListItems = { channel: SearchListItem };
