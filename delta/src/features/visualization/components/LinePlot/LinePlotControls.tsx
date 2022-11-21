import { useSelectRanges } from "@/features/workspace";
import { Synnax, ChannelPayload } from "@synnaxlabs/client";
import { Select, Space } from "@synnaxlabs/pluto";
import { useEffect, useState } from "react";
import { LinePlotVisualization, SugaredLinePlotVisualization } from "../../types";

export interface LinePlotControlsProps {
	visualization: SugaredLinePlotVisualization;
	onChange: (vis: LinePlotVisualization) => void;
	client: Synnax;
}

export const LinePlotControls = ({
	visualization,
	onChange,
	client,
}: LinePlotControlsProps) => {
	const ranges = useSelectRanges();
	const { channels } = visualization;
	const [channelOpts, setChannelOpts] = useState<(ChannelPayload & { key: string })[]>(
		[]
	);

	useEffect(() => {
		const fn = async () => {
			const channels = await client.channel.retrieveAll();
			setChannelOpts(
				channels.map((ch) => ch.payload as ChannelPayload & { key: string })
			);
		};
		fn();
	}, [client]);

	const handleChannelSelect = (selected: string[]) => {
		onChange({
			...visualization,
			ranges: visualization.ranges.map((range) => range.key),
			channels: selected,
		});
	};

	const handleRangeSelect = (selected: string[]) => {
		onChange({
			...visualization,
			ranges: selected,
			channels: visualization.channels,
		});
	};

	return (
		<Space direction="vertical">
			<Select.Multiple
				selected={channels}
				onSelect={handleChannelSelect}
				options={channelOpts as unknown as (Record<string, string> & { key: string })[]}
				tagKey="name"
				listPosition="top"
				columns={[
					{
						key: "name",
						label: "Name",
					},
				]}
			/>
			<Select.Multiple
				selected={visualization.ranges.map((range) => range.key)}
				listPosition="top"
				onSelect={handleRangeSelect}
				options={ranges}
				columns={[
					{
						key: "name",
						label: "Name",
					},
				]}
			/>
		</Space>
	);
};
