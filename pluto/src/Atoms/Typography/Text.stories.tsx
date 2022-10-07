import IconText from "./IconText";
import { AiOutlineDelete } from "react-icons/ai";

export default {
  title: "Atoms/IconText",
  component: IconText,
};

export const Primary = () => (
  <IconText
    level="h2"
    startIcon={<AiOutlineDelete />}
    endIcon={<AiOutlineDelete />}
  >
    Hello
  </IconText>
);
