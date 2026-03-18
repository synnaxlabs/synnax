// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, Menu as PMenu } from "@synnaxlabs/pluto";

import { Menu } from "@/components";
import { Common } from "@/hardware/common";

export interface ContextMenuProps {
  keys: string[];
  onDelete: (keys: string[]) => void;
  onDuplicate?: (keys: string[]) => void;
}

export const ContextMenu = ({ keys, onDuplicate, onDelete }: ContextMenuProps) => {
  const isSnapshot = Common.Task.useIsSnapshot();
  const canAct = keys.length > 0;
  const canDuplicate = onDuplicate != null;
  return (
    <PMenu.Menu level="small">
      {!isSnapshot && canAct && (
        <>
          {canDuplicate && (
            <PMenu.Item itemKey="duplicate" onClick={() => onDuplicate?.(keys)}>
              <Icon.Copy />
              Duplicate
            </PMenu.Item>
          )}
          <PMenu.Item itemKey="delete" onClick={() => onDelete(keys)}>
            <Icon.Close />
            Delete
          </PMenu.Item>
          <PMenu.Divider />
        </>
      )}
      <Menu.ReloadConsoleItem />
    </PMenu.Menu>
  );
};
