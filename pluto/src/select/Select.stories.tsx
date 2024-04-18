// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useState } from "react";

import type { Meta, StoryFn } from "@storybook/react";

import { Align } from "@/align";
import { type List } from "@/list";
import { Select } from "@/select";

const story: Meta<typeof Select.Multiple> = {
  title: "Select",
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

const sampleColumns: Array<List.ColumnSpec<string, SampleRecord>> = [
  {
    key: "name",
    name: "Name",
    visible: true,
  },
  {
    key: "dataType",
    name: "Data Type",
    visible: true,
  },
  {
    key: "sampleRate",
    name: "Sample Rate",
    visible: true,
  },
];

export const Multiple = (): ReactElement => {
  const [value, setValue] = useState<string[]>([]);
  const [valueTwo, setValueTwo] = useState<string[]>([]);

  return (
    <Align.Space>
      <Select.Multiple
        value={value}
        onChange={setValue}
        data={sampleData}
        columns={sampleColumns}
        location="top"
      />
      <Select.Multiple
        value={valueTwo}
        onChange={setValueTwo}
        data={sampleData}
        columns={sampleColumns}
      />
    </Align.Space>
  );
};

export const DropdownButton = (): ReactElement => {
  const [value, setValue] = useState<string>("");

  return (
    <Align.Space>
      <Select.DropdownButton
        value={value}
        onChange={setValue}
        data={sampleData}
        columns={sampleColumns}
        location="top"
      />
    </Align.Space>
  );
};

const Template = (
  props: Omit<Select.SingleProps<string, SampleRecord>, "value" | "onChange">,
): ReactElement => {
  const [value, setValue] = useState<string>("");
  return <Select.Single {...props} value={value} onChange={setValue} />;
};

export const Default: StoryFn<typeof Template> = Template.bind({});

Default.args = {
  columns: sampleColumns,
  data: sampleData,
};

// eslint-disable-next-line import/no-default-export
export default story;
