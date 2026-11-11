// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Menu } from "@synnaxlabs/lyra/menu";
import { Access } from "@synnaxlabs/pluto";

import { ContextMenu } from "@/platform/context-menu";
import { ContextMenuItem } from "@/platform/group/ContextMenuItem";
import { useCreateFromSelection } from "@/platform/group/useCreateFromSelection";
import { Tree } from "@/platform/tree";
import { Vcs } from "@/platform/vcs";

/** The menu for a tree selection of more than one resource type. */
export const SelectionContextMenu: Tree.ContextMenu = (props) => {
  const {
    selection: { ids, rootID, parentID },
    state: { shape },
  } = props;
  const groupFromSelection = useCreateFromSelection();
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  return (
    <ContextMenu.Menu>
      {hasUpdatePermission && (
        <>
          <ContextMenuItem
            ids={ids}
            shape={shape}
            rootID={rootID}
            parentID={parentID}
            onClick={() => groupFromSelection(props)}
          />
          <Menu.Divider />
        </>
      )}
      <Vcs.CommitContextMenuItem {...props} />
      <Menu.Divider />
      <Tree.CopyContextMenuItem {...props} />
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};
