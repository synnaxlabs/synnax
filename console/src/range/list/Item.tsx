// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger, UnexpectedError } from "@synnaxlabs/client";
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
  Telem,
} from "@synnaxlabs/pluto";
import { type NumericTimeRange } from "@synnaxlabs/x";
import { memo, useMemo } from "react";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { FavoriteButton } from "@/range/FavoriteButton";
import { ContextMenu } from "@/range/list/ContextMenu";
import { OVERVIEW_LAYOUT } from "@/range/overview/layout";

export interface ItemProps extends List.ItemProps<ranger.Key> {
  showParent?: boolean;
  showLabels?: boolean;
  showTimeRange?: boolean;
  showFavorite?: boolean;
}

const Base = ({
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
  if (getItem == null) throw new UnexpectedError("getItem is null");
  const item = List.useItem<ranger.Key, ranger.Range>(itemKey);
  const initialValues = useMemo(() => {
    if (item == null) return undefined;
    return {
      ...item.payload,
      labels: item.labels?.map((l) => l.key) ?? [],
      parent: item.parent?.key ?? "",
      timeRange: item.timeRange.numeric,
    };
  }, [item]);
  const { form } = Ranger.useForm({
    query: {},
    initialValues,
    sync: true,
    autoSave: true,
  });
  const menuProps = Menu.useContextMenu();
  if (initialValues == null || item == null) return null;

  const { name, parent, labels, timeRange } = item;

  const handleSelect = () => placeLayout({ ...OVERVIEW_LAYOUT, name, key: itemKey });

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
          {showLabels && labels != null && labels.length > 0 && (
            <Tag.Tags variant="text">
              {labels.map(({ key, name, color }) => (
                <Tag.Tag key={key} color={color} size="small">
                  {name}
                </Tag.Tag>
              ))}
            </Tag.Tags>
          )}
          {showTimeRange && (
            <Telem.Text.TimeRange level="small">{timeRange}</Telem.Text.TimeRange>
          )}
          {showFavorite && <FavoriteButton range={item} ghost />}
        </Flex.Box>
      </Form.Form>
    </List.Item>
  );
};

export const Item = memo(Base);
