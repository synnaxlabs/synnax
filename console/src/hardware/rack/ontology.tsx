// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { rack } from "@synnaxlabs/client";
import {
  ContextMenu as PContextMenu,
  Icon,
  Rack,
  Status,
  Text,
  Tree,
} from "@synnaxlabs/pluto";

import { ContextMenu } from "@/components";
import { Group } from "@/group";
import { Sequence } from "@/hardware/task/sequence";
import { Layout } from "@/layout";
import { Modals } from "@/modals";
import { Ontology } from "@/ontology";
import { createUseDelete } from "@/ontology/createUseDelete";
import { createUseRename } from "@/ontology/createUseRename";

const CreateSequenceIcon = Icon.createComposite(Icon.Control, {
  topRight: Icon.Add,
});

const useRename = createUseRename({
  query: Rack.useRename,
  ontologyID: rack.ontologyID,
  convertKey: Number,
});

const Item = ({ id, resource, ...rest }: Ontology.TreeItemProps) => {
  const { itemKey } = rest;
  const res = Rack.useRetrieve({ key: Number(id.key) });
  const status = res.data?.status;

  return (
    <Tree.Item {...rest}>
      <Icon.Rack />
      <Text.MaybeEditable
        id={itemKey}
        allowDoubleClick={false}
        value={resource.name}
        overflow="ellipsis"
        style={{ width: 0 }}
        grow
        onChange
      />
      <Rack.StatusIndicator status={status} />
    </Tree.Item>
  );
};

const useDelete = createUseDelete({
  type: "Rack",
  query: Rack.useDelete,
  convertKey: Number,
});

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection,
    state: { shape },
  } = props;
  const { ids, rootID } = selection;
  const handleDelete = useDelete(props);
  const placeLayout = Layout.usePlacer();
  const openRenameModal = Modals.useRename();
  const handleRename = useRename(props);
  const handleError = Status.useErrorHandler();
  const group = Group.useCreateFromSelection();
  const createSequence = () => {
    handleError(async () => {
      const layout = await Sequence.createLayout({
        rename: openRenameModal,
        rackKey: Number(ids[0].key),
      });
      if (layout == null) return;
      placeLayout(layout);
    }, "Failed to create control sequence");
  };
  const handleGroup = () => group(props);
  const isSingle = ids.length === 1;
  return (
    <>
      <Group.ContextMenuItem
        ids={ids}
        rootID={rootID}
        shape={shape}
        showBottomDivider
        onClick={handleGroup}
      />
      {isSingle && (
        <>
          <ContextMenu.RenameItem onClick={handleRename} />
          <PContextMenu.Item onClick={createSequence}>
            <CreateSequenceIcon />
            Create control sequence
          </PContextMenu.Item>
          <PContextMenu.Divider />
        </>
      )}
      <ContextMenu.DeleteItem onClick={handleDelete} />
      <PContextMenu.Divider />
      {isSingle && (
        <>
          <Ontology.CopyContextMenuItem {...props} />
          <PContextMenu.Divider />
        </>
      )}
      <ContextMenu.ReloadConsoleItem />
    </>
  );
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: "rack",
  icon: <Icon.Rack />,
  TreeContextMenu,
  Item,
};
