import { Input } from ".";

export default {
  title: "Atoms/Input",
  component: Input,
};

export const Basic = () => <Input />;

export const Time = () => <Input.Time size="medium" onChange={console.log} />;

export const Date = () => <Input.Date size="medium" onChange={console.log} />;
