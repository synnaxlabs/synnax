// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { resource } from "@synnaxlabs/client";
import { List } from "@synnaxlabs/lyra/list";
import { Text } from "@synnaxlabs/lyra/text";
import { type Flux } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";

import { type ContextMenuProps } from "@/platform/tree/types";

export interface CreateUseRenameParams<K extends record.Key> {
  useUpdate: Flux.UseUpdate<record.KeyedNamed<K>>;
  resourceID: (key: K) => resource.ID;
  convertKey: (key: string) => K;
  beforeUpdate?: (
    query: Flux.BeforeUpdateParams<record.KeyedNamed<K>> &
      ContextMenuProps & { oldName: string },
  ) => Promise<record.KeyedNamed<K> | boolean>;
}

export const createUseRename = <K extends record.Key>({
  useUpdate,
  resourceID,
  convertKey,
  beforeUpdate,
}: CreateUseRenameParams<K>): ((props: ContextMenuProps) => () => void) => {
  const useRename = (props: ContextMenuProps) => {
    const {
      selection: {
        ids: [firstID],
      },
      state: { getResource },
    } = props;
    const { update } = useUpdate({
      beforeUpdate: async (query: Flux.BeforeUpdateParams<record.KeyedNamed<K>>) => {
        const { data } = query;
        const { key, name: oldName } = data;
        const [name, renamed] = await Text.asyncEdit(
          List.itemNameID(resource.idToString(resourceID(key))),
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
    });
    return () =>
      update({ key: convertKey(firstID.key), name: getResource(firstID).name });
  };
  return useRename;
};
