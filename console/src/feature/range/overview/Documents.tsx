// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  lineplot,
  log,
  type ranger,
  resource,
  schematic,
  table,
  task,
} from "@synnaxlabs/client";
import { Badge } from "@synnaxlabs/lyra/badge";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Flux, LinePlot, Log, Schematic, Table, Task } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Section } from "@/feature/range/overview/Section";
import { ContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Panel } from "@/platform/panel";

interface Document {
  id: resource.ID;
  name: string;
  snapshot: boolean;
}

interface Kind {
  name: string;
  /** The path of the kind's icon in the icon registry. */
  icon: string;
}

const KINDS: Record<string, Kind> = {
  [lineplot.TYPE_RESOURCE_ID.type]: { name: "Line plot", icon: "LinePlot" },
  [schematic.TYPE_RESOURCE_ID.type]: { name: "Schematic", icon: "Schematic" },
  [table.TYPE_RESOURCE_ID.type]: { name: "Table", icon: "Table" },
  [log.TYPE_RESOURCE_ID.type]: { name: "Log", icon: "Log" },
  [task.TYPE_RESOURCE_ID.type]: { name: "Task", icon: "Task" },
};

interface Owned {
  key: string;
  name: string;
  snapshot?: boolean;
}

const toDocuments = (owned: Owned[], type: resource.Type): Document[] =>
  owned.map(({ key, name, snapshot }) => ({
    id: { type, key },
    name,
    snapshot: snapshot === true,
  }));

/** @returns every document that the range owns, by kind and then by name. */
export const useDocuments = (rangeKey: ranger.Key): Document[] => {
  const params = { query: { where: { ranges: [rangeKey] } }, eager: true };
  const plots = Flux.useList((c) => c.lineplots, { name: "line plots", ...params });
  const schematics = Flux.useList((c) => c.schematics, {
    name: "schematics",
    ...params,
  });
  const tables = Flux.useList((c) => c.tables, { name: "tables", ...params });
  const logs = Flux.useList((c) => c.logs, { name: "logs", ...params });
  const tasks = Flux.useList((c) => c.tasks, { name: "tasks", ...params });
  return [
    toDocuments(plots.getItem(plots.data), lineplot.TYPE_RESOURCE_ID.type),
    toDocuments(schematics.getItem(schematics.data), schematic.TYPE_RESOURCE_ID.type),
    toDocuments(tables.getItem(tables.data), table.TYPE_RESOURCE_ID.type),
    toDocuments(logs.getItem(logs.data), log.TYPE_RESOURCE_ID.type),
    toDocuments(tasks.getItem(tasks.data), task.TYPE_RESOURCE_ID.type),
  ].flatMap((docs) => docs.toSorted((a, b) => a.name.localeCompare(b.name)));
};

interface DocumentMenuProps {
  entry: Document;
}

const DocumentMenu = ({ entry: { id, name } }: DocumentMenuProps): ReactElement => {
  const openTab = Panel.useOpenTab();
  const kind = KINDS[id.type];
  const confirm = Modals.useConfirmDelete({ type: kind.name, icon: kind.icon });
  const canDelete = Access.useDeleteGranted(id);
  const params = { beforeUpdate: async () => await confirm({ name }) };
  // A hook cannot be picked by kind, so the menu holds the delete of each kind.
  const deletes: Record<string, (key: string) => void> = {
    [lineplot.TYPE_RESOURCE_ID.type]: LinePlot.useDelete(params).update,
    [schematic.TYPE_RESOURCE_ID.type]: Schematic.useDelete(params).update,
    [table.TYPE_RESOURCE_ID.type]: Table.useDelete(params).update,
    [log.TYPE_RESOURCE_ID.type]: Log.useDelete(params).update,
    [task.TYPE_RESOURCE_ID.type]: Task.useDelete(params).update,
  };
  return (
    <ContextMenu.Menu>
      <Menu.Item
        itemKey="open"
        onClick={() => openTab({ variant: "resource", resource: id })}
      >
        {Icon.resolve(kind.icon)}
        Open
      </Menu.Item>
      <Menu.Divider />
      {canDelete && <ContextMenu.DeleteItem onClick={() => deletes[id.type](id.key)} />}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

export const Documents = ({ documents }: { documents: Document[] }): ReactElement => {
  const openTab = Panel.useOpenTab();
  const menuProps = Menu.useContextMenu();
  const menu = ({ keys }: Menu.ContextMenuMenuProps): ReactElement => {
    const entry = documents.find(({ id }) => resource.idToString(id) === keys[0]);
    if (entry == null)
      return (
        <ContextMenu.Menu>
          <ContextMenu.ReloadConsoleItem />
        </ContextMenu.Menu>
      );
    return <DocumentMenu entry={entry} />;
  };
  return (
    <Section title="Attached" icon={<Icon.Attachment />}>
      <Menu.ContextMenu menu={menu} {...menuProps}>
        <Flex.Box
          y
          empty
          className={CSS.BE("range-overview", "fields")}
          onContextMenu={menuProps.open}
        >
          {documents.map(({ id, name, snapshot }) => {
            const kind = KINDS[id.type];
            const key = resource.idToString(id);
            return (
              <Button.Button
                key={key}
                variant="text"
                gap="medium"
                justify="start"
                className={CSS.cls(
                  CSS.BE("range-overview", "document"),
                  Menu.CONTEXT_TARGET,
                )}
                data-menu-key={key}
                onClick={() => openTab({ variant: "resource", resource: id })}
              >
                <Text.Text
                  el="span"
                  color={9}
                  className={CSS.BE("range-overview", "document-icon")}
                >
                  {Icon.resolve(kind.icon)}
                </Text.Text>
                <Text.Text
                  el="span"
                  overflow="fade"
                  className={CSS.BE("range-overview", "document-name")}
                >
                  {name}
                </Text.Text>
                {snapshot && <Badge.Badge>Snapshot</Badge.Badge>}
                <Text.Text
                  el="span"
                  level="small"
                  color={8}
                  className={CSS.BE("range-overview", "document-kind")}
                >
                  {kind.name}
                </Text.Text>
              </Button.Button>
            );
          })}
        </Flex.Box>
      </Menu.ContextMenu>
    </Section>
  );
};
