// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Flux } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import { useCallback } from "react";

import { type UseName } from "@/platform/tree/item";
import { type ContextMenuProps } from "@/platform/tree/types";
import { useConfirmDelete } from "@/platform/tree/useConfirmDelete";

export interface CreateUseDeleteArgs<K extends record.Key> {
  type: string;
  icon?: string;
  description?: string;
  query: Flux.UseUpdate<K | K[]>;
  convertKey: (key: string) => K;
  // useName resolves the name shown in the single-item delete confirmation.
  useName?: UseName;
  beforeUpdate?: (
    query: Flux.BeforeUpdateParams<K | K[]> & ContextMenuProps,
  ) => Promise<K | K[] | boolean>;
  afterSuccess?: (query: Flux.AfterSuccessParams<K | K[]> & ContextMenuProps) => void;
}

export const createUseDelete =
  <K extends record.Key>({
    type,
    icon,
    description,
    query,
    convertKey,
    useName = () => "",
    beforeUpdate,
    afterSuccess,
  }: CreateUseDeleteArgs<K>): ((props: ContextMenuProps) => () => void) =>
  (props: ContextMenuProps) => {
    const {
      selection: { ids },
    } = props;
    const confirm = useConfirmDelete({ type, description, icon });
    // Only the single-item confirmation shows a name; multi-delete uses the count.
    const name = useName(ids[0]);
    const { update } = query({
      beforeUpdate: useCallback(
        async (query: Flux.BeforeUpdateParams<K | K[]>) => {
          const res = await confirm(ids.map(() => ({ name })));
          if (!res) return false;
          if (beforeUpdate != null) return await beforeUpdate({ ...query, ...props });
          return true;
        },
        [props, confirm, name, ids],
      ),
      afterSuccess: useCallback(
        (query: Flux.AfterSuccessParams<K | K[]>) => {
          afterSuccess?.({ ...query, ...props });
        },
        [props],
      ),
    });
    return useCallback(
      () => update(ids.map((id) => convertKey(id.key))),
      [update, ids, convertKey],
    );
  };
