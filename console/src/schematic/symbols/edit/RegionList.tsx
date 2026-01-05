// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import {
  Button,
  Color,
  Flex,
  Form,
  Header,
  Icon,
  type Input,
  List,
  Select,
  Text,
  Tooltip,
} from "@synnaxlabs/pluto";
import { color } from "@synnaxlabs/x";

export interface RegionListProps extends Input.Control<string | undefined> {
  selectedState: string;
  onAddRegion: () => void;
}

export interface RegionListItemProps extends List.ItemRenderProps<string> {
  selectedState: string;
}

export const RegionListItem = ({ selectedState, ...props }: RegionListItemProps) => {
  const { itemKey } = props;
  const path = `data.states.${selectedState}.regions.${itemKey}`;
  const region = Form.useFieldValue<schematic.symbol.Region>(path, { optional: true });
  const { remove } = Form.useFieldListUtils<string, schematic.symbol.Region>(
    `data.states.${selectedState}.regions`,
  );
  if (region == null) return null;
  return (
    <Select.ListItem {...props} justify="between" style={{ paddingRight: "0.5rem" }}>
      <Flex.Box x align="center" gap={1}>
        <Form.Field<string> path={`${path}.name`} showLabel={false}>
          {({ onChange, value }) => (
            <Text.MaybeEditable
              level="small"
              value={value}
              onChange={onChange}
              style={{ minWidth: 80 }}
            />
          )}
        </Form.Field>
      </Flex.Box>
      <Flex.Box x align="center" gap={1}>
        <Text.Text level="small" color={7}>
          {region?.selectors?.length || 0} Elements
        </Text.Text>
        <Form.Field<string> path={`${path}.strokeColor`} showLabel={false}>
          {({ onChange, value }) => (
            <Color.Swatch
              value={value}
              onChange={(v) => onChange(color.hex(v))}
              size="small"
            />
          )}
        </Form.Field>
        <Form.Field<string> path={`${path}.fillColor`} showLabel={false}>
          {({ onChange, value }) => (
            <Color.Swatch
              value={value}
              onChange={(v) => onChange(color.hex(v))}
              size="small"
            />
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

export const RegionList = ({
  value,
  onChange,
  selectedState,
  onAddRegion,
}: RegionListProps) => {
  const { data } = Form.useFieldList<string, schematic.symbol.Region>(
    `data.states.${selectedState}.regions`,
  );
  return (
    <Flex.Box y gap={1} style={{ maxHeight: 200 }}>
      <Header.Header level="p" padded bordered={false}>
        <Header.Title level="p" weight={500}>
          Colors
        </Header.Title>
        <Header.Actions>
          <Text.Text level="p" color={7} gap={3}>
            <Tooltip.Dialog>
              <Text.Text level="small">Stroke Color</Text.Text>
              <Flex.Box>
                <Icon.StrokeColor />
              </Flex.Box>
            </Tooltip.Dialog>
            <Tooltip.Dialog>
              <Text.Text level="small">Fill Color</Text.Text>
              <Flex.Box>
                <Icon.FillColor />
              </Flex.Box>
            </Tooltip.Dialog>
          </Text.Text>
          <Button.Button onClick={onAddRegion} size="small" variant="outlined">
            <Icon.Add />
          </Button.Button>
        </Header.Actions>
      </Header.Header>
      <Select.Frame
        value={value}
        onChange={onChange}
        data={data}
        closeDialogOnSelect={false}
      >
        <List.Items<string> y gap={1}>
          {({ key, ...rest }) => (
            <RegionListItem selectedState={selectedState} key={key} {...rest} />
          )}
        </List.Items>
      </Select.Frame>
    </Flex.Box>
  );
};
