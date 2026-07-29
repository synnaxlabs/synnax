// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/status/list/Item.css";

import { type status } from "@synnaxlabs/client";
import {
  Flex,
  Form,
  Icon,
  Input,
  List,
  Select,
  Status,
  stopPropagation,
  Tag,
  Telem,
  Text,
} from "@synnaxlabs/pluto";
import { type ReactElement, useMemo } from "react";

import { Button } from "@/platform/button";
import { CSS } from "@/platform/css";
import { Session } from "@/session";

export interface ItemProps extends List.ItemProps<status.Key> {}

export const Item = (props: ItemProps): ReactElement | null => {
  const dispatch = Session.useDispatch();
  const { itemKey } = props;
  const isFavorite = Session.Status.useSelectIsFavorite(itemKey);
  const handleFavorite = () => {
    dispatch(Session.Status.toggleFavorite(itemKey));
  };
  const item = List.useItem<status.Key, status.Status>(itemKey);
  const initialValues = useMemo(() => {
    if (item == null) return undefined;
    return {
      ...item,
      labels: item.labels?.map((l) => l.key) ?? [],
    };
  }, [item]);
  const { form } = Status.useForm({
    query: {},
    initialValues,
    autoSave: true,
    sync: true,
  });
  const { selected, onSelect } = Select.useItemState(itemKey);

  if (item == null) return null;
  const { name, time, variant, message, labels } = item;

  return (
    <List.Item<status.Key>
      {...props}
      className={CSS(CSS.BE("status", "list-item"))}
      justify="between"
      selected={selected}
      rounded={!selected}
    >
      <Form.Form<typeof Status.formSchema> {...form}>
        <Flex.Box x empty>
          <Input.Checkbox
            value={selected}
            onChange={onSelect}
            size="medium"
            onClick={stopPropagation}
            ghost={!selected}
          />
          <Text.Text level="p" weight={450}>
            <Status.Indicator variant={variant} />
            <Text.Text el="span" status={variant}>
              {name}
            </Text.Text>
            {message.length > 0 && <Icon.Caret.Right />}
            <Text.Text el="span" color={9}>
              {message}
            </Text.Text>
          </Text.Text>
        </Flex.Box>
        <Flex.Box x align="center">
          {labels != null && labels.length > 0 && (
            <Tag.Tags variant="text">
              {labels.map(({ key, name, color }) => (
                <Tag.Tag key={key} color={color} size="small">
                  {name}
                </Tag.Tag>
              ))}
            </Tag.Tags>
          )}
          <Text.Text x>
            <Telem.Text.TimeSpanSince
              el="span"
              level="p"
              color="gray"
              format="semantic"
              variant="code"
            >
              {time}
            </Telem.Text.TimeSpanSince>
            <Icon.Time color={9} />
          </Text.Text>
          <Button.Favorite isFavorite={isFavorite} onFavorite={handleFavorite} ghost />
        </Flex.Box>
      </Form.Form>
    </List.Item>
  );
};
