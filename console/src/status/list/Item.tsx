// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/status/list/Item.css";

import { type status } from "@synnaxlabs/client";
import {
  Flex,
  Form,
  Icon,
  Input,
  List,
  Menu,
  Select,
  Status,
  stopPropagation,
  Tag,
  Telem,
  Text,
} from "@synnaxlabs/pluto";
import { type ReactElement, useMemo } from "react";
import { useDispatch } from "react-redux";

import { FavoriteButton as CoreFavoriteButton } from "@/components";
import { CSS } from "@/css";
import { ContextMenu } from "@/status/list/ContextMenu";
import { useSelectIsFavorite } from "@/status/selectors";
import { toggleFavorite } from "@/status/slice";

export interface ItemProps extends List.ItemProps<status.Key> {}

export const Item = (props: ItemProps): ReactElement | null => {
  const { itemKey } = props;
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
  const { getItem } = List.useUtilContext<status.Key, status.Status>();
  if (getItem == null) throw new Error("getItem is null");
  const menuProps = Menu.useContextMenu();

  if (item == null) return null;
  const { name, time, variant, message, labels } = item;

  return (
    <List.Item<status.Key>
      {...props}
      className={CSS(CSS.BE("status", "list-item"))}
      justify="between"
      selected={selected}
      rounded={!selected}
      onContextMenu={menuProps.open}
    >
      <Form.Form<typeof Status.formSchema> {...form}>
        <Menu.ContextMenu
          menu={(p) => <ContextMenu {...p} getItem={getItem} />}
          onClick={stopPropagation}
          {...menuProps}
        />
        <Flex.Box x empty>
          <Input.Checkbox
            value={selected}
            onChange={onSelect}
            size="medium"
            onClick={stopPropagation}
            ghost={!selected}
          />
          <Text.Text level="p" weight={450} status={variant}>
            <Status.Indicator variant={variant} />
            {name}
            {message.length > 0 && <Icon.Caret.Right />}
            <Text.Text el="span" status={variant}>
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
            <Icon.Time color={8} />
          </Text.Text>
          <FavoriteButton statusKey={itemKey} />
        </Flex.Box>
      </Form.Form>
    </List.Item>
  );
};

interface FavoriteButtonProps {
  statusKey: status.Key;
}

const FavoriteButton = ({ statusKey }: FavoriteButtonProps) => {
  const dispatch = useDispatch();
  const isFavorite = useSelectIsFavorite(statusKey);
  const handleFavorite = () => {
    dispatch(toggleFavorite(statusKey));
  };
  return <CoreFavoriteButton isFavorite={isFavorite} onFavorite={handleFavorite} />;
};
