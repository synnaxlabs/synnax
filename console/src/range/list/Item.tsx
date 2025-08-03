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
      labels: item.labels.map((l) => l.key),
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
      allowSelect
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
          />
          <Flex.Box x align="center" gap="tiny">
            <Form.Field<ranger.Stage> path="stage" showHelpText showLabel={false}>
              {({ value, onChange }) => (
                <Ranger.SelectStage
                  value={value}
                  onChange={onChange}
                  variant="floating"
                  location="bottom"
                  onClick={stopPropagation}
                  triggerProps={{ variant: "text" }}
                />
              )}
            </Form.Field>
            <Ranger.Breadcrumb
              name={name}
              parent={parent}
              showParent={showParent}
              noWrap
            />
          </Flex.Box>
        </Flex.Box>
        <Flex.Box x align="center">
          <Tag.Tags>
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
          {showFavorite && <FavoriteButton range={item} />}
        </Flex.Box>
      </Form.Form>
    </List.Item>
  );
};
