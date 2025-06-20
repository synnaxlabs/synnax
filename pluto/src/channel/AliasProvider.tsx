// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type ranger } from "@synnaxlabs/client";
import {
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useMemo,
  useState,
} from "react";

import { type Aliases, Context } from "@/channel/AliasContext";
import { useAsyncEffect } from "@/hooks";
import { Ranger } from "@/ranger";
import { Synnax } from "@/synnax";

const EMPTY_ALIASES: Aliases = {};

export interface AliasProviderProps extends PropsWithChildren {
  activeRange?: string | null;
}

export const AliasProvider = ({
  activeRange,
  ...rest
}: AliasProviderProps): ReactElement => {
  const client = Synnax.use();
  const [aliases, setAliases] = useState<Aliases>(EMPTY_ALIASES);
  useAsyncEffect(
    async (signal) => {
      if (client == null || activeRange == null) {
        setAliases(EMPTY_ALIASES);
        return;
      }
      const rng = await client.ranges.retrieve(activeRange);
      if (signal.aborted) return;
      const newAliases = await rng.listAliases();
      if (signal.aborted) return;
      setAliases(newAliases);
    },
    [client, activeRange],
  );

  const handleAliasSet = useCallback(
    ({ alias, channel, range }: ranger.Alias) =>
      setAliases((prevAliases) => {
        if (range !== activeRange) return prevAliases;
        const nextAliases = { ...prevAliases };
        nextAliases[channel] = alias;
        return nextAliases;
      }),
    [activeRange],
  );
  Ranger.useAliasSetSynchronizer(handleAliasSet);

  const handleAliasDelete = useCallback(
    ({ channel, range }: ranger.DecodedDeleteAliasChange) =>
      setAliases((prevAliases) => {
        if (range !== activeRange) return prevAliases;
        const nextAliases = { ...prevAliases };
        delete nextAliases[channel];
        return nextAliases;
      }),
    [activeRange],
  );
  Ranger.useAliasDeleteSynchronizer(handleAliasDelete);

  const setAlias = useCallback(
    async (key: channel.Key, alias: string) => {
      if (client == null || activeRange == null) return;
      const r = await client.ranges.retrieve(activeRange);
      await r.setAlias(key, alias);
    },
    [client, activeRange],
  );

  const getName = useCallback(
    async (key: channel.Key): Promise<string | undefined> => {
      if (client == null || key === 0) return undefined;
      const alias = aliases[key];
      if (alias != null) return alias;
      const { name } = await client.channels.retrieve(key);
      return name;
    },
    [aliases, client],
  );

  const value = useMemo(
    () => ({
      aliases,
      activeRange,
      setAlias: activeRange != null ? setAlias : null,
      getName,
    }),
    [aliases, activeRange, setAlias, getName],
  );

  return <Context {...rest} value={value} />;
};
