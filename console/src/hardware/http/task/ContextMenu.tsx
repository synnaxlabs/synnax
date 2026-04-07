// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, Menu } from "@synnaxlabs/pluto";

import { ContextMenu as CMenu } from "@/components";
import { Common } from "@/hardware/common";

export interface ContextMenuProps {
  keys: string[];
  onDelete: (keys: string[]) => void;
  onDuplicate?: (keys: string[]) => void;
  onRename?: (key: string) => void;
}

export const ContextMenu = ({
  keys,
  onDuplicate,
  onDelete,
  onRename,
}: ContextMenuProps) => {
  const isSnapshot = Common.Task.useIsSnapshot();
  const canAct = keys.length > 0;
  const canDuplicate = onDuplicate != null;
  const canRename = onRename != null && keys.length === 1;
  return (
    <CMenu.Menu>
      {!isSnapshot && canAct && (
        <>
          {canRename && <CMenu.RenameItem onClick={() => onRename(keys[0])} />}
          {canDuplicate && (
            <Menu.Item itemKey="duplicate" onClick={() => onDuplicate?.(keys)}>
              <Icon.Copy />
              Duplicate
            </Menu.Item>
          )}
          <CMenu.DeleteItem onClick={() => onDelete(keys)} />
          <Menu.Divider />
        </>
      )}
      <CMenu.ReloadConsoleItem />
    </CMenu.Menu>
  );
};
