// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, rack } from "@synnaxlabs/client";
import { Access, Icon, List, Menu, Rack, Text, Tree } from "@synnaxlabs/pluto";
import { useCallback, useMemo } from "react";

import { Arc } from "@/feature/arc";
import { ContextMenu } from "@/platform/context-menu";
import { Group } from "@/platform/group";
import { Ontology } from "@/platform/ontology";

const CreateArcIcon = Icon.createComposite(Icon.Arc, {
  topRight: Icon.Add,
});

const useRename = Ontology.createUseRename({
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
        id={List.itemNameID(itemKey)}
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

const useDelete = Ontology.createUseDelete({
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
  const ontologyIDs = useMemo(
    () => ids.map((id) => rack.ontologyID(Number(id.key))),
    [ids],
  );
  const hasUpdatePermission = Access.useUpdateGranted(ontologyIDs);
  const hasArcCreatePermission = Access.useCreateGranted(arc.TYPE_ONTOLOGY_ID);
  const hasDeletePermission = Access.useDeleteGranted(ontologyIDs);
  const handleDelete = useDelete(props);
  const rename = useRename(props);
  const group = Group.useCreateFromSelection();
  const create = Arc.useCreate();
  const isSingle = ids.length === 1;
  const handleCreate = useCallback(() => create(), [create]);
  return (
    <ContextMenu.Menu>
      {hasUpdatePermission && (
        <Group.ContextMenuItem
          ids={ids}
          rootID={rootID}
          shape={shape}
          showBottomDivider
          onClick={() => group(props)}
        />
      )}
      {hasUpdatePermission && isSingle && (
        <>
          <ContextMenu.RenameItem onClick={rename} />
          {hasArcCreatePermission && (
            <Menu.Item itemKey="createArc" onClick={handleCreate}>
              <CreateArcIcon />
              Create Arc automation
            </Menu.Item>
          )}
          <Menu.Divider />
        </>
      )}
      {hasDeletePermission && <ContextMenu.DeleteItem onClick={handleDelete} />}
      <Menu.Divider />
      {isSingle && (
        <>
          <Ontology.CopyPropertiesContextMenuItem {...props} />
          <Menu.Divider />
        </>
      )}
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: "rack",
  icon: <Icon.Rack />,
  TreeContextMenu,
  Item,
};
