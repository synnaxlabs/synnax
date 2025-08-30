import { type schematic } from "@synnaxlabs/client";
import { Button, Component, Form, type Input, List, Select } from "@synnaxlabs/pluto";

interface StateListProps extends Input.Control<string> {}

const StateListItem = (props: List.ItemRenderProps<string>) => {
  const { itemKey } = props;
  const state = Form.useFieldValue<schematic.symbol.State>(`data.states.${itemKey}`);
  const { selected, onSelect } = Select.useItemState(itemKey);
  if (state == null) return null;
  return (
    <Button.Button
      variant={selected ? "filled" : "outlined"}
      justify="center"
      onClick={onSelect}
      style={{ minWidth: 80 }}
      grow
    >
      {state.name}
    </Button.Button>
  );
};

const stateListItem = Component.renderProp(StateListItem);

export const StateList = ({ value, onChange }: StateListProps) => {
  const { data } = Form.useFieldList<string, schematic.symbol.State>("data.states");
  return (
    <Select.Frame
      value={value}
      onChange={onChange}
      data={data}
      closeDialogOnSelect={false}
    >
      <List.Items x gap={1}>
        {stateListItem}
      </List.Items>
    </Select.Frame>
  );
};
