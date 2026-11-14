// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type icd } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { ICD } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import pluralize from "pluralize";
import { type ReactElement } from "react";

import { formatHex } from "@/feature/icd/editor/FormattedField";
import { Button as PlatformButton } from "@/platform/button";
import { ContextMenu as PlatformContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";

const uniqueName = (messages: icd.Message[]): string => {
  const names = new Set(messages.map(({ name }) => name));
  let i = messages.length + 1;
  while (names.has(`Message ${i}`)) i++;
  return `Message ${i}`;
};

const createMessage = (messages: icd.Message[]): icd.Message => ({
  key: uuid.create(),
  name: uniqueName(messages),
  payload: { format: "binary", fields: [] },
});

const FORMAT_ICONS: Record<icd.PayloadType, ReactElement> = {
  binary: <Icon.Binary />,
  text: <Icon.Text />,
  json: <Icon.JSON />,
};

const describeIdentifier = ({
  identifier,
  fields,
}: icd.BinaryPayload): string | null => {
  if (identifier == null) return null;
  switch (identifier.type) {
    case "can":
      return `CAN ${formatHex(identifier.id)}`;
    case "field": {
      const field = fields.find(({ key }) => key === identifier.field);
      return field == null ? null : `${field.name} = ${identifier.value}`;
    }
  }
};

const describeMatch = ({ payload }: icd.Message): string | null => {
  switch (payload.format) {
    case "binary":
      return describeIdentifier(payload);
    case "text":
      return payload.prefix === "" ? null : payload.prefix;
    case "json":
      return null;
  }
};

/** @returns a one-line summary of how frames match a message and what it holds. */
const describe = (message: icd.Message): string => {
  const fields = pluralize("field", message.payload.fields.length, true);
  const match = describeMatch(message);
  return match == null ? fields : `${match} · ${fields}`;
};

const Item = (props: List.ItemRenderProps<string>): ReactElement | null => {
  const path = ICD.formPath.messages[props.itemKey];
  const message = Form.useFieldValue(path, { optional: true });
  const status = Form.useStatuses(path).at(0);
  if (message == null) return null;
  return (
    <Select.Item {...props} justify="between" align="center" gap="small">
      <Flex.Box x align="center" gap="small" className={CSS.BE("icd-message", "label")}>
        {FORMAT_ICONS[message.payload.format]}
        <Flex.Box y gap="tiny" className={CSS.BE("icd-message", "text")}>
          <Text.Text weight={500} overflow="fade">
            {message.name}
          </Text.Text>
          <Text.Text level="small" color={8} overflow="fade">
            {describe(message)}
          </Text.Text>
        </Flex.Box>
      </Flex.Box>
      {status != null && <Status.Indicator variant={status.variant} />}
    </Select.Item>
  );
};

interface ContextMenuProps extends Menu.ContextMenuMenuProps {
  onRemove: (keys: string[]) => void;
}

const ContextMenu = ({ keys, onRemove }: ContextMenuProps): ReactElement => {
  const { mode } = Form.useContext();
  return (
    <PlatformContextMenu.Menu>
      {mode !== "preview" && keys.length > 0 && (
        <>
          <PlatformContextMenu.RemoveItem onClick={() => onRemove(keys)} />
          <Menu.Divider />
        </>
      )}
      <PlatformContextMenu.ReloadConsoleItem />
    </PlatformContextMenu.Menu>
  );
};

export interface MessagesProps {
  /** The keys of the selected messages. The editor shows the first. */
  selected: string[];
  onSelect: (keys: string[]) => void;
}

/**
 * Lists the messages of the ICD and adds them. Shift selects a range and Cmd toggles a
 * message. The context menu removes the selected messages.
 */
export const Messages = ({ selected, onSelect }: MessagesProps): ReactElement => {
  const messages = Form.useFieldList(ICD.formPath.messages);
  const { data, push, remove } = messages;
  const { mode } = Form.useContext();
  const menuProps = Menu.useContextMenu();
  const handleAdd = () => {
    const message = createMessage(messages.value());
    push(message);
    onSelect([message.key]);
  };
  const handleRemove = (keys: string[]) => onSelect(remove(keys).slice(0, 1));
  return (
    <Flex.Box y empty grow className={CSS.BE("icd-editor", "messages")}>
      <Menu.ContextMenu
        {...menuProps}
        menu={(p) => <ContextMenu {...p} onRemove={handleRemove} />}
      >
        <Select.Frame<string, icd.Message>
          multiple
          data={data}
          value={selected}
          onChange={onSelect}
          replaceOnSingle
          allowNone={false}
        >
          <List.Scroll
            full="y"
            className={menuProps.className}
            onContextMenu={menuProps.open}
          >
            <List.Items<string>
              emptyContent={
                <Text.Text center level="small" color={9}>
                  No messages
                </Text.Text>
              }
            >
              {({ key, ...p }) => <Item key={key} {...p} />}
            </List.Items>
          </List.Scroll>
        </Select.Frame>
      </Menu.ContextMenu>
      {mode !== "preview" && (
        <PlatformButton.CreateListItem size="small" onClick={handleAdd}>
          Add message
        </PlatformButton.CreateListItem>
      )}
    </Flex.Box>
  );
};
