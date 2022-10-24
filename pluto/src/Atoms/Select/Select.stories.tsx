import { ComponentMeta, ComponentStory } from "@storybook/react";
import { ListEntry } from "../List/Types";
import { Select, SelectMultipleProps } from ".";

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

const MultipleTemplate = <E extends ListEntry>(
  args: SelectMultipleProps<E>
) => <Select.Multiple {...args} />;

export const Multiple: ComponentStory<
  typeof Select.Multiple<{
    key: string;
    sampleRate: number;
    name: string;
    dataType: string;
  }>
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
