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

import { type ContextMenuProps } from "@/platform/tree/types";

export interface CreateUseRenameParams<K extends record.Key> {
  query: Flux.UseUpdate<record.KeyedNamed<K>>;
  ontologyID: (key: K) => ontology.ID;
  convertKey: (key: string) => K;
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
    beforeUpdate,
  }: CreateUseRenameParams<K>): ((props: ContextMenuProps) => () => void) =>
  (props: ContextMenuProps) => {
    const {
      selection: {
        ids: [firstID],
      },
      state: { getResource },
    } = props;
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
      () => update({ key: convertKey(firstID.key), name: getResource(firstID).name }),
      [firstID, getResource],
    );
  };
