// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Input, Status, Button, Select, Align } from "@synnaxlabs/pluto";

import { Layout } from "@/layout";
import { listColumns } from "@/range/accordionEntry";
import { defineWindowLayout } from "@/range/Define";
import { type Range } from "@/range/range";

export interface SelectMultipleRangesProps
  extends Omit<Select.MultipleProps<string, Range>, "columns"> {}

export const SelectMultipleRanges = (
  props: SelectMultipleRangesProps
): ReactElement => <Select.Multiple columns={listColumns} tagKey="name" {...props} />;

export interface SelectSingleRangeProps
  extends Omit<Select.SingleProps<string, Range>, "columns"> {}

export const SelectRange = (props: SelectSingleRangeProps): ReactElement => (
  <Select.Single columns={listColumns} {...props} tagKey="name" />
);

export interface SelectMultipleInputItemProps
  extends Omit<
    Input.ItemProps<string[], string[], SelectMultipleRangesProps>,
    "label"
  > {}

const SelectEmptyContent = (): ReactElement => {
  const newLayout = Layout.usePlacer();
  return (
    <Align.Center style={{ height: 150 }} direction="x">
      <Status.Text variant="disabled" hideIcon>
        No Ranges:
      </Status.Text>
      <Button.Button
        variant="outlined"
        onClick={() => {
          newLayout(defineWindowLayout);
        }}
      >
        Define a Range
      </Button.Button>
    </Align.Center>
  );
};

export const SelectMultipleInputItem = (
  props: SelectMultipleInputItemProps
): ReactElement => (
  <Input.Item<string[], string[], SelectMultipleRangesProps>
    direction="x"
    label="Ranges"
    emptyContent={<SelectEmptyContent />}
    {...props}
  >
    {SelectMultipleRanges}
  </Input.Item>
);

export interface SelectInputItemProps
  extends Omit<Input.ItemProps<string, string, SelectSingleRangeProps>, "label"> {}

export const SelectInputItem = (props: SelectInputItemProps): ReactElement => (
  <Input.Item<string, string, SelectSingleRangeProps> label="Range:" {...props}>
    {SelectRange}
  </Input.Item>
);
