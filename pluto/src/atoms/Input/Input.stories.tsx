import type { ComponentMeta } from "@storybook/react";

import { Input } from ".";

const story: ComponentMeta<typeof Input> = {
  title: "Atoms/Input",
  component: Input,
};

export const Basic = (): JSX.Element => <Input />;

export const Time = (): JSX.Element => (
  <Input.Time size="medium" onChange={console.log} />
);

export const Date = (): JSX.Element => (
  <Input.Date size="medium" onChange={console.log} />
);

export default story;
