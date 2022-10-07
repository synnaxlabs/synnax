import Input from "../Input/Input";
import MultiSelect, { MultiSelectProps } from "./MultiSelect";

export default {
  title: "Atoms/MultiSelect",
  component: MultiSelect,
};

const options = Array.from({ length: 500 }).map((_, i) => ({
  key: i,
  name: "strainGauge" + i,
  dataType: "Float64",
  sampleRate: i,
}));

const Template = (props: MultiSelectProps<string, { key: string }>) => (
  <div style={{ top: 400, position: "absolute", width: "80%" }}>
    <MultiSelect {...props} listPosition="bottom" />
    <MultiSelect {...props} />
  </div>
);

export const Primary = Template.bind({});
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
