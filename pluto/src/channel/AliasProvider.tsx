// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  useState,
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  createContext,
  useContext as reactUseContext,
} from "react";

import { type channel, type ranger } from "@synnaxlabs/client";

import { useAsyncEffect } from "@/hooks";
import { Synnax } from "@/synnax";

interface AliasContextValue {
  aliases: Record<channel.Key, string>;
}

const AliasContext = createContext<AliasContextValue>({ aliases: {} });

export interface AliasProviderProps extends PropsWithChildren {
  activeRange?: string | null;
}

export const useAlias = (key: channel.Key): string | null => {
  const { aliases } = reactUseContext(AliasContext);
  return aliases[key] ?? null;
};

export const useAliases = (): Record<channel.Key, string> => {
  const { aliases } = reactUseContext(AliasContext);
  return aliases;
};

export const AliasProvider = ({ activeRange }: AliasProviderProps): ReactElement => {
  const c = Synnax.use();
  const [aliases, setAliases] = useState<Record<channel.Key, string>>({});

  const handleAliasChange = useCallback(
    (changes: ranger.AliasChange[]) => {
      setAliases((aliases) => {
        const newAliases = { ...aliases };
        changes.forEach(({ variant, key, value }) => {
          // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
          if (variant === "delete") delete newAliases[Number(key.split(":")[0])];
          else newAliases[value.channel] = value.alias;
        });
        return newAliases;
      });
    },
    [setAliases],
  );

  useAsyncEffect(async () => {
    if (c == null || activeRange == null) {
      setAliases({});
      return;
    }
    const r = await c.ranges.retrieve(activeRange);
    const aliases = await r.listAliases();
    setAliases(aliases);
    const tracker = await r.openAliasTracker();
    const disconnect = tracker.onChange(handleAliasChange);
    return () => {
      disconnect();
      void tracker.close();
    };
  });

  return <AliasContext.Provider value={{ aliases }} />;
};
