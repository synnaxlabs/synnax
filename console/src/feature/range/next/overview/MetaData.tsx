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
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Ranger } from "@synnaxlabs/pluto";
import { link } from "@synnaxlabs/x";
import { type KeyboardEvent, type ReactElement, useState } from "react";

import {
  AddButton,
  Field,
  FieldName,
  RemoveButton,
  Section,
} from "@/feature/range/next/overview/Section";
import { CSS } from "@/platform/css";

interface RangeKeyProps {
  rangeKey: ranger.Key;
}

interface PairProps extends RangeKeyProps {
  pair: ranger.kv.Pair;
}

const Pair = ({ rangeKey, pair }: PairProps): ReactElement => {
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
            successMessage="Copied value to clipboard"
          />
          <RemoveButton onClick={() => remove({ rangeKey, key: pair.key })} />
        </>
      }
    >
      <Input.Text
        variant="shadow"
        value={pair.value}
        placeholder="Value"
        onlyChangeOnBlur
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
  onDone: () => void;
}

const PairDraft = ({ rangeKey, onDone }: DraftProps): ReactElement => {
  const [key, setKey] = useState("");
  const [value, setValue] = useState("");
  const { update } = Ranger.useUpdateKV();
  const handleKeyDown = (e: KeyboardEvent) => {
    if (e.key === "Escape") onDone();
    if (e.key !== "Enter" || key.trim() === "") return;
    update({ rangeKey, range: rangeKey, key: key.trim(), value });
    onDone();
  };
  return (
    <Field
      name={
        <Input.Text
          autoFocus
          variant="shadow"
          placeholder="Key"
          value={key}
          onChange={setKey}
          onKeyDown={handleKeyDown}
          className={CSS.BE("range-overview", "field-input")}
        />
      }
      actions={<RemoveButton onClick={onDone} />}
    >
      <Input.Text
        variant="shadow"
        placeholder="Value"
        value={value}
        onChange={setValue}
        onKeyDown={handleKeyDown}
        className={CSS.BE("range-overview", "field-value")}
      />
    </Field>
  );
};

const byKey = (a: ranger.kv.Pair, b: ranger.kv.Pair) => a.key.localeCompare(b.key);

interface MetaDataProps extends RangeKeyProps {
  pairs: ranger.kv.Pair[];
  adding: boolean;
  onAdd: (adding: boolean) => void;
}

export const MetaData = ({
  rangeKey,
  pairs,
  adding,
  onAdd,
}: MetaDataProps): ReactElement => (
  <Section
    title="Metadata"
    icon={<Icon.Details />}
    actions={<AddButton tooltip="Add metadata" onClick={() => onAdd(true)} />}
  >
    <Flex.Box y empty className={CSS.BE("range-overview", "fields")}>
      {pairs.toSorted(byKey).map((pair) => (
        <Pair key={pair.key} rangeKey={rangeKey} pair={pair} />
      ))}
      {adding && <PairDraft rangeKey={rangeKey} onDone={() => onAdd(false)} />}
    </Flex.Box>
  </Section>
);
