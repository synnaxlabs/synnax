// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { type ReactElement } from "react";

import { type ContextMenuProps, type TreeState } from "@/platform/tree/types";
import { useResource } from "@/platform/tree/useResource";

interface ItemProps {
  id: ontology.ID;
  state: TreeState;
}

const Item = ({ id, state }: ItemProps): ReactElement | null => {
  const resource = useResource(state, id);
  if (resource == null) return null;
  const { data, name } = resource;
  return (
    <Menu.CopyItem
      itemKey="copyData"
      text={() => JSON.stringify(data)}
      successMessage={{ message: "Copied properties to clipboard", name }}
    >
      <Icon.JSON />
      Copy properties
    </Menu.CopyItem>
  );
};

export const CopyPropertiesContextMenuItem = ({
  selection: { ids },
  state,
}: ContextMenuProps): ReactElement | null => {
  if (ids.length !== 1) return null;
  return <Item id={ids[0]} state={state} />;
};
