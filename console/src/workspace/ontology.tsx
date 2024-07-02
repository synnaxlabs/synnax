// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.
import { ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Menu as PMenu, Tree } from "@synnaxlabs/pluto";
import { deep, type UnknownRecord } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Cluster } from "@/cluster";
import { Menu } from "@/components/menu";
import { Group } from "@/group";
import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { Link } from "@/link";
import { Ontology } from "@/ontology";
import { Schematic } from "@/schematic";
import { selectActiveKey } from "@/workspace/selectors";
import { add, rename, setActive } from "@/workspace/slice";

const TreeContextMenu: Ontology.TreeContextMenu = (props): ReactElement => {
  const {
    selection,
    client,
    services,
    placeLayout,
    store,
    state: { nodes, setNodes, resources, setResources },
    addStatus,
  } = props;

  const handleDelete = (): void => {
    void (async () => {
      const s = store.getState();
      const activeKey = selectActiveKey(s);
      const active = selection.resources.find((r) => r.id.key === activeKey);
      if (active != null) {
        store.dispatch(setActive(null));
        store.dispatch(Layout.clearWorkspace());
      }
      await client.workspaces.delete(...selection.resources.map((r) => r.id.key));
      const next = Tree.removeNode({
        tree: nodes,
        keys: selection.resources.map((r) => r.id.toString()),
      });
      setNodes([...next]);
    })();
  };

  const handleCreateNewSchematic = (): void => {
    const workspace = selection.resources[0].id.key;
    void (async () => {
      const schematic = await client.workspaces.schematic.create(workspace, {
        name: "New Schematic",
        snapshot: false,
        data: deep.copy(Schematic.ZERO_STATE) as unknown as UnknownRecord,
      });
      const otg = await client.ontology.retrieve(
        new ontology.ID({ key: schematic.key, type: "schematic" }),
      );
      placeLayout(
        Schematic.create({
          ...(schematic.data as unknown as Schematic.State),
          key: schematic.key,
          name: schematic.name,
          snapshot: schematic.snapshot,
        }),
      );
      setResources([...resources, otg]);
      const nextNodes = Tree.setNode({
        tree: nodes,
        destination: selection.resources[0].key,
        additions: Ontology.toTreeNodes(services, [otg]),
      });
      setNodes([...nextNodes]);
    })();
  };

  const handleCreateNewLinePlot = (): void => {
    const workspace = selection.resources[0].id.key;
    void (async () => {
      const linePlot = await client.workspaces.linePlot.create(workspace, {
        name: "New Line Plot",
        data: deep.copy(LinePlot.ZERO_SLICE_STATE) as unknown as UnknownRecord,
      });
      const otg = await client.ontology.retrieve(
        new ontology.ID({ key: linePlot.key, type: "lineplot" }),
      );
      placeLayout(
        LinePlot.create({
          ...(linePlot.data as unknown as LinePlot.SliceState),
          key: linePlot.key,
          name: linePlot.name,
        }),
      );
      setResources([...resources, otg]);
      const nextNodes = Tree.setNode({
        tree: nodes,
        destination: selection.resources[0].key,
        additions: Ontology.toTreeNodes(services, [otg]),
      });
      setNodes([...nextNodes]);
    })();
  };

  const handleRename = (): void => {
    return Tree.startRenaming(selection.resources[0].id.toString());
  };

  const handleGroup = (): void => {
    void Group.fromSelection(props);
  };

  const clusterKey = Cluster.useSelectActiveKey();
  const handleLink = (): void => {
    if (clusterKey == null) return;
    Link.CopyToClipboard({
      clusterKey,
      resource: {
        type: "workspace",
        key: selection.resources[0].id.key,
      },
      addStatus,
      name: selection.resources[0].name,
    });
  };

  const f: Record<string, () => void> = {
    delete: handleDelete,
    rename: handleRename,
    group: handleGroup,
    plot: handleCreateNewLinePlot,
    schematic: handleCreateNewSchematic,
    link: handleLink,
  };
  const onSelect = (key: string): void => f[key]?.();

  const singleResource = selection.resources.length === 1;
  return (
    <PMenu.Menu onChange={onSelect} level="small" iconSpacing="small">
      <Menu.DeleteItem />
      {singleResource && <Menu.RenameItem />}
      <Group.GroupMenuItem selection={props.selection} />
      {singleResource && (
        <>
          <PMenu.Item itemKey="plot" startIcon={<Icon.Visualize />}>
            New Line Plot
          </PMenu.Item>
          <PMenu.Item itemKey="schematic" startIcon={<Icon.Schematic />}>
            New Schematic
          </PMenu.Item>
          <Link.CopyMenuItem />
        </>
      )}
      <Menu.HardReloadItem />
    </PMenu.Menu>
  );
};

const handleSelect: Ontology.HandleSelect = ({ selection, client, store }) => {
  void (async () => {
    const workspace = await client.workspaces.retrieve(selection[0].id.key);
    store.dispatch(add({ workspaces: [workspace] }));
    store.dispatch(
      Layout.setWorkspace({
        slice: workspace.layout as unknown as Layout.SliceState,
        keepNav: false,
      }),
    );
  })();
};

const handleRename: Ontology.HandleTreeRename = ({
  client,
  id,
  name,
  store,
  state: { nodes, setNodes },
}) => {
  void client.workspaces.rename(id.key, name);
  store.dispatch(rename({ key: id.key, name }));

  const next = Tree.updateNode({
    tree: nodes,
    key: id.toString(),
    updater: (node) => ({ ...node, name }),
  });
  setNodes([...next]);
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  type: "workspace",
  icon: <Icon.Workspace />,
  hasChildren: true,
  canDrop: () => false,
  TreeContextMenu,
  onSelect: handleSelect,
  haulItems: () => [],
  allowRename: () => true,
  onRename: handleRename,
};
