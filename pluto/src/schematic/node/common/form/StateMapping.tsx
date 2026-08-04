// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, id } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Button } from "@/button";
import { Color } from "@/color";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Form } from "@/form";
import { Icon } from "@/icon";
import { List } from "@/list";
import { type StateMapping } from "@/schematic/node/general/stateIndicator/config";
import { Text } from "@/text";

interface StateMappingFormProps {
  path: string;
  showColor?: boolean;
}

interface StateMappingListItemProps {
  itemKey: string;
  index: number;
  path: string;
  showColor: boolean;
  duplicateValue: boolean;
  onRemove: (key: string) => void;
}

const StateMappingListItem = ({
  itemKey,
  index,
  path,
  showColor,
  duplicateValue,
  onRemove,
}: StateMappingListItemProps): ReactElement => {
  const basePath = `${path}.${itemKey}`;
  return (
    <List.Item
      key={itemKey}
      itemKey={itemKey}
      index={index}
      x
      align="center"
      gap="small"
    >
      <Form.TextField
        showLabel={false}
        showHelpText={false}
        path={`${basePath}.name`}
        grow
      />
      <Form.NumericField
        path={`${basePath}.value`}
        showHelpText={false}
        showLabel={false}
        inputProps={{
          status: duplicateValue ? "error" : undefined,
          className: CSS.BE("state-mapping-list", "value"),
          showDragHandle: false,
          tooltip: duplicateValue ? "Duplicate value" : undefined,
        }}
      />
      {showColor && (
        <Form.Field<color.Crude>
          path={`${basePath}.color`}
          showLabel={false}
          showHelpText={false}
        >
          {({ value, onChange }) => (
            <Color.Swatch value={value ?? color.ZERO} onChange={onChange} bordered />
          )}
        </Form.Field>
      )}
      <Button.Button
        onClick={() => onRemove(itemKey)}
        size="small"
        variant="text"
        reveal
      >
        <Icon.Close />
      </Button.Button>
    </List.Item>
  );
};

export const StateMappingForm = ({
  path,
  showColor = false,
}: StateMappingFormProps): ReactElement => {
  const { data, push, remove } = Form.useFieldList<string, StateMapping>(path);
  const options = Form.useFieldValue<StateMapping[]>(path);

  const handleAddOption = (): void => {
    const nextValue =
      options.length === 0 ? 0 : Math.max(...options.map((o) => o.value)) + 1;
    push({ key: id.create(), name: "", value: nextValue });
  };

  const duplicateValues = new Set(
    options.map((o) => o.value).filter((v, i, arr) => arr.indexOf(v) !== i),
  );

  return (
    <Flex.Box
      y
      gap="small"
      align="stretch"
      grow={options.length === 0}
      className={CSS.B("state-mapping-list")}
    >
      <List.Frame data={data}>
        <List.Items<string>
          grow
          emptyContent={
            <Flex.Box center grow>
              <Text.Text center status="disabled" gap="tiny">
                No options added.
                <Text.Text variant="link" onClick={handleAddOption}>
                  Add an option
                </Text.Text>
              </Text.Text>
            </Flex.Box>
          }
        >
          {({ itemKey, index }) => {
            if (index >= options.length) return null;
            return (
              <StateMappingListItem
                itemKey={itemKey}
                index={index}
                path={path}
                showColor={showColor}
                duplicateValue={duplicateValues.has(options[index].value)}
                onRemove={remove}
              />
            );
          }}
        </List.Items>
      </List.Frame>
      {options.length > 0 && (
        <Button.Button
          onClick={handleAddOption}
          variant="text"
          size="small"
          textColor={10}
        >
          <Icon.Add />
          Add option
        </Button.Button>
      )}
    </Flex.Box>
  );
};
