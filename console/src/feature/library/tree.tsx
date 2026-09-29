// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { library } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Access, Library } from "@synnaxlabs/pluto";

import { useImport } from "@/feature/library/useImport";
import { ContextMenu } from "@/platform/context-menu";
import { Core } from "@/platform/core";
import { Link } from "@/platform/link";
import { Panel } from "@/platform/panel";
import { Tree } from "@/platform/tree";

const useRename = Tree.createUseRename({
  query: Library.useRename,
  ontologyID: library.ontologyID,
  convertKey: String,
});

const useDelete = Tree.createUseDelete({
  type: "Library",
  query: Library.useDelete,
  convertKey: String,
});

const TreeContextMenu: Tree.ContextMenu = (props) => {
  const {
    selection: { ids },
    state: { getResource },
  } = props;
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const rename = useRename(props);
  const handleDelete = useDelete(props);
  const handleImport = useImport();
  const handleLink = Core.useCopyLinkToClipboard();
  const [firstID] = ids;
  const single = ids.length === 1;
  return (
    <ContextMenu.Menu>
      {hasUpdatePermission && single && (
        <>
          <ContextMenu.RenameItem onClick={rename} />
          <Menu.Item itemKey="import" onClick={() => handleImport(firstID.key)}>
            <Icon.Import />
            Import
          </Menu.Item>
        </>
      )}
      <Menu.Divider />
      {single && (
        <Link.CopyContextMenuItem
          onClick={() =>
            handleLink({ name: getResource(firstID).name, ontologyID: firstID })
          }
        />
      )}
      <Menu.Divider />
      {hasDeletePermission && <ContextMenu.DeleteItem onClick={handleDelete} />}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const TreeItem = Tree.createItem({
  type: "library",
  icon: <Icon.Library />,
  hasChildren: false,
  useOnSelect: Panel.useOpenResource,
  ContextMenu: TreeContextMenu,
});

export const TREE_ITEMS = { library: TreeItem } satisfies Tree.Items;
