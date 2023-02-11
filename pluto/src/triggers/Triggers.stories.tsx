import { List } from "@/core/List/List";
import { ComponentMeta } from "@storybook/react";
import { Triggers } from ".";

const story: ComponentMeta<typeof Triggers.Provider> = {
  title: "Triggers/Triggers",
  component: Triggers.Provider,
};

export const Basic = (): JSX.Element => {
  return (
    <Triggers.Provider>
      <Child />
    </Triggers.Provider>
  );
};

const Child = (): JSX.Element => {
  const { triggers } = Triggers.useHeld([
    ["ArrowDown", null],
    ["ArrowUp", "Shift"],
    ["MouseLeft", "Alt"],
  ]);
  return (
    <div>
      {triggers.map((trigger) => (
        <h1>{Array.isArray(trigger) ? trigger.join(" + ") : trigger}</h1>
      ))}
    </div>
  );
};

export default story;
