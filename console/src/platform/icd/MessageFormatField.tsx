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
import { Form } from "@synnaxlabs/lyra/form";
import { useInitializerRef } from "@synnaxlabs/lyra/hooks";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { useMemoDeepEqual } from "@synnaxlabs/lyra/memo";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { ICD as PICD } from "@synnaxlabs/pluto";
import type { deep, destructor, record } from "@synnaxlabs/x";
import type { ReactElement } from "react";

import type { ReadSource } from "@/platform/icd/read";
import { Task } from "@/platform/task";

/** A JSON message of an ICD, as one option of the field. */
interface Choice extends record.KeyedNamed<string> {
  icdName: string;
}

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

export interface MessageFormatFieldProps {
  /** The path of a source that reads one message. */
  path: deep.Path<ReadSource>;
  onChange: (icdKey: icd.Key, message: icd.Message) => void;
  disabled?: boolean;
}

/**
 * Picks the message that a source reads from the JSON messages of every ICD. A search
 * matches ICD names on the Core and lists every JSON message of each match.
 */
export const MessageFormatField = ({
  path,
  onChange,
  disabled = false,
}: MessageFormatFieldProps): ReactElement => {
  const isPreview = Task.useIsPreview();
  const icdKey = Form.useFieldValue(path.icd);
  const { message } = Form.useFieldValue(path.messages[0]);
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
  const data = useMemoDeepEqual(
    icds.getItem(icds.data).flatMap((parent) => [...choicesOf(parent).keys()]),
  );
  const handleChange = (key: string) => {
    const [parent, messageKey] = parseChoiceKey(key);
    const picked = icds.getItem(parent)?.messages.find(({ key }) => key === messageKey);
    if (picked == null) throw new Error(`No message format has the key ${key}`);
    onChange(parent, picked);
  };
  return (
    <Input.Item label="Message format" padHelpText={false}>
      <Select.Single<string, Choice>
        resourceName="message format"
        value={choiceKey(icdKey, message)}
        onChange={handleChange}
        data={data}
        getItem={getItem}
        subscribe={subscribe}
        onFetchMore={icds.more}
        onSearch={icds.search}
        status={icds.status}
        allowNone={false}
        preview={isPreview}
        disabled={disabled}
      >
        {choiceRenderProp}
      </Select.Single>
    </Input.Item>
  );
};
