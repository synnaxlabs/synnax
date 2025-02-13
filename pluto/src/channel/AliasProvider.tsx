// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type ranger } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  use,
  useCallback,
  useState,
} from "react";

import { Button } from "@/button";
import { useAsyncEffect } from "@/hooks";
import { Input } from "@/input";
import { Synnax } from "@/synnax";
import { Text } from "@/text";

export interface AliasContextValue {
  aliases: Record<channel.Key, string>;
  getName: (key: channel.Key) => Promise<string | undefined>;
  setAlias: ((key: channel.Key, alias: string) => Promise<void>) | null;
  activeRange?: string | null;
}

const Context = createContext<AliasContextValue>({
  aliases: {},
  getName: async () => await Promise.resolve(undefined),
  setAlias: null,
});

export const useContext = () => use(Context);

export const useAlias = (key: channel.Key): string | null =>
  useContext().aliases[key] ?? null;

export const useAliases = (): Record<channel.Key, string> => useContext().aliases;

export const useName = (key: channel.Key, def: string = ""): string => {
  const { getName } = useContext();
  const [name, setName] = useState<string | undefined>(def);
  useAsyncEffect(async () => {
    const n = await getName(key);
    setName(n);
  }, [key, getName]);
  return name ?? def;
};

export const useActiveRange = () => useContext().activeRange ?? undefined;

export interface AliasProviderProps extends PropsWithChildren {
  activeRange?: string | null;
}

export const AliasProvider = ({
  activeRange,
  children,
}: AliasProviderProps): ReactElement => {
  const c = Synnax.use();
  const [aliases, setAliases] = useState<Record<channel.Key, string>>({});

  const handleAliasChange = useCallback(
    (changes: ranger.AliasChange[]) => {
      setAliases((aliases) => {
        const newAliases = { ...aliases };
        changes.forEach(({ variant, key, value }) => {
          const channelKey = Number(key.split("---")[1]);

          if (variant === "delete") delete newAliases[channelKey];
          else newAliases[value.channel] = value.alias;
        });
        return newAliases;
      });
    },
    [setAliases],
  );

  const setAlias = useCallback(
    async (key: channel.Key, alias: string) => {
      if (c == null || activeRange == null) return;
      const r = await c.ranges.retrieve(activeRange);
      await r.setAlias(key, alias);
    },
    [c, activeRange],
  );

  const getName = useCallback(
    async (key: channel.Key): Promise<string | undefined> => {
      if (c == null || key === 0) return undefined;
      const alias = aliases[key];
      if (alias != null) return alias;
      return (await c.channels.retrieve(key)).name;
    },
    [aliases, c],
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
  }, [c, activeRange]);

  return (
    <Context
      value={{
        aliases,
        activeRange,
        setAlias: activeRange != null ? setAlias : null,
        getName,
      }}
    >
      {children}
    </Context>
  );
};

export interface AliasInputProps extends Input.TextProps {
  channelKey: channel.Key;
  shadow?: boolean;
}

export const AliasInput = ({
  channelKey,
  value,
  shadow,
  className,
  ...props
}: AliasInputProps): ReactElement => {
  const [loading, setLoading] = useState(false);
  const { setAlias } = useContext();
  const alias = useAlias(channelKey);
  const name = useName(channelKey);
  let icon = <Icon.Rename />;
  if (loading) icon = <Icon.Loading />;
  else if (alias === value) icon = <Icon.Check />;
  const canSetAlias =
    setAlias != null && !loading && alias !== value && channelKey !== 0;
  const handleSetAlias = (): void => {
    if (!canSetAlias) return;
    void (async () => {
      setLoading(true);
      await setAlias(channelKey, value);
      setLoading(false);
    })();
  };

  const handleSetValueToAlias = (): void => {
    if (alias == null) return;
    props.onChange?.(alias);
  };

  const SetAliasTooltip = (): ReactElement => {
    if (channelKey === 0)
      return (
        <Text.Text level="small">
          Select a channel to enable alias syncing with this label
        </Text.Text>
      );
    if (setAlias == null)
      return (
        <Text.Text level="small">
          Select a range to enable alias syncing with this label
        </Text.Text>
      );
    if (value.length === 0)
      return (
        <Text.Text level="small">
          Enter a value to enable alias syncing with this label
        </Text.Text>
      );
    if (alias === value)
      return <Text.Text level="small">Alias synced with this label</Text.Text>;
    return <Text.Text level="small">Sync alias for {name} with this label</Text.Text>;
  };

  return (
    <Input.Text value={value} {...props}>
      {canSetAlias && (
        <Button.Icon
          onClick={handleSetValueToAlias}
          tooltip={<Text.Text level="small">Set {name} as label</Text.Text>}
          tooltipLocation={{ y: "top" }}
          variant="outlined"
        >
          <Icon.Sync />
        </Button.Icon>
      )}
      <Button.Icon
        onClick={handleSetAlias}
        disabled={!canSetAlias}
        tooltip={<SetAliasTooltip />}
        tooltipLocation={{ y: "top" }}
        variant="outlined"
      >
        {icon}
      </Button.Icon>
    </Input.Text>
  );
};
