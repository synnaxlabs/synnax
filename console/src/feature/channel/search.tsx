// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot, type ontology, panel } from "@synnaxlabs/client";
import { Icon, Status, Synnax } from "@synnaxlabs/pluto";

import { LinePlot } from "@/platform/lineplot";
import { Panel } from "@/platform/panel";
import { Search } from "@/platform/search";
import { Session } from "@/session";

const useOpen = () => {
  const client = Synnax.use();
  const openTab = Panel.useOpenTab();
  const getFocusedTab = Session.Panel.useGetFocusedTab();
  const getSelectedPanel = Session.Panel.useGetSelected();
  const getSelectedProject = Session.Project.useGetSelected();
  const getSelectedRange = Session.Range.useGetSelectedKey();
  const store = Session.useStore();
  const handleError = Status.useErrorHandler();
  return (resource: ontology.Resource) => {
    if (client == null) return;
    const nonVirtualSelection = [resource]
      .filter((s) => s.data?.virtual !== true || s.data.expression != "")
      .map((s) => Number(s.id.key));
    if (nonVirtualSelection.length === 0) return;
    handleError(async () => {
      const focusedTab = getFocusedTab();
      const panelKey = getSelectedPanel();
      if (focusedTab != null && panelKey != null) {
        const doc = await client.panels.retrieve(panelKey);
        const tab = panel.findTab(doc.root, focusedTab);
        if (tab?.variant === "resource" && tab.resource.type === "lineplot") {
          await LinePlot.addChannelsToActivePlot(
            client,
            tab.resource.key,
            nonVirtualSelection,
          );
          return;
        }
      }
      const project = getSelectedProject();
      const selectedRange = getSelectedRange() ?? Session.Range.RECENT_KEY;
      const { key } = await client.lineplots.create(project, {
        name: "Line Plot",
        channels: { y1: nonVirtualSelection },
        ranges: { x1: [selectedRange] },
      });
      store.dispatch(Session.LinePlot.create({ key }));
      openTab({ variant: "resource", resource: lineplot.ontologyID(key) });
    }, "Failed to add channels to plot");
  };
};

export const SearchListItem = Search.createListItem({
  icon: <Icon.Channel />,
  useOnSelect: useOpen,
});

export const SEARCH_LIST_ITEMS: Search.ListItems = { channel: SearchListItem };
