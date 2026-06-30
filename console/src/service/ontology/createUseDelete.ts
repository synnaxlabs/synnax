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

import { Ontology } from "@/ontology";

export interface CreateUseDeleteArgs<K extends record.Key> {
  type: string;
  icon?: string;
  description?: string;
  query: Flux.UseUpdate<K | K[]>;
  convertKey: (key: string) => K;
  beforeUpdate?: (
    query: Flux.BeforeUpdateParams<K | K[]> & Ontology.TreeContextMenuProps,
  ) => Promise<K | K[] | boolean>;
  afterSuccess?: (
    query: Flux.AfterSuccessParams<K | K[]> & Ontology.TreeContextMenuProps,
  ) => void;
}

export const createUseDelete =
  <K extends record.Key>({
    type,
    icon,
    description,
    query,
    convertKey,
    beforeUpdate,
    afterSuccess,
  }: CreateUseDeleteArgs<K>): ((props: Ontology.TreeContextMenuProps) => () => void) =>
  (props: Ontology.TreeContextMenuProps) => {
    const {
      selection: { ids },
      state: { getResource },
    } = props;
    const confirm = Ontology.useConfirmDelete({ type, description, icon });
    const { update } = query({
      beforeUpdate: useCallback(
        async (query: Flux.BeforeUpdateParams<K | K[]>) => {
          const res = await confirm(getResource(ids));
          if (!res) return false;
          if (beforeUpdate != null) return await beforeUpdate({ ...query, ...props });
          return true;
        },
        [props],
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
