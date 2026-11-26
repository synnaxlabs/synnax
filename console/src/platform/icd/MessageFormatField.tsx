// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { icd } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Divider } from "@synnaxlabs/lyra/divider";
import { Flex } from "@synnaxlabs/lyra/flex";
import { useInitializerRef } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { useMemoDeepEqual } from "@synnaxlabs/lyra/memo";
import { Select } from "@synnaxlabs/lyra/select";
import { Store } from "@synnaxlabs/lyra/store";
import { Text } from "@synnaxlabs/lyra/text";
import { ICD as PICD } from "@synnaxlabs/pluto";
import type { destructor, record } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useRef } from "react";

import { OpenInEditor } from "@/platform/icd/OpenInEditor";

/** A JSON message of an ICD, as one option of the field. */
interface Choice extends record.KeyedNamed<string> {
  icdName: string;
}

const NONE = "none";

const choiceKey = (icdKey: icd.Key, message: icd.MessageKey): string =>
  `${icdKey}/${message}`;

const parseChoiceKey = (key: string): [icd.Key, icd.MessageKey] => {
  const [icdKey, message] = key.split("/");
  return [icdKey, message];
};

const ChoiceItem = (props: List.ItemRenderProps<string>): ReactElement => {
  const choice = List.useItem<string, Choice>(props.itemKey);
  return (
    <Select.Item {...props}>
      <Text.Text>{choice?.name}</Text.Text>
      <Text.Text color={9}>{choice?.icdName}</Text.Text>
    </Select.Item>
  );
};

const choiceRenderProp = Component.renderProp(ChoiceItem);

export interface MessageFormatFieldProps extends Pick<
  Input.ItemProps,
  "helpText" | "status" | "padHelpText"
> {
  /** The ICD of the picked message. Absent before a pick. */
  icdKey?: icd.Key;
  /** The key of the picked message. Absent before a pick. */
  message?: icd.MessageKey;
  onChange: (icdKey: icd.Key, message: icd.Message) => void;
  /** Empties the pick. When given, the field offers a pinned "No message format". */
  onClear?: () => void;
  disabled?: boolean;
  /** Shows the pick read only. */
  preview?: boolean;
}

/**
 * Picks a message from the JSON messages of every ICD, beside a button that opens the
 * editor at it. A search matches ICD names on the Core and lists every JSON message of
 * each match.
 */
export const MessageFormatField = ({
  icdKey,
  message,
  onChange,
  onClear,
  disabled = false,
  preview = false,
  ...rest
}: MessageFormatFieldProps): ReactElement => {
  const icds = PICD.useList();
  // List readers need one choice object for as long as its ICD stays the same.
  const cache = useInitializerRef(() => new WeakMap<icd.ICD, Map<string, Choice>>());
  const choicesOf = (parent: icd.ICD): Map<string, Choice> => {
    let choices = cache.current.get(parent);
    if (choices == null) {
      choices = new Map(
        parent.messages
          .filter(({ payload }) => payload.format === "json")
          .map(({ key, name }) => {
            const itemKey = choiceKey(parent.key, key);
            return [itemKey, { key: itemKey, name, icdName: parent.name }];
          }),
      );
      cache.current.set(parent, choices);
    }
    return choices;
  };
  const getChoice = (key: string): Choice | undefined => {
    const parent = icds.getItem(parseChoiceKey(key)[0]);
    return parent == null ? undefined : choicesOf(parent).get(key);
  };
  const getItem = List.createGetItem<string, Choice>(getChoice, (keys) =>
    keys.map(getChoice).filter((c) => c != null),
  );
  const subscribe = (callback: () => void, key: string): destructor.Destructor =>
    icds.subscribe(callback, parseChoiceKey(key)[0]);
  const parents = Store.useItems({
    keys: icds.data,
    subscribe: icds.subscribe,
    getItem: icds.getItem,
  });
  const data = useMemoDeepEqual(
    parents.flatMap((parent) => [...choicesOf(parent).keys()]),
  );
  // A page of ICDs with no JSON message adds no row, so no scroll asks for the next.
  const rowsBeforeRead = useRef<number | null>(null);
  const handleFetchMore = () => {
    rowsBeforeRead.current = data.length;
    icds.more();
  };
  const handleSearch = (term: string) => {
    rowsBeforeRead.current = 0;
    icds.search(term);
  };
  useEffect(() => {
    const before = rowsBeforeRead.current;
    if (before == null || icds.variant !== "success") return;
    rowsBeforeRead.current = null;
    if (data.length <= before && !icds.complete) handleFetchMore();
  }, [icds.variant, data]);
  const handleChange = (key: string) => {
    if (key === NONE) {
      onClear?.();
      return;
    }
    const [parent, messageKey] = parseChoiceKey(key);
    const picked = icds.getItem(parent)?.messages.find(({ key }) => key === messageKey);
    if (picked == null) throw new Error(`No message format has the key ${key}`);
    onChange(parent, picked);
  };
  return (
    <Input.Item label="Message format" {...rest}>
      <Flex.Box x align="center" gap="small">
        <Select.Single<string, Choice>
          resourceName="message format"
          value={icdKey == null || message == null ? "" : choiceKey(icdKey, message)}
          onChange={handleChange}
          data={data}
          getItem={getItem}
          subscribe={subscribe}
          onFetchMore={handleFetchMore}
          onSearch={handleSearch}
          status={icds.status}
          allowNone={false}
          preview={preview}
          disabled={disabled}
          footer={
            onClear != null && (
              <>
                <Divider.Divider x />
                <Select.Item itemKey={NONE} textValue="No message format">
                  <Icon.None />
                  No message format
                </Select.Item>
              </>
            )
          }
        >
          {choiceRenderProp}
        </Select.Single>
        {icdKey != null && message != null && (
          <OpenInEditor icdKey={icdKey} message={message} />
        )}
      </Flex.Box>
    </Input.Item>
  );
};
