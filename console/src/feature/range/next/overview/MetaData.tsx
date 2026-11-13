// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Ranger } from "@synnaxlabs/pluto";
import { link } from "@synnaxlabs/x";
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

interface PairProps extends RangeKeyProps {
  pair: ranger.kv.Pair;
  preview: boolean;
}

const Pair = ({ rangeKey, pair, preview }: PairProps): ReactElement => {
  const { update } = Ranger.useUpdateKV();
  const { update: remove } = Ranger.useDeleteKV();
  const isLink = link.is(pair.value);
  return (
    <Field
      name={<FieldName>{pair.key}</FieldName>}
      actions={
        <>
          {isLink && (
            <Button.Button
              href={pair.value}
              target="_blank"
              autoFormatHref
              variant="text"
              size="small"
              textColor={9}
              tooltip="Open link"
            >
              <Icon.LinkExternal />
            </Button.Button>
          )}
          <Button.Copy
            text={pair.value}
            variant="text"
            size="small"
            textColor={9}
            tooltip="Copy value"
          />
          {!preview && (
            <RemoveButton onClick={() => remove({ rangeKey, key: pair.key })} />
          )}
        </>
      }
    >
      <Input.Text
        variant="shadow"
        value={pair.value}
        placeholder="Value"
        onlyChangeOnBlur
        preview={preview}
        className={CSS.BE("range-overview", "field-value")}
        textColor={isLink ? "var(--pluto-primary-z)" : undefined}
        onChange={(value) =>
          update({ rangeKey, range: rangeKey, key: pair.key, value })
        }
      />
    </Field>
  );
};

interface DraftProps extends RangeKeyProps {
  onClose: () => void;
}

const pairPath = Ranger.kvPairFormPath;

/** A new pair. It closes once the pair saves, so a failed save keeps the text. */
const PairDraft = ({ rangeKey, onClose }: DraftProps): ReactElement => {
  const { form, save } = Ranger.useCreateKVPairForm({
    query: null,
    initialValues: { key: "", value: "", range: rangeKey },
    afterSave: onClose,
  });
  const handleKeyDown = (e: KeyboardEvent) => {
    if (e.key === "Escape") onClose();
    if (e.key === "Enter") save();
  };
  return (
    <Form.Form<typeof Ranger.kvPairFormSchema> {...form}>
      <Field
        name={
          <Form.TextField
            path={pairPath.key}
            showLabel={false}
            showHelpText={false}
            inputProps={{
              autoFocus: true,
              variant: "shadow",
              placeholder: "Key",
              onKeyDown: handleKeyDown,
              className: CSS.BE("range-overview", "field-input"),
            }}
          />
        }
        actions={
          <>
            <SaveButton onClick={() => save()} />
            <RemoveButton onClick={onClose} />
          </>
        }
      >
        <Form.TextField
          path={pairPath.value}
          showLabel={false}
          showHelpText={false}
          inputProps={{
            variant: "shadow",
            placeholder: "Value",
            onKeyDown: handleKeyDown,
            className: CSS.BE("range-overview", "field-value"),
          }}
        />
      </Field>
    </Form.Form>
  );
};

const byKey = (a: ranger.kv.Pair, b: ranger.kv.Pair) => a.key.localeCompare(b.key);

interface MetaDataProps extends RangeKeyProps {
  pairs: ranger.kv.Pair[];
  adding: boolean;
  onAddingChange: (adding: boolean) => void;
  /** Shows the pairs without the controls that change them. */
  preview: boolean;
}

export const MetaData = ({
  rangeKey,
  pairs,
  adding,
  onAddingChange,
  preview,
}: MetaDataProps): ReactElement => (
  <Section
    title="Metadata"
    icon={<Icon.Details />}
    actions={
      preview ? undefined : (
        <AddButton tooltip="Add metadata" onClick={() => onAddingChange(true)} />
      )
    }
  >
    <Flex.Box y empty className={CSS.BE("range-overview", "fields")}>
      {pairs.toSorted(byKey).map((pair) => (
        <Pair key={pair.key} rangeKey={rangeKey} pair={pair} preview={preview} />
      ))}
      {adding && (
        <PairDraft rangeKey={rangeKey} onClose={() => onAddingChange(false)} />
      )}
    </Flex.Box>
  </Section>
);
