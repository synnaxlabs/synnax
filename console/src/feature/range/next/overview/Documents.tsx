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
  type query,
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
import {
  Access,
  LinePlot,
  Log,
  Schematic,
  Synnax,
  Table,
  Task,
} from "@synnaxlabs/pluto";
import { type ReactElement, useEffect, useSyncExternalStore } from "react";

import { Section } from "@/feature/range/next/overview/Section";
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

const NO_RECORDS: Owned[] = [];
const noop = () => {};

const useOwned = (
  list: query.List<Owned, unknown> | undefined,
  type: resource.Type,
): Document[] => {
  const window = useSyncExternalStore(
    (callback) => list?.onChange(callback) ?? noop,
    () => list?.getCached(),
  );
  useEffect(() => {
    list?.retrieve().catch(console.error);
  }, [list]);
  return (window?.records ?? NO_RECORDS).map((r) => ({
    id: { type, key: r.key },
    name: r.name,
    snapshot: r.snapshot === true,
  }));
};

/** @returns every document that the range owns, by kind and then by name. */
export const useDocuments = (rangeKey: ranger.Key): Document[] => {
  const client = Synnax.use();
  const where = { where: { ranges: [rangeKey] } };
  const plots = useOwned(
    client?.lineplots.watch(where),
    lineplot.TYPE_RESOURCE_ID.type,
  );
  const schematics = useOwned(
    client?.schematics.watch(where),
    schematic.TYPE_RESOURCE_ID.type,
  );
  const tables = useOwned(client?.tables.watch(where), table.TYPE_RESOURCE_ID.type);
  const logs = useOwned(client?.logs.watch(where), log.TYPE_RESOURCE_ID.type);
  const tasks = useOwned(client?.tasks.watch(where), task.TYPE_RESOURCE_ID.type);
  return [plots, schematics, tables, logs, tasks].flatMap((docs) =>
    docs.toSorted((a, b) => a.name.localeCompare(b.name)),
  );
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
