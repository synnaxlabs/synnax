// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, query, type ranger } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Text } from "@synnaxlabs/lyra/text";
import { Channel, Ranger } from "@synnaxlabs/pluto";
import { type KeyboardEvent, type ReactElement } from "react";

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

interface AliasProps extends RangeKeyProps {
  channel: channel.Key;
  /** The name of the channel, or undefined until it loads. */
  name?: string;
  alias: string;
  /** An inherited alias cannot be removed here. Changing it sets the range's own. */
  inherited: boolean;
}

const Alias = ({
  rangeKey,
  channel,
  name,
  alias,
  inherited,
}: AliasProps): ReactElement => {
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

const aliasPath = Ranger.aliasFormPath;

/** A new alias. It closes once the alias saves, so a failed save keeps the text. */
const AliasDraft = ({ rangeKey, onClose }: DraftProps): ReactElement => {
  const { form, save } = Ranger.useCreateAliasForm({
    query: null,
    initialValues: { range: rangeKey, channel: 0, alias: "" },
    afterSave: onClose,
  });
  const handleKeyDown = (e: KeyboardEvent) => {
    if (e.key === "Escape") onClose();
    if (e.key === "Enter") save();
  };
  return (
    <Form.Form<typeof Ranger.aliasFormSchema> {...form}>
      <Field
        name={
          <Form.Field<channel.Key>
            path={aliasPath.channel}
            showLabel={false}
            showHelpText={false}
          >
            {(p) => (
              <Channel.SelectSingle
                {...p}
                triggerProps={{ variant: "shadow", placeholder: "Channel" }}
                dialogProps={{ className: CSS.BE("range-overview", "channel-dialog") }}
                className={CSS.BE("range-overview", "field-input")}
              />
            )}
          </Form.Field>
        }
        actions={
          <>
            <SaveButton onClick={() => save()} />
            <RemoveButton onClick={onClose} />
          </>
        }
      >
        <Form.TextField
          path={aliasPath.alias}
          showLabel={false}
          showHelpText={false}
          inputProps={{
            variant: "shadow",
            placeholder: "Alias",
            onKeyDown: handleKeyDown,
            className: CSS.BE("range-overview", "field-value"),
          }}
        />
      </Field>
    </Form.Form>
  );
};

interface AliasesProps extends RangeKeyProps {
  aliases: ranger.ResolvedAliases;
  adding: boolean;
  onAddingChange: (adding: boolean) => void;
}

export const Aliases = ({
  rangeKey,
  aliases: { aliases, own },
  adding,
  onAddingChange,
}: AliasesProps): ReactElement => {
  const keys = Object.keys(aliases).map(Number);
  const channels = Channel.useResultMultiple(
    keys.length === 0 ? null : query.keysRequest(keys),
  ).data;
  const names = new Map(channels?.map(({ key, name }) => [key, name]));
  return (
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
              name={names.get(Number(key))}
              alias={alias}
              inherited={!own.includes(Number(key))}
            />
          ))}
        {adding && (
          <AliasDraft rangeKey={rangeKey} onClose={() => onAddingChange(false)} />
        )}
      </Flex.Box>
    </Section>
  );
};
