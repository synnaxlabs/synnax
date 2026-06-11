// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, panel } from "@synnaxlabs/client";
import { Panel, type Pluto } from "@synnaxlabs/pluto";

import { useSelectActivePanelKey, useSelectActiveTabKey } from "@/layout/selectors";

// activeTab resolves the tab the user is currently working in: the explicitly
// selected tab when one is set, otherwise the panel's first tab (mirroring the
// mosaic adapter's default-to-first-tab behavior).
const activeTab = (
  root: panel.Node | undefined,
  selected: string | null,
): panel.Tab | null => {
  const tab = selected != null ? panel.findTab(root, selected) : null;
  return tab ?? panel.firstTab(root);
};

// useActiveResource resolves the ontology resource displayed by the active tab of
// the active panel, or null when there is no active panel, no tabs, or the active
// tab is an inline view rather than a resource. This bridges the session-level
// active-tab cursor (Redux) to the server-backed panel tree (Flux), giving callers
// the identity of "what the user is currently looking at" — the successor to the
// old active-mosaic-tab selectors.
export const useActiveResource = (): ontology.ID | null => {
  const panelKey = useSelectActivePanelKey();
  const selectedTab = useSelectActiveTabKey();
  const { data } = Panel.useRetrieve({ key: panelKey ?? "" });
  if (panelKey == null || data == null) return null;
  const tab = activeTab(data.root, selectedTab);
  return tab?.variant === "resource" ? tab.resource : null;
};

// getActiveResource is the imperative form of useActiveResource for callers outside
// React render (e.g. ontology service handlers). It reads the active panel from the
// Flux cache; prefer useActiveResource where a hook is usable. Returns null when
// there is no active panel (or it has not loaded) or the active tab is a view
// rather than a resource.
export const getActiveResource = (
  fluxStore: Pluto.FluxStore,
  panelKey: panel.Key | null,
  selectedTab: string | null,
): ontology.ID | null => {
  if (panelKey == null) return null;
  const p = fluxStore.panels.get(panelKey);
  if (p == null) return null;
  const tab = activeTab(p.root, selectedTab);
  return tab?.variant === "resource" ? tab.resource : null;
};
