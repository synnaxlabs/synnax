import { type effect } from "@synnaxlabs/client";
import {
  Align,
  Effect,
  Form,
  Input,
  List,
  Select,
  Status,
  stopPropagation,
  Tag,
  Text,
} from "@synnaxlabs/pluto";
import { useMemo } from "react";

export interface ItemProps extends List.ItemProps<effect.Key> {
  showLabels?: boolean;
  showStatus?: boolean;
}

export const Item = ({ showLabels = true, showStatus = true, ...props }: ItemProps) => {
  const { itemKey } = props;
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

  return (
    <List.Item
      {...props}
      {...selectProps}
      selected={selected}
      rounded={!selected}
      onSelect={onSelect}
      justify="spaceBetween"
      align="center"
      allowSelect
    >
      <Form.Form<typeof Effect.formSchema> {...form}>
        <Align.Space x align="center">
          <Input.Checkbox
            value={selected}
            onChange={onSelect}
            onClick={stopPropagation}
          />
          <Text.Text level="p">{name}</Text.Text>
        </Align.Space>
        <Align.Space x align="center">
          <Tag.Tags>
            {showLabels &&
              labels?.map((l) => (
                <Tag.Tag key={l.key} color={l.color} size="small" shade={9}>
                  {l.name}
                </Tag.Tag>
              ))}
          </Tag.Tags>
          {showStatus && status != null && <Status.Text {...status} key={status.key} />}
        </Align.Space>
      </Form.Form>
    </List.Item>
  );
};
