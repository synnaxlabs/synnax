import type { ComponentMeta, ComponentStory } from "@storybook/react";

import { Accordion, AccordionEntry } from ".";

const story: ComponentMeta<typeof Accordion> = {
  title: "Molecules/Accordion",
  component: Accordion,
};

const entries: AccordionEntry[] = [
  {
    key: "cluster",
    title: "Cluster",
    content: <p>Content</p>,
  },
  {
    key: "Devices",
    title: "Devices",
    content: <p>Content</p>,
  },
];

export const Primary: ComponentStory<typeof Accordion> = () => (
  <Accordion entries={entries} direction="vertical" />
);

export default story;
