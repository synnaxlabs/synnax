// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type resource, type Synnax } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { binary } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { type ContextMenuProps, type TreeState } from "@/platform/tree/types";
import { useResource } from "@/platform/tree/useResource";

/** Retrieves the record that a tree node shows. */
export interface Retrieve {
  (client: Synnax, key: string): Promise<unknown>;
}

interface ItemProps {
  client: Synnax;
  id: resource.ID;
  state: TreeState;
  retrieve: Retrieve;
}

const Item = ({ client, id, state, retrieve }: ItemProps): ReactElement | null => {
  const resource = useResource(state, id);
  if (resource == null) return null;
  return (
    <Menu.CopyItem
      itemKey="copyData"
      text={async () => binary.JSON_CODEC.encodeString(await retrieve(client, id.key))}
      successMessage={{
        message: "Copied properties to clipboard",
        name: resource.name,
      }}
    >
      <Icon.JSON />
      Copy properties
    </Menu.CopyItem>
  );
};

export interface CopyPropertiesContextMenuItemProps extends ContextMenuProps {
  retrieve: Retrieve;
}

export const CopyPropertiesContextMenuItem = ({
  client,
  selection: { ids },
  state,
  retrieve,
}: CopyPropertiesContextMenuItemProps): ReactElement | null => {
  if (ids.length !== 1) return null;
  return <Item client={client} id={ids[0]} state={state} retrieve={retrieve} />;
};
