// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { CSS as LyraCSS } from "@synnaxlabs/lyra/css";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Access, Ontology } from "@synnaxlabs/pluto";
import { type ReactElement, useState } from "react";

import { Explorer } from "@/feature/arc/explorer";
import { RootContextMenu } from "@/feature/arc/tree";
import { Arc } from "@/platform/arc";
import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { type Nav } from "@/platform/nav";
import { Search } from "@/platform/search";
import { Toolbar } from "@/platform/toolbar";
import { Tree } from "@/platform/tree";

interface EmptyContentProps {
  onCreate: () => void;
}

const EmptyContent = ({ onCreate }: EmptyContentProps) => {
  const hasCreatePermission = Access.useCreateGranted(arc.TYPE_ONTOLOGY_ID);
  return (
    <Empty.Action
      message="No Arcs"
      action={hasCreatePermission ? "Create Arc automation" : undefined}
      onClick={onCreate}
    />
  );
};

// A group fails, so it shows only while one of its Arcs passes.
const typeFilter =
  (types: arc.Type[]): Tree.Filter =>
  ({ id, data }) =>
    id.type === "arc" && types.includes((data as Pick<arc.Arc, "type">).type);

interface TypeFilterProps {
  value: arc.Type[];
  onChange: (types: arc.Type[]) => void;
}

const TypeFilter = ({ value, onChange }: TypeFilterProps): ReactElement => {
  const toggle = (type: arc.Type) => () =>
    onChange(value.includes(type) ? value.filter((t) => t !== type) : [...value, type]);
  return (
    <Search.FilterMenu size="tiny" variant="text">
      <Menu.Item
        itemKey="automation"
        className={CSS.cls(LyraCSS.selected(value.includes("automation")))}
        onClick={toggle("automation")}
      >
        <Icon.Arc />
        Automation
      </Menu.Item>
      <Menu.Item
        itemKey="module"
        className={CSS.cls(LyraCSS.selected(value.includes("module")))}
        onClick={toggle("module")}
      >
        <Icon.Module />
        Module
      </Menu.Item>
    </Search.FilterMenu>
  );
};

const Content = (): ReactElement => {
  const groupID = Ontology.useRootChildID({ name: "Arcs" });
  const create = Arc.useCreate();
  const openExplorer = Explorer.useOpenTab();
  const hasCreatePermission = Access.useCreateGranted(arc.TYPE_ONTOLOGY_ID);
  const hasRetrievePermission = Access.useRetrieveGranted(arc.TYPE_ONTOLOGY_ID);
  const [searching, setSearching] = useState(false);
  const [types, setTypes] = useState<arc.Type[]>([...arc.TYPES]);
  return (
    <Toolbar.Content className={CSS.B("arc-toolbar")}>
      <Toolbar.Header>
        <Toolbar.Title>
          <Icon.Arc />
          Arcs
        </Toolbar.Title>
        <Toolbar.Actions>
          <Toolbar.SearchAction value={searching} onChange={setSearching} />
          {hasRetrievePermission && (
            <Toolbar.Action tooltip="Open Arc explorer" onClick={openExplorer}>
              <Icon.Explore />
            </Toolbar.Action>
          )}
          {hasCreatePermission && (
            <Toolbar.Action
              tooltip="Create Arc automation"
              onClick={create}
              variant="filled"
            >
              <Icon.Add />
            </Toolbar.Action>
          )}
        </Toolbar.Actions>
      </Toolbar.Header>
      <Toolbar.Body>
        <Tree.Tree
          root={groupID}
          search={searching}
          filter={
            searching && types.length < arc.TYPES.length ? typeFilter(types) : undefined
          }
          bar={searching ? <TypeFilter value={types} onChange={setTypes} /> : undefined}
          emptyContent={<EmptyContent onCreate={create} />}
          rootContextMenu={RootContextMenu}
        />
      </Toolbar.Body>
    </Toolbar.Content>
  );
};

export const TOOLBAR: Nav.Toolbar = {
  key: "arc",
  icon: <Icon.Arc />,
  content: <Content />,
  trigger: ["A"],
  tooltip: "Arcs",
  sizeBounds: { lower: 225, upper: 400 },
  initialSize: 300,
  useVisible: () => Access.useRetrieveGranted(arc.TYPE_ONTOLOGY_ID),
};
