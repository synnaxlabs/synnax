// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, type Synnax } from "@synnaxlabs/client";
import { Flux, Menu } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";

import { type ContextMenuProps } from "@/platform/tree/types";

export interface RetrievePropertiesParams {
  client: Synnax;
  store: Flux.Store;
  id: ontology.ID;
}

export interface CopyPropertiesContextMenuItemProps extends ContextMenuProps {
  // retrieveProperties fetches the record whose JSON representation is copied to the
  // clipboard. Implementations should go through the resource type's flux retrieve so
  // cached records are reused.
  retrieveProperties: (params: RetrievePropertiesParams) => Promise<unknown>;
}

export const CopyPropertiesContextMenuItem = ({
  client,
  selection: { ids },
  state: { getName },
  retrieveProperties,
}: CopyPropertiesContextMenuItemProps): ReactElement | null => {
  const store = Flux.useStore();
  if (ids.length !== 1) return null;
  const id = ids[0];
  const name = getName(id);
  const getText = useCallback(
    async () => JSON.stringify(await retrieveProperties({ client, store, id })),
    [client, store, id, retrieveProperties],
  );
  return (
    <Menu.CopyItem
      itemKey="copyData"
      text={getText}
      successMessage={`Copied properties for ${name} to clipboard`}
    >
      Copy properties
    </Menu.CopyItem>
  );
};
