// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { context } from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactElement } from "react";

import { DefaultItem, type Items } from "@/platform/tree/item";

const [Context, useContext] = context.create<Items>({
  displayName: "Tree.Context",
  providerName: "Tree.Provider",
});

export const useItems = (): Items => useContext("Tree.useItems");

// useName resolves the display name for a resource through its registered Item, so
// callers outside the tree (search, snapshot lists) get names the same way the tree
// does. The id's type must be stable across renders (derive it from a stable key).
export const useName = (id: ontology.ID): string =>
  (useItems()[id.type] ?? DefaultItem).useName(id);

export interface ProviderProps extends PropsWithChildren {
  items: Items;
}

export const Provider = ({ items, ...rest }: ProviderProps): ReactElement => (
  <Context value={items} {...rest} />
);
