import { type arc } from "@synnaxlabs/client";
import {
  Arc,
  Flex,
  Form,
  Input,
  List,
  Menu,
  Select,
  Status,
  stopPropagation,
  Tag,
  Text,
} from "@synnaxlabs/pluto";
import { useMemo } from "react";

import { ContextMenu } from "@/arc/list/ContextMenu";

export interface ItemProps extends List.ItemProps<arc.Key> {
  showLabels?: boolean;
  showStatus?: boolean;
}

export const Item = ({ showLabels = true, showStatus = true, ...props }: ItemProps) => {
  const { itemKey } = props;
  const { getItem } = List.useUtilContext<arc.Key, arc.Arc>();
  if (getItem == null) throw new Error("getItem is null");
  const arc = List.useItem<arc.Key, arc.Arc>(itemKey);
  const { onSelect, selected, ...selectProps } = Select.useItemState(itemKey);
  const initialValues = useMemo(() => {
    if (arc == null) return null;
    return {
      key: arc.key,
      name: arc.name,
    };
  }, [arc]);

  if (initialValues == null || arc == null) return null;

  const { form } = Arc.useForm({
    params: { key: itemKey },
    initialValues,
    sync: true,
    autoSave: true,
  });
  const { name, labels, status } = arc;

  const menuProps = Menu.useContextMenu();

  return (
    <List.Item
      {...props}
      {...selectProps}
      selected={selected}
      rounded={!selected}
      onSelect={onSelect}
      onContextMenu={menuProps.open}
      justify="between"
      align="center"
    >
      <Form.Form<typeof Arc.formSchema> {...form}>
        <Menu.ContextMenu
          menu={(p) => <ContextMenu {...p} getItem={getItem} />}
          onClick={stopPropagation}
          {...menuProps}
        />
        <Flex.Box x align="center">
          <Input.Checkbox
            value={selected}
            onChange={onSelect}
            onClick={stopPropagation}
          />
          <Text.Text level="p">{name}</Text.Text>
        </Flex.Box>
        <Flex.Box x align="center">
          <Tag.Tags variant="text">
            {showLabels &&
              labels?.map((l) => (
                <Tag.Tag key={l.key} color={l.color} size="small">
                  {l.name}
                </Tag.Tag>
              ))}
          </Tag.Tags>
          {showStatus && status != null && (
            <Status.Summary
              variant={status.variant}
              key={status.key}
              message={status.message}
            />
          )}
        </Flex.Box>
      </Form.Form>
    </List.Item>
  );
};
