import { ComponentMeta, ComponentStory } from "@storybook/react";
import { AiFillDatabase, AiFillPhone } from "react-icons/ai";
import { Accordion, AccordionEntry } from ".";

export default {
	title: "Molecules/Accordion",
	component: Accordion,
} as ComponentMeta<typeof Accordion>;

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
