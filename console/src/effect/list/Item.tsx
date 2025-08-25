import { type effect } from "@synnaxlabs/client";
import {
  Effect,
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

import { ContextMenu } from "@/effect/list/ContextMenu";

export interface ItemProps extends List.ItemProps<effect.Key> {
  showLabels?: boolean;
  showStatus?: boolean;
}

export const Item = ({ showLabels = true, showStatus = true, ...props }: ItemProps) => {
  const { itemKey } = props;
  const { getItem } = List.useUtilContext<effect.Key, effect.Effect>();
  if (getItem == null) throw new Error("getItem is null");
  const effect = List.useItem<effect.Key, effect.Effect>(itemKey);
  const { onSelect, selected, ...selectProps } = Select.useItemState(itemKey);
  const initialValues = useMemo(() => {
    if (effect == null) return null;
    return {
      key: effect.key,
      name: effect.name,
      enabled: effect.enabled,
      slate: effect.slate,
      labels: effect.labels?.map((l) => l.key) ?? [],
    };
  }, [effect]);

  if (initialValues == null || effect == null) return null;

  const { form } = Effect.useForm({
    params: { key: itemKey },
    initialValues,
    sync: true,
    autoSave: true,
  });
  const { name, labels, status } = effect;

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
      <Form.Form<typeof Effect.formSchema> {...form}>
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
