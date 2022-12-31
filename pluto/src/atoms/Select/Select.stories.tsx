// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { ComponentMeta, ComponentStory } from "@storybook/react";

import { RenderableRecord } from "../List/types";

import { Select, SelectMultipleProps } from ".";

const story: ComponentMeta<typeof Select.Multiple> = {
  title: "Atoms/Select",
  component: Select.Multiple,
};

const options = Array.from({ length: 500 }).map((_, i) => ({
  key: `Option ${i}`,
  name: `strain-gauge-${i}`,
  dataType: "Float64",
  sampleRate: i,
}));

const MultipleTemplate = <E extends RenderableRecord<E>>(
  args: SelectMultipleProps<E>
): JSX.Element => <Select.Multiple {...args} />;

export const Multiple: ComponentStory<
  typeof Select.Multiple<{
    key: string;
    sampleRate: number;
    name: string;
    dataType: string;
  }>
> = MultipleTemplate.bind({});

Multiple.args = {
  selected: [],
  columns: [
    {
      key: "name",
      label: "Name",
      visible: true,
    },
    {
      key: "dataType",
      label: "Data Type",
      visible: true,
    },
    {
      key: "sampleRate",
      label: "Sample Rate",
      visible: true,
    },
  ],
  options,
};

export default story;
