// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Button,
  Flex,
  Form,
  Header,
  Icon,
  type Input,
  List,
  Select,
  Text,
} from "@synnaxlabs/pluto";
import { location, xy } from "@synnaxlabs/x";

export const SelectHandleOrientation = (
  props: Omit<Select.ButtonsProps<location.Outer>, "keys">,
) => (
  <Select.Buttons keys={location.OUTER_LOCATIONS} size="small" {...props}>
    <Select.Button itemKey="left" size="small">
      <Icon.Arrow.Left />
    </Select.Button>
    <Select.Button itemKey="right" size="small">
      <Icon.Arrow.Right />
    </Select.Button>
    <Select.Button itemKey="top" size="small">
      <Icon.Arrow.Up />
    </Select.Button>
    <Select.Button itemKey="bottom" size="small">
      <Icon.Arrow.Down />
    </Select.Button>
  </Select.Buttons>
);

interface HandleListItemProps extends List.ItemRenderProps<string> {}

const HandleListItem = (props: HandleListItemProps) => {
  const { itemKey, index } = props;
  const path = `data.handles.${itemKey}`;
  const handle = Form.useFieldValue<{
    key: string;
    position: { x: number; y: number };
  }>(path);
  const { remove } = Form.useFieldListUtils<
    string,
    { key: string; position: { x: number; y: number } }
  >("data.handles");
  if (handle == null) return null;
  const scaledPos = xy.scale(handle.position, 100);
  return (
    <Select.ListItem {...props} justify="between" style={{ paddingRight: "0.5rem" }}>
      <Flex.Box x align="center" gap={1}>
        <Text.Text level="small" weight={500}>
          Handle {index + 1}
        </Text.Text>
        <Text.Text level="small" color={7}>
          ({Math.round(scaledPos.x)}%, {Math.round(scaledPos.y)}%)
        </Text.Text>
      </Flex.Box>
      <Flex.Box x align="center" empty>
        <Form.Field<location.Outer> path={`${path}.orientation`} showLabel={false}>
          {({ onChange, value }) => (
            <SelectHandleOrientation value={value} onChange={onChange} />
          )}
        </Form.Field>
        <Button.Button
          onClick={() => remove(itemKey)}
          size="small"
          variant="text"
          ghost
        >
          <Icon.Close />
        </Button.Button>
      </Flex.Box>
    </Select.ListItem>
  );
};

interface HandleListProps extends Input.Control<string | undefined> {
  onAddHandle: () => void;
}

export const HandleList = ({ value, onChange, onAddHandle }: HandleListProps) => {
  const { data } = Form.useFieldList<
    string,
    { key: string; position: { x: number; y: number } }
  >("data.handles");

  return (
    <Flex.Box y gap={1} style={{ maxHeight: 200 }}>
      <Header.Header level="p" bordered={false} padded>
        <Header.Title level="p" weight={500}>
          Handles
        </Header.Title>
        <Button.Button onClick={onAddHandle} size="small" variant="outlined">
          <Icon.Add />
        </Button.Button>
      </Header.Header>
      <Select.Frame
        value={value}
        onChange={onChange}
        data={data}
        closeDialogOnSelect={false}
      >
        <List.Items<string> y gap={1}>
          {({ key, index }) => <HandleListItem key={key} itemKey={key} index={index} />}
        </List.Items>
      </Select.Frame>
    </Flex.Box>
  );
};
