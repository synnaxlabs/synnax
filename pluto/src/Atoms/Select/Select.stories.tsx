import { ComponentMeta, ComponentStory } from "@storybook/react";
import Select from ".";
import { Key, TypedListEntry } from "../List/Types";
import SelectMultiple, { SelectMultipleProps } from "./SelectMultiple";

export default {
  title: "Atoms/Select",
  component: Select.Multiple,
} as ComponentMeta<typeof Select.Multiple>;

const options = Array.from({ length: 500 }).map((_, i) => ({
  key: `Option ${i}`,
  name: "strainGauge" + i,
  dataType: "Float64",
  sampleRate: i,
}));

const MultipleTemplate = <K extends Key, E extends TypedListEntry<K>>(
  args: SelectMultipleProps<K, E>
) => <SelectMultiple {...args} />;

export const Multiple: ComponentStory<
  typeof SelectMultiple<
    string,
    {
      key: string;
      sampleRate: number;
      name: string;
      dataType: string;
    }
  >
> = MultipleTemplate.bind({});

Multiple.args = {
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
