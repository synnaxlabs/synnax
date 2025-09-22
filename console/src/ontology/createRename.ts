import { ontology } from "@synnaxlabs/client";
import { type Flux, Text } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import { useCallback } from "react";

import { type Ontology } from "@/ontology";

export interface CreateUseRenameArgs<K extends record.Key> {
  query: Flux.UseUpdate<record.KeyedNamed<K>>;
  ontologyID: (key: K) => ontology.ID;
  convertKey: (key: string) => K;
  beforeUpdate?: (
    params: Flux.BeforeUpdateParams<record.KeyedNamed<K>> &
      Ontology.TreeContextMenuProps & { oldName: string },
  ) => Promise<record.KeyedNamed<K> | boolean>;
}

export const createUseRename =
  <K extends record.Key>({
    query,
    ontologyID,
    convertKey,
    beforeUpdate,
  }: CreateUseRenameArgs<K>): ((props: Ontology.TreeContextMenuProps) => () => void) =>
  (props: Ontology.TreeContextMenuProps) => {
    const {
      selection: {
        ids: [firstID],
      },
      state: { getResource },
    } = props;
    const { update } = query({
      beforeUpdate: useCallback(
        async (params: Flux.BeforeUpdateParams<record.KeyedNamed<K>>) => {
          const { data } = params;
          const { key, name: oldName } = data;
          const [name, renamed] = await Text.asyncEdit(
            ontology.idToString(ontologyID(key)),
          );
          if (!renamed) return false;
          if (beforeUpdate != null)
            return await beforeUpdate({
              ...params,
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
