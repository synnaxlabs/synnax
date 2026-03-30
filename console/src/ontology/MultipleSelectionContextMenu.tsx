// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Access, Menu } from "@synnaxlabs/pluto";

import { ContextMenu } from "@/components";
import { Group } from "@/group";
import { type TreeContextMenu } from "@/ontology/service";

export const MultipleSelectionContextMenu: TreeContextMenu = (props) => {
  const {
    selection: { ids, rootID },
    state: { shape },
  } = props;
  const groupFromSelection = Group.useCreateFromSelection();
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  return (
    <ContextMenu.Menu>
      {hasUpdatePermission && (
        <>
          <Group.ContextMenuItem
            ids={ids}
            shape={shape}
            rootID={rootID}
            onClick={() => groupFromSelection(props)}
          />
          <Menu.Divider />
        </>
      )}
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};
