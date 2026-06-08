// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, type panel, type Synnax } from "@synnaxlabs/client";
import { Panel } from "@synnaxlabs/pluto";

import { useSelectActivePanelKey, useSelectActiveTabKey } from "@/layout/selectors";

const firstTab = (node?: panel.Node | null): panel.Tab | null => {
  if (node == null) return null;
  if (node.leaf != null) return node.leaf.tabs[0] ?? null;
  if (node.split != null)
    return firstTab(node.split.first) ?? firstTab(node.split.last);
  return null;
};

const findTab = (
  node: panel.Node | undefined | null,
  key: string,
): panel.Tab | null => {
  if (node == null) return null;
  if (node.leaf != null) return node.leaf.tabs.find((t) => t.key === key) ?? null;
  if (node.split != null)
    return findTab(node.split.first, key) ?? findTab(node.split.last, key);
  return null;
};

// activeTab resolves the tab the user is currently working in: the explicitly
// selected tab when one is set, otherwise the panel's first tab (mirroring the
// mosaic adapter's default-to-first-tab behavior).
const activeTab = (
  root: panel.Node | undefined,
  selected: string | null,
): panel.Tab | null => {
  const tab = selected != null ? findTab(root, selected) : null;
  return tab ?? firstTab(root);
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
  return activeTab(data.root, selectedTab)?.resource ?? null;
};

// getActiveResource is the imperative form of useActiveResource for callers outside
// React render (e.g. ontology service handlers). It fetches the active panel from the
// server, so prefer useActiveResource where a hook is usable. Returns null when there
// is no active panel or the active tab is a view rather than a resource.
export const getActiveResource = async (
  client: Synnax,
  panelKey: panel.Key | null,
  selectedTab: string | null,
): Promise<ontology.ID | null> => {
  if (panelKey == null) return null;
  const p = await client.panels.retrieve(panelKey);
  return activeTab(p.root, selectedTab)?.resource ?? null;
};
