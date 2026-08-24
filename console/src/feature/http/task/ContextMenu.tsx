// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, Menu } from "@synnaxlabs/pluto";

import { ContextMenu as Base } from "@/platform/context-menu";
import { Task } from "@/platform/task";

export interface ContextMenuProps {
  keys: string[];
  onRemove: (keys: string[]) => void;
  onDuplicate?: (keys: string[]) => void;
  onRename?: (key: string) => void;
  onAddField?: () => void;
  onEnable?: () => void;
  onDisable?: () => void;
}

export const ContextMenu = ({
  keys,
  onDuplicate,
  onRemove,
  onRename,
  onAddField,
  onEnable,
  onDisable,
}: ContextMenuProps) => {
  const isPreview = Task.useIsPreview();
  const canAct = keys.length > 0;
  const canDuplicate = onDuplicate != null;
  const canRename = onRename != null && keys.length === 1;
  return (
    <Base.Menu>
      {!isPreview && canAct && (
        <>
          {onAddField != null && (
            <Menu.Item itemKey="addField" onClick={onAddField}>
              <Icon.Add />
              Add field
            </Menu.Item>
          )}
          {canRename && <Base.RenameItem onClick={() => onRename(keys[0])} />}
          {onEnable != null && (
            <Menu.Item itemKey="enable" onClick={onEnable}>
              <Icon.Enable />
              Enable
            </Menu.Item>
          )}
          {onDisable != null && (
            <Menu.Item itemKey="disable" onClick={onDisable}>
              <Icon.Disable />
              Disable
            </Menu.Item>
          )}
          {canDuplicate && (
            <Menu.Item itemKey="duplicate" onClick={() => onDuplicate?.(keys)}>
              <Icon.Copy />
              Duplicate
            </Menu.Item>
          )}
          <Menu.Divider />
          <Base.RemoveItem onClick={() => onRemove(keys)} />
          <Menu.Divider />
        </>
      )}
      <Base.ReloadConsoleItem />
    </Base.Menu>
  );
};
