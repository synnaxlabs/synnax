// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, ranger } from "@synnaxlabs/client";
import {
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useMemo,
  useState,
} from "react";

import { type Aliases, Context } from "@/channel/AliasContext";
import { useAsyncEffect } from "@/hooks";
import { Synch } from "@/synch";
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
  const addListener = Synch.useAddListener();
  useAsyncEffect(async () => {
    if (client == null || activeRange == null) {
      setAliases(EMPTY_ALIASES);
      return;
    }
    const rng = await client.ranges.retrieve(activeRange);
    const newAliases = await rng.listAliases();
    setAliases(newAliases);
    return addListener({
      channels: [ranger.SET_ALIAS_CHANNEL_NAME, ranger.DELETE_ALIAS_CHANNEL_NAME],
      handler: (frame) => {
        // TODO: some type of function that takes in a set channel name and a delete
        // channel name, and a decoder and returns a FrameHandler
        const createdAliases: ranger.Alias[] = frame
          .get(ranger.SET_ALIAS_CHANNEL_NAME)
          .parseJSON(ranger.aliasZ);
        const deletedAliases: ranger.DecodedDeleteAliasChange[] = frame
          .get(ranger.DELETE_ALIAS_CHANNEL_NAME)
          .series.flatMap((s) => s.toStrings())
          .map(ranger.decodeDeleteAliasChange);
        setAliases((prevAliases) => {
          const nextAliases: Aliases = { ...prevAliases };
          deletedAliases.forEach(({ channel, range }) => {
            if (range === activeRange) delete nextAliases[channel];
          });
          createdAliases.forEach(({ alias, channel, range }) => {
            if (range === activeRange) nextAliases[channel] = alias;
          });
          return nextAliases;
        });
      },
    });
  }, [client, activeRange]);

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
