// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Form, Menu as PMenu } from "@synnaxlabs/pluto";
import { type Key, type Keyed } from "@synnaxlabs/x";

import { Menu } from "@/components/menu";

interface ChannelListContextMenuProps<T> {
  keys: string[];
  value: T[];
  onSelect: (keys: string[], index: number) => void;
  remove: (indices: number[]) => void;
  path: string;
  onDuplicate?: (indices: number[]) => void;
  snapshot?: boolean;
  allowTare?: boolean;
  onTare?: (indices: number[]) => void;
}

export const ChannelListContextMenu = <
  K extends Key,
  T extends Keyed<K> & { enabled: boolean },
>({
  keys,
  value,
  onSelect,
  remove,
  onDuplicate,
  path,
  snapshot,
  allowTare,
  onTare,
}: ChannelListContextMenuProps<T>) => {
  const methods = Form.useContext();
  const indices = keys
    .map((k) => value.findIndex((v) => v.key === k))
    .filter((i) => i !== -1);
  const handleRemove = () => {
    remove(indices);
    onSelect([], -1);
  };
  const handleDuplicate = () => onDuplicate?.(indices);
  const handleDisable = () =>
    value.forEach((_, i) => {
      if (!indices.includes(i)) return;
      methods.set(`${path}.${i}.enabled`, false);
    });
  const handleEnable = () =>
    value.forEach((_, i) => {
      if (!indices.includes(i)) return;
      methods.set(`${path}.${i}.enabled`, true);
    });
  const handleTare = () => onTare?.(indices);
  const allowDisable = indices.some((i) => value[i].enabled);
  const allowEnable = indices.some((i) => !value[i].enabled);
  return (
    <PMenu.Menu
      onChange={{
        remove: handleRemove,
        duplicate: handleDuplicate,
        disable: handleDisable,
        enable: handleEnable,
        tare: handleTare,
      }}
      level="small"
    >
      {!snapshot && indices.length > 0 && (
        <>
          <PMenu.Item itemKey="remove" startIcon={<Icon.Close />}>
            Remove
          </PMenu.Item>
          {onDuplicate != null && (
            <PMenu.Item itemKey="duplicate" startIcon={<Icon.Copy />}>
              Duplicate
            </PMenu.Item>
          )}
          <PMenu.Divider />
          {allowDisable && (
            <PMenu.Item itemKey="disable" startIcon={<Icon.Disable />}>
              Disable
            </PMenu.Item>
          )}
          {allowEnable && (
            <PMenu.Item itemKey="enable" startIcon={<Icon.Enable />}>
              Enable
            </PMenu.Item>
          )}
          {(allowEnable || allowDisable) && <PMenu.Divider />}
          {allowTare && (
            <>
              <PMenu.Item itemKey="tare" startIcon={<Icon.Tare />}>
                Tare
              </PMenu.Item>
              <PMenu.Divider />
            </>
          )}
        </>
      )}
      <Menu.HardReloadItem />
    </PMenu.Menu>
  );
};
