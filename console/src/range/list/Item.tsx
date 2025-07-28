import "@/range/list/Item.css";

import { type ranger } from "@synnaxlabs/client";
import {
  Align,
  Button,
  Form,
  Icon,
  Input,
  List,
  Menu,
  Ranger,
  Select,
  stopPropagation,
  Tag,
  Text,
  Tooltip,
} from "@synnaxlabs/pluto";
import { useMemo } from "react";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { fromClientRange } from "@/range/ContextMenu";
import { OVERVIEW_LAYOUT } from "@/range/external";
import { ContextMenu } from "@/range/list/ContextMenu";
import { useSelect } from "@/range/selectors";
import { add, remove } from "@/range/slice";

export const ListItem = (props: List.ItemProps<ranger.Key>) => {
  const { itemKey } = props;
  const { onSelect, selected, ...selectProps } = Select.useItemState(itemKey);
  const placeLayout = Layout.usePlacer();
  const { getItem } = List.useUtilContext<ranger.Key, ranger.Range>();
  if (getItem == null) throw new Error("getItem is null");
  const dispatch = useDispatch();
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
  const sliceRange = useSelect(itemKey);
  const starred = sliceRange != null;
  const handleSelect = () => placeLayout({ ...OVERVIEW_LAYOUT, name, key: itemKey });
  const handleStar = () => {
    if (!starred) dispatch(add({ ranges: fromClientRange(item) }));
    else dispatch(remove({ keys: [itemKey] }));
  };
  const menuProps = Menu.useContextMenu();
  console.log(labels);
  return (
    <List.Item
      className={CSS(CSS.BE("range", "list-item"), starred && CSS.M("starred"))}
      onSelect={handleSelect}
      justify="spaceBetween"
      onContextMenu={menuProps.open}
      selected={selected}
      {...selectProps}
      {...props}
    >
      <Form.Form<typeof Ranger.rangeFormSchema> {...form}>
        <Menu.ContextMenu
          menu={(p) => <ContextMenu {...p} getItem={getItem} />}
          onClick={stopPropagation}
          {...menuProps}
        />
        <Align.Space x empty>
          <Input.Checkbox
            value={selected}
            onChange={onSelect}
            onClick={stopPropagation}
          />
          <Align.Space x align="center" gap="tiny">
            <Form.Field<ranger.Stage> path="stage" showHelpText showLabel={false}>
              {({ value, onChange }) => (
                <Ranger.SelectStage
                  value={value}
                  onChange={onChange}
                  variant="floating"
                  location="bottom"
                  onClick={stopPropagation}
                  triggerProps={{ iconOnly: true, variant: "text" }}
                />
              )}
            </Form.Field>
            <Ranger.Breadcrumb name={name} parent={parent} />
          </Align.Space>
        </Align.Space>
        <Align.Space x align="center">
          <Tag.Tags>
            {labels?.map(({ key, name, color }) => (
              <Tag.Tag key={key} color={color} size="small" shade={9}>
                {name}
              </Tag.Tag>
            ))}
          </Tag.Tags>
          <Ranger.TimeRangeChip level="small" timeRange={timeRange} />
          <Tooltip.Dialog>
            <Text.Text level="small" shade={10}>
              {starred ? "Remove from" : "Add to"} Workspace Favorites
            </Text.Text>
            <Button.Icon
              className={CSS(CSS.B("star-button"))}
              onClick={(e) => {
                e.stopPropagation();
                handleStar();
              }}
              size="small"
            >
              {sliceRange != null ? <Icon.StarFilled /> : <Icon.StarOutlined />}
            </Button.Icon>
          </Tooltip.Dialog>
        </Align.Space>
      </Form.Form>
    </List.Item>
  );
};
