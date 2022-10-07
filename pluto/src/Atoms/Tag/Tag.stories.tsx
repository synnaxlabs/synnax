import { AiFillAmazonSquare } from "react-icons/ai";
import Tag, { TagProps } from "./Tag";

export default {
  title: "Atoms/Tag",
  component: Tag,
};

const Template = (props: TagProps) => <Tag {...props} />;

export const Primary = Template.bind({});
Primary.args = {
  children: "Tag",
  onClose: () => {},
  variant: "filled",
  size: "medium",
};
