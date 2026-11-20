// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/common/form/StateMapping.css";

import { type schematic } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Color } from "@synnaxlabs/lyra/color";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { array, color, type deep, id } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";

interface StateMappingFormProps {
  path: deep.Path<schematic.StateMapping[]>;
  showColor?: boolean;
}

interface StateMappingListItemProps {
  itemKey: string;
  index: number;
  path: deep.Path<schematic.StateMapping[]>;
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
  const basePath = path[itemKey];
  return (
    <List.Item
      key={itemKey}
      itemKey={itemKey}
      index={index}
      x
      align="center"
      gap="small"
      preventClick
      className={CSS.BE("state-mapping-list", "item")}
    >
      <Form.TextField
        showLabel={false}
        showHelpText={false}
        path={basePath.name}
        grow
      />
      <Form.NumericField
        path={basePath.value}
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
        <Color.Field
          path={basePath.color}
          fallback={color.TRANSPARENT}
          showLabel={false}
          showHelpText={false}
        />
      )}
      <Button.Button onClick={() => onRemove(itemKey)} size="small" variant="text">
        <Icon.Close />
      </Button.Button>
    </List.Item>
  );
};

export const StateMappingForm = ({
  path,
  showColor = false,
}: StateMappingFormProps): ReactElement => {
  const { data, push, remove } = Form.useFieldList(path);
  const options = Form.useFieldValue(path);

  const handleAddOption = (): void => {
    const max = array.maxBy(options, (o) => o.value);
    const nextValue = max == null ? 0 : max.value + 1;
    push({ key: id.create(), name: "", value: nextValue });
  };

  const duplicateValues = new Set(
    options.map((o) => o.value).filter((v, i, arr) => arr.indexOf(v) !== i),
  );

  return (
    <Flex.Box
      y
      empty
      align="stretch"
      justify={options.length === 0 ? "center" : undefined}
      className={CSS.cls(
        CSS.B("state-mapping-list"),
        options.length === 0 && CSS.BM("state-mapping-list", "empty"),
      )}
    >
      {options.length > 0 && (
        <List.Frame data={data}>
          <List.Scroll grow className={CSS.BE("state-mapping-list", "scroll")}>
            <List.Items<string>>
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
          </List.Scroll>
        </List.Frame>
      )}
      <Button.Button
        onClick={handleAddOption}
        variant="text"
        size="small"
        textColor={10}
      >
        <Icon.Add />
        Add option
      </Button.Button>
    </Flex.Box>
  );
};
