import { ComponentMeta, ComponentStory } from "@storybook/react";
import { Key, TypedListEntry } from "../List/Types";
import SelectMultiple, { SelectMultipleProps } from "./SelectMultiple";

export default {
  title: "Atoms/MultiSelect",
  component: SelectMultiple,
} as ComponentMeta<typeof SelectMultiple>;

const options = Array.from({ length: 500 }).map((_, i) => ({
  key: `Option ${i}`,
  name: "strainGauge" + i,
  dataType: "Float64",
  sampleRate: i,
}));

const Template = <K extends Key, E extends TypedListEntry<K>>(
  args: SelectMultipleProps<K, E>
) => <SelectMultiple {...args} />;

export const Primary: ComponentStory<
  typeof SelectMultiple<
    string,
    {
      key: string;
      sampleRate: number;
      name: string;
      dataType: string;
    }
  >
> = Template.bind({});

Primary.args = {
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
