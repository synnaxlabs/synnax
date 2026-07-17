// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology } from "@synnaxlabs/client";
import { type Flux, List, Text } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import { useCallback } from "react";

import { type UseName } from "@/platform/tree/item";
import { type ContextMenuProps } from "@/platform/tree/types";

export interface CreateUseRenameArgs<K extends record.Key> {
  query: Flux.UseUpdate<record.KeyedNamed<K>>;
  ontologyID: (key: K) => ontology.ID;
  convertKey: (key: string) => K;
  // useName resolves the current name, seeded as oldName for beforeUpdate. Only needed
  // by types whose beforeUpdate rolls back to the old name.
  useName?: UseName;
  beforeUpdate?: (
    query: Flux.BeforeUpdateParams<record.KeyedNamed<K>> &
      ContextMenuProps & { oldName: string },
  ) => Promise<record.KeyedNamed<K> | boolean>;
}

export const createUseRename =
  <K extends record.Key>({
    query,
    ontologyID,
    convertKey,
    useName = () => "",
    beforeUpdate,
  }: CreateUseRenameArgs<K>): ((props: ContextMenuProps) => () => void) =>
  (props: ContextMenuProps) => {
    const {
      selection: {
        ids: [firstID],
      },
    } = props;
    const oldName = useName(firstID);
    const { update } = query({
      beforeUpdate: useCallback(
        async (query: Flux.BeforeUpdateParams<record.KeyedNamed<K>>) => {
          const { data } = query;
          const { key, name: oldName } = data;
          const [name, renamed] = await Text.asyncEdit(
            List.itemNameID(ontology.idToString(ontologyID(key))),
          );
          if (!renamed) return false;
          if (beforeUpdate != null)
            return await beforeUpdate({
              ...query,
              ...props,
              oldName,
              data: { ...data, name },
            });
          return { ...data, name: name ?? "" };
        },
        [beforeUpdate, props],
      ),
    });
    return useCallback(
      // The new name is read from the inline editor in beforeUpdate; this seed only
      // carries the old name through to a type's beforeUpdate for rollback.
      () => update({ key: convertKey(firstID.key), name: oldName }),
      [firstID, update, convertKey, oldName],
    );
  };
