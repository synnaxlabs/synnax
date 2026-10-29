// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Access } from "@synnaxlabs/pluto";

import { Panel } from "@/platform/panel";

export interface ItemProps extends List.ItemProps<arc.Key> {
  onRename?: (name: string) => void;
  textIdPrefix?: string;
}

export const Item = ({ onRename, textIdPrefix = "text", ...props }: ItemProps) => {
  const { itemKey } = props;
  const item = List.useItem<arc.Key, arc.Arc>(itemKey);
  const canRename = Access.useUpdateGranted(arc.ontologyID(itemKey));
  const openTab = Panel.useOpenTab();

  if (item == null) return null;
  const { name } = item;
  const handleActivate = () =>
    openTab({ variant: "resource", resource: arc.ontologyID(itemKey) });

  return (
    <Select.Item<arc.Key>
      {...props}
      checkable
      onActivate={handleActivate}
      align="center"
    >
      <Text.MaybeEditable
        id={`${textIdPrefix}-${itemKey}`}
        level="p"
        value={name}
        onChange={canRename ? onRename : undefined}
        allowDoubleClick={false}
      />
    </Select.Item>
  );
};
