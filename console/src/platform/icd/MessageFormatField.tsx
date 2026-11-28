// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/icd/MessageFormatField.css";

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
import type { destructor, payload, record } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useRef } from "react";

import { CSS } from "@/platform/css";
import { useFormats } from "@/platform/icd/Formats";
import { OpenInEditor } from "@/platform/icd/OpenInEditor";
import { FORMAT_KINDS } from "@/platform/icd/payload";
import { Task } from "@/platform/task";

/** A message of an ICD, as one option of the field. */
interface Choice extends record.KeyedNamed<string> {
  icdName: string;
  format: payload.Format;
}

const NONE = "none";

const JSON_ONLY: readonly payload.Format[] = ["json"];

const CREATE_TEXTS: Record<payload.Format, string> = {
  json: "New JSON message",
  text: "New text message",
  binary: "New binary message",
};

const createKey = (format: payload.Format): string => `new/${format}`;

const choiceKey = (icdKey: icd.Key, message: icd.MessageKey): string =>
  `${icdKey}/${message}`;

const parseChoiceKey = (key: string): [icd.Key, icd.MessageKey] => {
  const [icdKey, message] = key.split("/");
  return [icdKey, message];
};

interface ChoiceItemProps extends List.ItemRenderProps<string> {
  /** Shows the format of the message before its name, as a task tree does. */
  formatShown?: boolean;
}

const ChoiceItem = ({
  formatShown = false,
  ...rest
}: ChoiceItemProps): ReactElement => {
  const choice = List.useItem<string, Choice>(rest.itemKey);
  return (
    <Select.Item {...rest}>
      {formatShown && choice != null ? (
        <Task.Views.ItemLabel kind={FORMAT_KINDS[choice.format]} kindColor={9}>
          {choice.name}
        </Task.Views.ItemLabel>
      ) : (
        <Text.Text overflow="fade">{choice?.name}</Text.Text>
      )}
      <Text.Text color={9} overflow="fade">
        {choice?.icdName}
      </Text.Text>
    </Select.Item>
  );
};

const FormatChoiceItem = (props: List.ItemRenderProps<string>): ReactElement => (
  <ChoiceItem {...props} formatShown />
);

const choiceRenderProp = Component.renderProp(ChoiceItem);
const formatChoiceRenderProp = Component.renderProp(FormatChoiceItem);

const DIALOG_PROPS = { className: CSS.BE("message-format", "dialog") };

export interface MessageFormatFieldProps extends Pick<
  Input.ItemProps,
  "helpText" | "status" | "padHelpText"
> {
  /** The ICD of the picked message. Absent before a pick. */
  icdKey?: icd.Key;
  /** The key of the picked message. Absent before a pick. */
  message?: icd.MessageKey;
  onChange: (icdKey: icd.Key, message: icd.Message) => void;
  /** The formats of the messages that the field lists and creates. Defaults to JSON. */
  formats?: readonly payload.Format[];
  /**
   * Creates an empty message of the format in the ICD of icdKey. When given, the field
   * offers a pinned create item, such as "New text message", for each format that it
   * lists, unless the ICD's form is in preview.
   */
  onCreate?: (format: payload.Format) => void;
  /** Empties the pick. When given, the field offers a pinned "No message format". */
  onClear?: () => void;
  disabled?: boolean;
  /** Shows the pick read only. */
  preview?: boolean;
}

/**
 * Picks a message from the messages of every ICD in formats, beside a button that opens
 * the editor at it. A search matches ICD names on the Core and lists every message of
 * each match in formats. The field shows a picked message of any format.
 */
export const MessageFormatField = ({
  icdKey,
  message,
  onChange,
  formats = JSON_ONLY,
  onCreate,
  onClear,
  disabled = false,
  preview = false,
  ...rest
}: MessageFormatFieldProps): ReactElement => {
  const icds = PICD.useList();
  const forms = useFormats();
  // A user who cannot update the ICD gets its form in preview.
  const editable = icdKey != null && forms.get(icdKey)?.mode === "normal";
  const creates = onCreate != null && editable ? formats : [];
  // List readers need one choice object for as long as its ICD stays the same.
  const cache = useInitializerRef(() => new WeakMap<icd.ICD, Map<string, Choice>>());
  const choicesOf = (parent: icd.ICD): Map<string, Choice> => {
    let choices = cache.current.get(parent);
    if (choices == null) {
      choices = new Map(
        parent.messages.map(({ key, name, payload: { format } }) => {
          const itemKey = choiceKey(parent.key, key);
          return [itemKey, { key: itemKey, name, icdName: parent.name, format }];
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
    parents.flatMap((parent) =>
      [...choicesOf(parent).values()]
        .filter(({ format }) => formats.includes(format))
        .map(({ key }) => key),
    ),
  );
  // A page of ICDs with no listed message adds no row, so no scroll asks for the next.
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
    const created = creates.find((format) => createKey(format) === key);
    if (created != null) {
      onCreate?.(created);
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
          dialogProps={DIALOG_PROPS}
          footer={
            (creates.length > 0 || onClear != null) && (
              <>
                <Divider.Divider x />
                {creates.map((format) => (
                  <Select.Item
                    key={format}
                    itemKey={createKey(format)}
                    textValue={CREATE_TEXTS[format]}
                  >
                    <Icon.Add />
                    {CREATE_TEXTS[format]}
                  </Select.Item>
                ))}
                {onClear != null && (
                  <Select.Item itemKey={NONE} textValue="No message format">
                    <Icon.None />
                    No message format
                  </Select.Item>
                )}
              </>
            )
          }
        >
          {formats.length > 1 ? formatChoiceRenderProp : choiceRenderProp}
        </Select.Single>
        {icdKey != null && message != null && (
          <OpenInEditor icdKey={icdKey} message={message} />
        )}
      </Flex.Box>
    </Input.Item>
  );
};
