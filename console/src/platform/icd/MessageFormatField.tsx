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
import { ICD as PICD } from "@synnaxlabs/pluto";
import { type deep } from "@synnaxlabs/x";
import { type ReactElement, useEffect } from "react";

import { type ReadSource } from "@/platform/icd/read";
import { Task } from "@/platform/task";

const choiceKey = (icdKey: icd.Key, message: icd.MessageKey): string =>
  `${icdKey}/${message}`;

interface Choice {
  icd: icd.ICD;
  message: icd.Message;
}

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
  const { data, getItem, more } = PICD.useList({ pageSize: 1000 });
  // The trigger names the message that the source reads before the dialog opens.
  useEffect(() => {
    more();
  }, []);
  const choices = new Map<string, Choice>(
    getItem(data).flatMap((parent) =>
      parent.messages
        .filter(({ payload }) => payload.format === "json")
        .map((m) => [choiceKey(parent.key, m.key), { icd: parent, message: m }]),
    ),
  );
  const handleChange = (key: string) => {
    const choice = choices.get(key);
    if (choice == null) throw new Error(`No message format has the key ${key}`);
    onChange(choice.icd.key, choice.message);
  };
  return (
    <Input.Item label="Message format" padHelpText={false}>
      <Select.Simple<string>
        resourceName="message format"
        value={choiceKey(icdKey, message)}
        onChange={handleChange}
        variant="floating"
        allowNone={false}
        preview={isPreview}
      >
        {[...choices].map(([key, choice]) => (
          <Select.Item key={key} itemKey={key}>
            <Text.Text>{choice.message.name}</Text.Text>
            <Text.Text color={9}>{choice.icd.name}</Text.Text>
          </Select.Item>
        ))}
      </Select.Simple>
    </Input.Item>
  );
};
