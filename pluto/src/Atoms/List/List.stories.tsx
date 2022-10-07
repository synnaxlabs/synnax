import VirtualCore from "./CoreList";
import { SelectableColumnSearchList, ColumnListProps } from "./Lists";

export default {
  title: "Atoms/List",
  component: VirtualCore,
};

const data = Array.from({ length: 500 }, (_, i) => ({
  key: i,
  name: `Item ${i + 1000}`,
  count: i,
}));

const Template = (
  args: ColumnListProps<
    string,
    {
      key: string;
      name: string;
      count: number;
    }
  >
) => <SelectableColumnSearchList {...args} />;

export const Primary = Template.bind({});
Primary.args = {
  data,
  itemHeight: 30,
  columns: [
    { key: "name", label: "Name", visible: true },
    { key: "count", label: "Count", visible: true },
  ],
  style: { height: 300 },
};
