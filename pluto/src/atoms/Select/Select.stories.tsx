// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useState } from "@storybook/addons";
import type { ComponentMeta, ComponentStory } from "@storybook/react";

import { ListColumn } from "../List";

import { Select, SelectMultipleProps } from ".";

import { RenderableRecord } from "@/util/record";

const story: ComponentMeta<typeof Select.Multiple> = {
  title: "Atoms/Select",
  component: Select.Multiple,
};

const sampleData = Array.from({ length: 500 }).map((_, i) => ({
  key: `Option ${i}`,
  name: `strain-gauge-${i}`,
  dataType: "Float64",
  sampleRate: i,
}));

interface SampleRecord {
  key: string;
  sampleRate: number;
  name: string;
  dataType: string;
}

const sampleColumns: Array<ListColumn<SampleRecord>> = [
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
];

const MultipleTemplate = <E extends RenderableRecord<E>>(
  args: Omit<SelectMultipleProps<E>, "value" | "onChange">
): JSX.Element => {
  const [value, setValue] = useState<readonly string[]>([]);
  return <Select.Multiple {...args} value={value} onChange={setValue} />;
};

export const Multiple: ComponentStory<typeof Select.Multiple<SampleRecord>> =
  MultipleTemplate.bind({});

Multiple.args = {
  value: [],
  columns: sampleColumns,
  location: "bottom",
  data: sampleData,
};

const Template = (
  props: Omit<SelectMultipleProps<SampleRecord>, "value" | "onChange">
): JSX.Element => {
  const [value, setValue] = useState<string>("");
  return <Select {...props} value={value} onChange={setValue} />;
};

export const Default: ComponentStory<typeof Template> = Template.bind({});

Default.args = {
  columns: sampleColumns,
  location: "bottom",
  data: sampleData,
};

export default story;
