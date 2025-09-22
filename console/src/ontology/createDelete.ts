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
    params: Flux.BeforeUpdateParams<K | K[]> & Ontology.TreeContextMenuProps,
  ) => Promise<K | K[] | boolean>;
  afterSuccess?: (
    params: Flux.AfterSuccessParams<K | K[]> & Ontology.TreeContextMenuProps,
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
        async (params: Flux.BeforeUpdateParams<K | K[]>) => {
          const res = await confirm(getResource(ids));
          if (!res) return false;
          if (beforeUpdate != null) return await beforeUpdate({ ...params, ...props });
          return true;
        },
        [props],
      ),
      afterSuccess: useCallback(
        (params: Flux.AfterSuccessParams<K | K[]>) => {
          afterSuccess?.({ ...params, ...props });
        },
        [props],
      ),
    });
    return useCallback(
      () => update(ids.map((id) => convertKey(id.key))),
      [update, ids, convertKey],
    );
  };
