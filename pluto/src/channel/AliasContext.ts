// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, DisconnectedError, ontology, type ranger } from "@synnaxlabs/client";
import { createContext, use, useCallback, useState } from "react";

import { useAsyncEffect } from "@/hooks";
import { Ontology } from "@/ontology";
import { Status } from "@/status";
import { Synnax } from "@/synnax";

export interface Aliases extends Record<channel.Key, string> {}

export interface AliasSetter {
  (key: channel.Key, alias: string): Promise<void>;
}

export interface AliasContextValue {
  activeRange?: ranger.Key | string | null;
  aliases: Aliases;
  getName: (key: channel.Key) => Promise<string | undefined>;
  setAlias: AliasSetter | null;
}

export const Context = createContext<AliasContextValue | undefined>(undefined);

const useContext = (): AliasContextValue => {
  const ctx = use(Context);
  if (ctx == null) throw new Error("Channel.AliasProvider not found");
  return ctx;
};

export const useActiveRange = (): ranger.Key | string | null | undefined =>
  useContext().activeRange;

export const useAlias = (key: channel.Key): string | null =>
  useContext().aliases[key] ?? null;

export const useAliases = (): Aliases => useContext().aliases;

export const useName = (
  key: channel.Key,
  defaultName: string = "",
): [string, (newName: string) => void] => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const handleResourceSet = useCallback(
    (id: ontology.ID) => {
      if (!ontology.idsEqual(id, channel.ontologyID(key))) return;
      if (currentAlias != null) return;
      handleError(async () => {
        const resource = await client?.ontology.retrieve(id);
        setName(resource?.name);
      }, "Failed to retrieve resource");
    },
    [client, handleError, key],
  );
  Ontology.useResourceSetSynchronizer(handleResourceSet);
  const currentAlias = useAlias(key);
  const { getName, setAlias } = useContext();
  const [name, setName] = useState<string | undefined>(defaultName);
  useAsyncEffect(
    async (signal) => {
      const n = await getName(key);
      if (signal.aborted) return;
      setName(n);
    },
    [key, getName],
  );
  const handleRename = useCallback(
    (newName: string) => {
      handleError(async () => {
        const oldName = name;
        try {
          if (currentAlias != null) {
            if (setAlias == null) throw new Error("AliasSetter not found");
            await setAlias(key, newName);
            return;
          }
          if (client == null) throw new DisconnectedError();
          await client.channels.rename(key, newName);
        } catch (e) {
          setName(oldName);
          throw e;
        }
      }, "Failed to rename channel");
    },
    [currentAlias, setAlias, key, handleError, name],
  );
  return [name ?? defaultName, handleRename];
};

export const useAliasSetter = (): AliasSetter | null => useContext().setAlias;
