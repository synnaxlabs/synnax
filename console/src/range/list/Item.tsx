// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/range/list/Item.css";

import { type ranger } from "@synnaxlabs/client";
import {
  Flex,
  Form,
  Input,
  List,
  Menu,
  Ranger,
  Select,
  stopPropagation,
  Tag,
} from "@synnaxlabs/pluto";
import { type NumericTimeRange } from "@synnaxlabs/x";
import { useMemo } from "react";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { OVERVIEW_LAYOUT } from "@/range/external";
import { FavoriteButton } from "@/range/FavoriteButton";
import { ContextMenu } from "@/range/list/ContextMenu";

export interface ItemProps extends List.ItemProps<ranger.Key> {
  showParent?: boolean;
  showLabels?: boolean;
  showTimeRange?: boolean;
  showFavorite?: boolean;
}

export const Item = ({
  showParent = true,
  showLabels = true,
  showTimeRange = true,
  showFavorite = true,
  ...props
}: ItemProps) => {
  const { itemKey } = props;
  const { onSelect, selected, ...selectProps } = Select.useItemState(itemKey);
  const placeLayout = Layout.usePlacer();
  const { getItem } = List.useUtilContext<ranger.Key, ranger.Range>();
  if (getItem == null) throw new Error("getItem is null");
  const item = List.useItem<ranger.Key, ranger.Range>(itemKey);
  const initialValues = useMemo(() => {
    if (item == null) return null;
    return {
      ...item.payload,
      labels: item.labels?.map((l) => l.key) ?? [],
      parent: item.parent?.key ?? "",
      timeRange: item.timeRange.numeric,
    };
  }, [item]);
  if (initialValues == null || item == null) return null;

  const { name, parent, labels, timeRange } = item;

  const { form } = Ranger.useForm({
    params: {},
    initialValues,
    sync: true,
    autoSave: true,
  });

  const handleSelect = () => placeLayout({ ...OVERVIEW_LAYOUT, name, key: itemKey });

  const menuProps = Menu.useContextMenu();
  return (
    <List.Item
      className={CSS(CSS.BE("range", "list-item"))}
      onSelect={handleSelect}
      justify="between"
      onContextMenu={menuProps.open}
      selected={selected}
      rounded={!selected}
      {...selectProps}
      {...props}
    >
      <Form.Form<typeof Ranger.formSchema> {...form}>
        <Menu.ContextMenu
          menu={(p) => <ContextMenu {...p} getItem={getItem} />}
          onClick={stopPropagation}
          {...menuProps}
        />
        <Flex.Box x empty>
          <Input.Checkbox
            value={selected}
            onChange={onSelect}
            onClick={stopPropagation}
            size="medium"
            variant="text"
            ghost={!selected}
          />
          <Flex.Box x align="center" gap="tiny">
            <Form.Field<NumericTimeRange>
              path="timeRange"
              showHelpText
              showLabel={false}
            >
              {({ value, onChange }) => (
                <Ranger.SelectStage
                  {...Ranger.wrapNumericTimeRangeToStage({ value, onChange })}
                  variant="floating"
                  location="bottom"
                  onClick={stopPropagation}
                  triggerProps={{ variant: "text", iconOnly: true }}
                />
              )}
            </Form.Field>
            <Ranger.Breadcrumb
              name={name}
              parent={parent}
              showParent={showParent}
              overflow="nowrap"
            />
          </Flex.Box>
        </Flex.Box>
        <Flex.Box x align="center">
          <Tag.Tags variant="text">
            {showLabels &&
              labels?.map(({ key, name, color }) => (
                <Tag.Tag key={key} color={color} size="small">
                  {name}
                </Tag.Tag>
              ))}
          </Tag.Tags>
          {showTimeRange && (
            <Ranger.TimeRangeChip level="small" timeRange={timeRange} />
          )}
          {showFavorite && <FavoriteButton range={item} ghost />}
        </Flex.Box>
      </Form.Form>
    </List.Item>
  );
};
