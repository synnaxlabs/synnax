// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot, log, type ontology, panel } from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { color } from "@synnaxlabs/x";
import { useCallback } from "react";

import { LinePlot } from "@/platform/lineplot";
import { Log } from "@/platform/log";
import { Panel } from "@/platform/panel";
import { Session } from "@/session";

// A virtual channel without an expression has no stored data to plot, so it
// routes to a log, which displays its live samples. Calculated channels plot
// like persisted ones.
const isPlainVirtual = ({ data }: ontology.Resource): boolean =>
  data?.virtual === true && data.expression == "";

// useOpen returns a handler that opens a channel resource in a visualization:
// plottable channels go to the focused line plot or a new one, plain virtual
// channels to the focused log or a new one.
export const useOpen = (): ((resource: ontology.Resource) => void) => {
  const client = Synnax.use();
  const openTab = Panel.useOpenTab();
  const getFocusedTab = Session.Panel.useGetFocusedTab();
  const getSelectedPanel = Session.Panel.useGetSelected();
  const getSelectedProject = Session.Project.useGetSelected();
  const getSelectedRange = Session.Range.useGetSelectedKey();
  const store = Session.useStore();
  const handleError = Status.useErrorHandler();
  return useCallback(
    (resource) => {
      if (client == null) return;
      const channelKey = Number(resource.id.key);
      const virtual = isPlainVirtual(resource);
      handleError(
        async () => {
          const focusedTab = getFocusedTab();
          const panelKey = getSelectedPanel();
          if (focusedTab != null && panelKey != null) {
            const doc = await client.panels.retrieve(panelKey);
            const tab = panel.findTab(doc.root, focusedTab);
            if (tab?.variant === "resource")
              if (virtual && tab.resource.type === "log") {
                await Log.addChannelsToActiveLog(client, tab.resource.key, [
                  channelKey,
                ]);
                return;
              } else if (!virtual && tab.resource.type === "lineplot") {
                await LinePlot.addChannelsToActivePlot(client, tab.resource.key, [
                  channelKey,
                ]);
                return;
              }
          }
          const project = getSelectedProject();
          if (virtual) {
            const { key } = await client.logs.create(project, {
              name: "Log",
              channels: [{ channel: channelKey, color: color.ZERO }],
            });
            store.dispatch(Session.Log.create({ key }));
            openTab({ variant: "resource", resource: log.ontologyID(key) });
            return;
          }
          const selectedRange = getSelectedRange() ?? Session.Range.RECENT_KEY;
          const { key } = await client.lineplots.create(project, {
            name: "Line Plot",
            channels: { y1: [channelKey] },
            ranges: { x1: [selectedRange] },
          });
          store.dispatch(Session.LinePlot.create({ key }));
          openTab({ variant: "resource", resource: lineplot.ontologyID(key) });
        },
        virtual ? "Failed to add channel to log" : "Failed to add channel to plot",
      );
    },
    [
      client,
      openTab,
      getFocusedTab,
      getSelectedPanel,
      getSelectedProject,
      getSelectedRange,
      store,
      handleError,
    ],
  );
};
