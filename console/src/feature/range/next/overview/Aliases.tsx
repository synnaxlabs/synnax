// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, query, ranger } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Text } from "@synnaxlabs/lyra/text";
import { Channel, Synnax } from "@synnaxlabs/pluto";
import { type KeyboardEvent, type ReactElement, useEffect, useState } from "react";

import {
  AddButton,
  Field,
  FieldName,
  RemoveButton,
  SaveButton,
  Section,
} from "@/feature/range/next/overview/Section";
import { CSS } from "@/platform/css";

interface RangeKeyProps {
  rangeKey: ranger.Key;
}

interface DraftProps extends RangeKeyProps {
  onClose: () => void;
}

const NO_ALIASES: Record<channel.Key, string> = {};

interface AliasList {
  aliases: Record<channel.Key, string>;
  /** The channels whose alias comes from an ancestor of the range. */
  inherited: Set<channel.Key>;
}

/** @returns the channel aliases of the range, or null until the Core answers. */
export const useAliases = (
  rangeKey: ranger.Key,
  parent?: ranger.Key,
): AliasList | null => {
  const client = Synnax.use();
  const [list, setList] = useState<AliasList | null>(null);
  useEffect(() => {
    if (client == null) return;
    let live = true;
    const load = () => {
      Promise.all([
        client.ranges.listAliases(rangeKey),
        parent == null ? NO_ALIASES : client.ranges.listAliases(parent),
      ]).then(([aliases, fromParent]) => {
        if (!live) return;
        const inherited = new Set(
          Object.keys(aliases)
            .map(Number)
            .filter((ch) => fromParent[ch] === aliases[ch]),
        );
        setList({ aliases, inherited });
      }, console.error);
    };
    load();
    const stop = client.ranges.aliases.subscribeBatch((events) => {
      const touched = events.some((e) => {
        if (!query.isChange(e)) return false;
        const { range } = ranger.alias.decodeDeleteChange(e.key);
        return range === rangeKey || range === parent;
      });
      if (touched) load();
    });
    return () => {
      live = false;
      stop();
    };
  }, [client, rangeKey, parent]);
  return list;
};

interface AliasProps extends RangeKeyProps {
  channel: channel.Key;
  alias: string;
  /** An inherited alias cannot be removed here. Changing it sets the range's own. */
  inherited: boolean;
}

const Alias = ({ rangeKey, channel, alias, inherited }: AliasProps): ReactElement => {
  const name = Channel.useResultName({ key: channel }).data;
  const { update } = Channel.useUpdateAlias();
  const { update: remove } = Channel.useDeleteAlias();
  const input = (
    <Input.Text
      variant="shadow"
      value={alias}
      placeholder="Alias"
      onlyChangeOnBlur
      resetOnBlurIfEmpty
      className={CSS.BE("range-overview", "field-value")}
      onChange={(next) => update({ range: rangeKey, channel, alias: next })}
    />
  );
  if (inherited)
    return (
      <Field
        name={<FieldName>{name}</FieldName>}
        note={
          <Text.Text
            level="small"
            color={8}
            className={CSS.BE("range-overview", "field-note")}
          >
            Inherited
          </Text.Text>
        }
      >
        {input}
      </Field>
    );
  return (
    <Field
      name={<FieldName>{name}</FieldName>}
      actions={
        <RemoveButton onClick={() => remove({ range: rangeKey, channels: channel })} />
      }
    >
      {input}
    </Field>
  );
};

/** A new alias. It closes once the alias saves, so a failed save keeps the text. */
const AliasDraft = ({ rangeKey, onClose }: DraftProps): ReactElement => {
  const [channel, setChannel] = useState<channel.Key | undefined>(undefined);
  const [alias, setAlias] = useState("");
  const { update } = Channel.useUpdateAlias({ afterSuccess: onClose });
  const save = () => {
    if (channel == null || alias.trim() === "") return;
    update({ range: rangeKey, channel, alias: alias.trim() });
  };
  const handleKeyDown = (e: KeyboardEvent) => {
    if (e.key === "Escape") onClose();
    if (e.key === "Enter") save();
  };
  return (
    <Field
      name={
        <Channel.SelectSingle
          value={channel}
          onChange={setChannel}
          triggerProps={{ variant: "shadow", placeholder: "Channel" }}
          dialogProps={{ className: CSS.BE("range-overview", "channel-dialog") }}
          className={CSS.BE("range-overview", "field-input")}
        />
      }
      actions={
        <>
          <SaveButton onClick={save} />
          <RemoveButton onClick={onClose} />
        </>
      }
    >
      <Input.Text
        variant="shadow"
        placeholder="Alias"
        value={alias}
        onChange={setAlias}
        onKeyDown={handleKeyDown}
        className={CSS.BE("range-overview", "field-value")}
      />
    </Field>
  );
};

interface AliasesProps extends RangeKeyProps, AliasList {
  adding: boolean;
  onAddingChange: (adding: boolean) => void;
}

export const Aliases = ({
  rangeKey,
  aliases,
  inherited,
  adding,
  onAddingChange,
}: AliasesProps): ReactElement => (
  <Section
    title="Channel aliases"
    icon={<Icon.Channel />}
    actions={<AddButton tooltip="Add alias" onClick={() => onAddingChange(true)} />}
  >
    <Flex.Box y empty className={CSS.BE("range-overview", "fields")}>
      {Object.entries(aliases)
        .toSorted(([, a], [, b]) => a.localeCompare(b))
        .map(([key, alias]) => (
          <Alias
            key={key}
            rangeKey={rangeKey}
            channel={Number(key)}
            alias={alias}
            inherited={inherited.has(Number(key))}
          />
        ))}
      {adding && (
        <AliasDraft rangeKey={rangeKey} onClose={() => onAddingChange(false)} />
      )}
    </Flex.Box>
  </Section>
);
