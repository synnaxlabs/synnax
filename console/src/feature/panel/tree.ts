// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel, query, type resource, type Synnax as Client } from "@synnaxlabs/client";
import { type Icon } from "@synnaxlabs/lyra/icon";
import { type destructor, type record } from "@synnaxlabs/x";

import { renderIcon } from "@/feature/panel/icon";
import { type Group } from "@/platform/group";
import { Tree } from "@/platform/tree";

const appearanceZ = panel.panelZ.pick({ icon: true, color: true });

// The panel's tree resource carries the panel, so the tree item needs no retrieve.
const icon = (resource: Tree.Resource): Icon.ReactElement =>
  renderIcon(appearanceZ.parse(resource.data ?? {}));

/** Lists the panels that sit in a group. */
export const GROUP_MEMBER: Group.Member = {
  type: "panel",
  children: (client, key) =>
    Tree.createSource(client.panels, { groups: [key] }, Tree.toResource("panel")),
};

const tabResources = (node: panel.Node): resource.ID[] =>
  node.variant === "leaf"
    ? node.tabs.flatMap((tab) => (tab.variant === "resource" ? [tab.resource] : []))
    : [...tabResources(node.first), ...tabResources(node.last)];

interface TabsQuery {
  panel: panel.Key;
}

// Reads the records of one type that the tabs of a panel display. The list follows the
// panel, so a tab added or removed while the node is open shows at once.
const followTabs = <D extends record.Keyed<string>>(
  client: Client,
  type: resource.Type,
  records: Tree.Retriever<{ where: { key: { in: string[] } } }, D>,
): Tree.Retriever<TabsQuery, D> => {
  const keysOf = (pan: panel.Panel): string[] =>
    tabResources(pan.root)
      .filter((id) => id.type === type)
      .map(({ key }) => key);
  const list = async (pan: panel.Panel): Promise<D[]> => {
    const keys = keysOf(pan);
    return keys.length === 0 ? [] : await records.retrieve(query.keysRequest(keys));
  };
  function onChange(
    key: string,
    handler: query.ChangeHandler<D>,
  ): destructor.Destructor;
  function onChange(
    tabs: TabsQuery,
    handler: query.ChangeHandler<D[]>,
  ): destructor.Destructor;
  function onChange(
    target: string | TabsQuery,
    handler: query.ChangeHandler<D> | query.ChangeHandler<D[]>,
  ): destructor.Destructor {
    if (typeof target === "string")
      return records.onChange(target, handler as query.ChangeHandler<D>);
    let latest = 0;
    return client.panels.onChange(target.panel, (pan) => {
      if (!query.isLive(pan)) return;
      const call = ++latest;
      void list(pan).then((listed) => {
        if (call === latest) (handler as query.ChangeHandler<D[]>)(listed);
      });
    });
  }
  function getCached(key: string): query.Cached<D> | undefined;
  function getCached(tabs: TabsQuery): query.Cached<D[]> | undefined;
  function getCached(
    target: string | TabsQuery,
  ): query.Cached<D> | query.Cached<D[]> | undefined {
    if (typeof target === "string") return records.getCached(target);
    const pan = client.panels.getCached(target.panel);
    if (!query.isLive(pan)) return undefined;
    const listed: D[] = [];
    for (const key of keysOf(pan)) {
      const cached = records.getCached(key);
      if (!query.isLive(cached)) return undefined;
      listed.push(cached);
    }
    return listed;
  }
  return {
    retrieve: async ({ panel: key }) => await list(await client.panels.retrieve(key)),
    onChange,
    getCached,
  };
};

const SOURCES: ((client: Client, tabs: TabsQuery) => Tree.Source)[] = [
  (client, tabs) =>
    Tree.createSource(
      followTabs(client, "schematic", client.schematics),
      tabs,
      Tree.toResource("schematic"),
    ),
  (client, tabs) =>
    Tree.createSource(
      followTabs(client, "lineplot", client.lineplots),
      tabs,
      Tree.toResource("lineplot"),
    ),
  (client, tabs) =>
    Tree.createSource(
      followTabs(client, "log", client.logs),
      tabs,
      Tree.toResource("log"),
    ),
  (client, tabs) =>
    Tree.createSource(
      followTabs(client, "table", client.tables),
      tabs,
      Tree.toResource("table"),
    ),
  (client, tabs) =>
    Tree.createSource(
      followTabs(client, "arc", client.arcs),
      tabs,
      Tree.toResource("arc"),
    ),
  (client, tabs) =>
    Tree.createSource(followTabs(client, "task", client.tasks), tabs, (tsk) =>
      Tree.toResource("task")(tsk.payload),
    ),
  (client, tabs) =>
    Tree.createSource(followTabs(client, "range", client.ranges), tabs, (rng) =>
      Tree.toResource("range")(rng.payload),
    ),
];

/** Lists the resources that the tabs of the panel display. */
const children: Tree.Children = ({ client, id }) =>
  SOURCES.map((source) => source(client, { panel: id.key }));

const TreeItem = Tree.createItem({ type: "panel", icon, children });

export const TREE_ITEMS: Tree.Items = { panel: TreeItem };
