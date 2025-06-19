// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/rack/ontology.css";

import { ontology, rack } from "@synnaxlabs/client";
import { Icon, Menu as PMenu, Status, Text, Tooltip, Tree } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { useEffect, useRef } from "react";

import { Menu } from "@/components";
import { Group } from "@/group";
import { useState } from "@/hardware/rack/StateContext";
import { Sequence } from "@/hardware/task/sequence";
import { useCopyToClipboard } from "@/hooks/useCopyToClipboard";
import { Layout } from "@/layout";
import { Modals } from "@/modals";
import { Ontology } from "@/ontology";

const CreateSequenceIcon = Icon.createComposite(Icon.Control, {
  topRight: Icon.Add,
});

const useDelete = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const confirm = Ontology.useConfirmDelete({ type: "Rack" });
  return useMutation<void, Error, Ontology.TreeContextMenuProps, Tree.Node[]>({
    onMutate: async ({ state: { nodes, setNodes }, selection: { resources } }) => {
      if (!(await confirm(resources))) throw new errors.Canceled();
      const prevNodes = Tree.deepCopy(nodes);
      setNodes([
        ...Tree.removeNode({
          tree: nodes,
          keys: resources.map(({ id }) => id.toString()),
        }),
      ]);
      return prevNodes;
    },
    mutationFn: async ({ selection: { resources }, client }) =>
      await client.hardware.racks.delete(resources.map(({ id }) => Number(id.key))),
    onError: (e, { handleError, state: { setNodes } }, prevNodes) => {
      if (prevNodes != null) setNodes(prevNodes);
      if (errors.Canceled.matches(e)) return;
      handleError(e, "Failed to delete racks");
    },
  }).mutate;
};

const useCopyKeyToClipboard = (): ((props: Ontology.TreeContextMenuProps) => void) => {
  const copy = useCopyToClipboard();
  return ({ selection: { resources } }) => {
    copy(resources[0].id.key, `key to ${resources[0].name}`);
  };
};

const handleRename: Ontology.HandleTreeRename = {
  execute: async ({ client, id, name }) => {
    const rack = await client.hardware.racks.retrieve(id.key);
    await client.hardware.racks.create({ ...rack, name });
  },
};

const Item: Tree.Item = ({ entry, ...rest }: Tree.ItemProps) => {
  const id = new ontology.ID(entry.key);
  const state = useState(id.key);

  const heartRef = useRef<SVGSVGElement>(null);

  const variant = state?.variant ?? "disabled";

  useEffect(() => {
    if (variant !== "success") return;
    const heart = heartRef.current;
    if (!heart) return;
    heart.classList.remove("synnax-rack-heartbeat--beat");
    requestAnimationFrame(() => heart.classList.add("synnax-rack-heartbeat--beat"));
  }, [state]);

  return (
    <Tree.DefaultItem {...rest} entry={entry}>
      {({ entry, onRename, key }) => (
        <>
          <Text.MaybeEditable
            id={`text-${key}`}
            level="p"
            allowDoubleClick={false}
            value={entry.name}
            disabled={!entry.allowRename}
            onChange={(name) => onRename?.(entry.key, name)}
            style={{
              textOverflow: "ellipsis",
              width: 0,
              overflow: "hidden",
              flexGrow: 1,
            }}
          />
          <Tooltip.Dialog location="right">
            <Status.Text variant={variant} hideIcon level="small" weight={450}>
              {state?.message}
            </Status.Text>
            <Icon.Heart
              ref={heartRef}
              className="synnax-rack-heartbeat"
              style={{ color: Status.VARIANT_COLORS[variant] }}
            />
          </Tooltip.Dialog>
        </>
      )}
    </Tree.DefaultItem>
  );
};

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const { selection } = props;
  const { nodes } = selection;
  const handleDelete = useDelete();
  const placeLayout = Layout.usePlacer();
  const rename = Modals.useRename();
  const handleError = Status.useErrorHandler();
  const group = Group.useCreateFromSelection();
  const copyKeyToClipboard = useCopyKeyToClipboard();
  const createSequence = () => {
    Sequence.createLayout({ rename, rackKey: Number(selection.resources[0].id.key) })
      .then((layout) => {
        if (layout == null) return;
        placeLayout(layout);
      })
      .catch((e) => handleError(e, "Failed to create control sequence"));
  };
  const onSelect = {
    group: () => group(props),
    rename: () => Tree.startRenaming(nodes[0].key),
    createSequence,
    copy: () => copyKeyToClipboard(props),
    delete: () => handleDelete(props),
  };
  const isSingle = nodes.length === 1;
  return (
    <PMenu.Menu level="small" iconSpacing="small" onChange={onSelect}>
      <Group.MenuItem selection={selection} showBottomDivider />
      {isSingle && (
        <>
          <Menu.RenameItem />
          <PMenu.Item itemKey="createSequence" startIcon={<CreateSequenceIcon />}>
            Create Control Sequence
          </PMenu.Item>
          <PMenu.Item itemKey="copy" startIcon={<Icon.Copy />}>
            Copy Key
          </PMenu.Item>
          <PMenu.Divider />
        </>
      )}
      <Menu.DeleteItem />
      <PMenu.Divider />
      <Menu.HardReloadItem />
    </PMenu.Menu>
  );
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  type: rack.ONTOLOGY_TYPE,
  icon: <Icon.Rack />,
  hasChildren: true,
  canDrop: () => false,
  onSelect: () => {},
  haulItems: () => [],
  allowRename: () => true,
  onRename: handleRename,
  TreeContextMenu,
  Item,
};
