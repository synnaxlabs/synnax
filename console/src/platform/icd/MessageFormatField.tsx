// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type icd } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { type Flux, ICD as PICD } from "@synnaxlabs/pluto";
import { type deep } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useSyncExternalStore } from "react";

import { type ReadSource } from "@/platform/icd/read";
import { Task } from "@/platform/task";

const choiceKey = (icdKey: icd.Key, message: icd.MessageKey): string =>
  `${icdKey}/${message}`;

interface ChoicesProps extends Pick<
  Flux.UseListReturn<icd.Key, icd.ICD>,
  "getItem" | "subscribe"
> {
  icdKey: icd.Key;
}

/** Lists the JSON messages of an ICD, current with its edits. */
const Choices = ({ icdKey, getItem, subscribe }: ChoicesProps): ReactElement[] => {
  const parent = useSyncExternalStore(
    (callback) => subscribe(callback, icdKey),
    () => getItem(icdKey),
  );
  if (parent == null) return [];
  return parent.messages
    .filter(({ payload }) => payload.format === "json")
    .map((m) => (
      <Select.Item key={m.key} itemKey={choiceKey(parent.key, m.key)}>
        <Text.Text>{m.name}</Text.Text>
        <Text.Text color={9}>{parent.name}</Text.Text>
      </Select.Item>
    ));
};

export interface MessageFormatFieldProps {
  /** The path of a source that reads one message. */
  path: deep.Path<ReadSource>;
  onChange: (icdKey: icd.Key, message: icd.Message) => void;
}

/** Picks the message that a source reads from the JSON messages of every ICD. */
export const MessageFormatField = ({
  path,
  onChange,
}: MessageFormatFieldProps): ReactElement => {
  const isPreview = Task.useIsPreview();
  const icdKey = Form.useFieldValue(path.icd);
  const { message } = Form.useFieldValue(path.messages[0]);
  const { data, getItem, subscribe, more } = PICD.useList({ pageSize: 1000 });
  // The trigger names the message that the source reads before the dialog opens.
  useEffect(() => {
    more();
  }, []);
  const handleChange = (key: string) => {
    const [parent, messageKey] = key.split("/");
    const picked = getItem(parent)?.messages.find(({ key }) => key === messageKey);
    if (picked == null) throw new Error(`No message format has the key ${key}`);
    onChange(parent, picked);
  };
  return (
    <Input.Item label="Message format" padHelpText={false}>
      <Select.Simple<string>
        resourceName="message format"
        value={choiceKey(icdKey, message)}
        onChange={handleChange}
        allowNone={false}
        preview={isPreview}
      >
        {data.map((key) => (
          <Choices key={key} icdKey={key} getItem={getItem} subscribe={subscribe} />
        ))}
      </Select.Simple>
    </Input.Item>
  );
};
