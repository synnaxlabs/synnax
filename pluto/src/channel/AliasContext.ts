// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type ranger } from "@synnaxlabs/client";
import { createContext, use, useCallback, useState } from "react";

import { useAsyncEffect } from "@/hooks";
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

export const Context = createContext<AliasContextValue>({
  aliases: {},
  getName: () => {
    throw new Error("getName must be used within a Channel.AliasProvider component");
  },
  setAlias: () => {
    throw new Error("setAlias must be used within a Channel.AliasProvider component");
  },
});

const useContext = (): AliasContextValue => use(Context);

export const useActiveRange = (): ranger.Key | string | null | undefined =>
  useContext().activeRange;

export const useAlias = (key: channel.Key): string | null =>
  useContext().aliases[key] ?? null;

export const useAliases = (): Aliases => useContext().aliases;

export const useName = (
  key: channel.Key,
  defaultName: string = "",
): [string, (newName: string) => Promise<void>] => {
  const currentAlias = useAlias(key);
  const { getName, setAlias } = useContext();
  const [name, setName] = useState<string | undefined>(defaultName);
  const client = Synnax.use();
  useAsyncEffect(async () => {
    const n = await getName(key);
    setName(n);
  }, [key, getName]);
  const rename = useCallback(
    async (newName: string) => {
      if (client == null) return;
      const oldName = name;
      setName(newName);
      try {
        if (currentAlias != null) await setAlias?.(key, newName);
        else await client.channels.rename(key, newName);
      } catch (e) {
        setName(oldName);
        throw e;
      }
    },
    [client, name, currentAlias, setAlias],
  );
  return [name ?? defaultName, rename];
};

export const useAliasSetter = (): AliasSetter | null => useContext().setAlias;
