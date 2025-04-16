// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type ranger } from "@synnaxlabs/client";
import { createContext, use, useState } from "react";

import { useAsyncEffect } from "@/hooks";

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

// this one needs work
export const useName = (key: channel.Key, def: string = ""): string => {
  const { getName } = useContext();
  const [name, setName] = useState<string | undefined>(def);
  useAsyncEffect(async () => {
    // needs to listen to a channel name change
    const n = await getName(key);
    setName(n);
  }, [key, getName]);
  return name ?? def;
};

export const useAliasSetter = (): AliasSetter | null => useContext().setAlias;
