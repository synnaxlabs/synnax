// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Arc, Component, Icon, Menu as PMenu, Text } from "@synnaxlabs/pluto";

import { Editor } from "@/arc/editor";
import { translateGraphToConsole } from "@/arc/types/translate";
import { Menu } from "@/components";
import { Layout } from "@/layout";
// import { Modals } from "@/modals";
import { useConfirmDelete } from "@/ontology/hooks";

export interface ContextMenuProps extends PMenu.ContextMenuMenuProps {}

export const ContextMenu = ({ keys }: ContextMenuProps) => {
  const q = Arc.useRetrieveMultiple({ keys });
  const { update: handleToggleDeploy } = Arc.useToggleDeploy();
  const placeLayout = Layout.usePlacer();
  // const ctx = Form.useContext();
  // const rename = Modals.useRename();
  const confirm = useConfirmDelete({
    type: "Arc",
    description: "Deleting this Arc will permanently remove it.",
  });
  const { update: handleDelete } = Arc.useDelete();

  if (q.variant !== "success") return <Layout.DefaultContextMenu />;
  const arcs = q.data;
  const canStart = arcs.some((arc) => arc.deploy === false);
  const canStop = arcs.some((arc) => arc.deploy === true);
  const someSelected = arcs.length > 0;
  const isSingle = arcs.length === 1;

  const handleSelect: PMenu.MenuProps["onChange"] = {
    start: () =>
      arcs.forEach((arc) => {
        if (!arc.deploy) handleToggleDeploy(arc.key);
      }),
    stop: () =>
      arcs.forEach((arc) => {
        if (arc.deploy) handleToggleDeploy(arc.key);
      }),
    edit: () => {
      const graph = translateGraphToConsole(arcs[0].graph);
      placeLayout(Editor.create({ key: arcs[0].key, name: arcs[0].name, graph }));
    },
    rename: () => {
      Text.edit(`text-${arcs[0].key}`);
      // rename({ initialValue: arcs[0].name }, { icon: "Arc", name: "Arc.Rename" })
      //   .then((renamed) => {
      //     if (renamed == null) return;
      //     ctx.set("name", renamed);
      //   })
      //   .catch(console.error);
    },
    delete: () => {
      confirm(arcs)
        .then((confirmed) => {
          if (confirmed) handleDelete(arcs.map((a) => a.key));
        })
        .catch(console.error);
    },
  };

  return (
    <PMenu.Menu level="small" gap="small" onChange={handleSelect}>
      {canStart && (
        <PMenu.Item itemKey="start">
          <Icon.Play />
          Start
        </PMenu.Item>
      )}
      {canStop && (
        <PMenu.Item itemKey="stop">
          <Icon.Pause />
          Stop
        </PMenu.Item>
      )}
      {(canStart || canStop) && <PMenu.Divider />}
      {isSingle && (
        <>
          <PMenu.Item itemKey="edit">
            <Icon.Edit />
            Edit Arc
          </PMenu.Item>
          <PMenu.Divider />
          <Menu.RenameItem />
          <PMenu.Divider />
        </>
      )}
      {someSelected && (
        <>
          <PMenu.Item itemKey="delete">
            <Icon.Delete />
            Delete
          </PMenu.Item>
          <PMenu.Divider />
        </>
      )}
      <Menu.ReloadConsoleItem />
    </PMenu.Menu>
  );
};

export const contextMenuRenderProp = Component.renderProp(ContextMenu);
