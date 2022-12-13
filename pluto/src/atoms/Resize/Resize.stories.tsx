import { ComponentMeta, ComponentStory } from "@storybook/react";
import { Resize } from ".";
import { ResizePanelProps } from "./Resize";

export default {
	title: "Atoms/Resize",
	component: Resize,
} as ComponentMeta<typeof Resize>;

const Template = (args: ResizePanelProps) => (
	<Resize {...args}>
		<h1>Resize</h1>
	</Resize>
);

export const Primary: ComponentStory<typeof Resize> = Template.bind({});
Primary.args = {
	style: {
		height: "100%",
	},
};

export const Multiple: ComponentStory<typeof Resize.Multiple> = () => {
	const props = Resize.useMultiple({ initialSizes: [100, 200], itemCount: 3 });
	return (
		<Resize.Multiple
			{...props}
			style={{ border: "1px solid var(--pluto-gray-m2)", height: "100%" }}
		>
			<h1>Hello From One</h1>
			<h1>Hello From Two</h1>
			<h1>Hello From Three</h1>
		</Resize.Multiple>
	);
};
