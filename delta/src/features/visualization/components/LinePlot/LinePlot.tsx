import { Synnax } from "@synnaxlabs/client";
import {
	AutoSize,
	LinePlot as PlutoLinePlot,
	PlotData,
	Theming,
} from "@synnaxlabs/pluto";
import { useEffect, useRef, useState } from "react";
import { SugaredLinePlotVisualization, Visualization } from "../../types";
import { LinePlotControls } from "./LinePlotControls";
import "./LinePlot.css";

export interface LinePlotProps {
	visualization: SugaredLinePlotVisualization;
	onChange: (vis: Visualization) => void;
	client: Synnax;
	resizeDebounce: number;
}

function usePrevious<V>(value: V) {
	// The ref object is a generic container whose current property is mutable ...
	// ... and can hold any value, similar to an instance property on a class
	const ref = useRef<V>();
	// Store current value in ref
	useEffect(() => {
		ref.current = value;
	}, [value]); // Only re-run if value changes
	// Return previous value (happens before update in useEffect above)
	return ref.current;
}

export const LinePlot = ({
	visualization,
	client,
	onChange,
	resizeDebounce,
}: LinePlotProps) => {
	const { axes, series, channels, ranges } = visualization;
	const [data, setData] = useState<PlotData>({});
	const {
		theme: { colors },
	} = Theming.useContext();
	const prevVisu = usePrevious(visualization);

	useEffect(() => {
		if (
			prevVisu &&
			prevVisu.channels.length == visualization.channels.length &&
			prevVisu.ranges.length === visualization.ranges.length
		)
			return;
		const fn = async () => {
			const nextData: PlotData = {};
			console.log("HELLO", ranges, channels);
			for (const range of ranges) {
				for (const key of channels) {
					const data = await client.data.read(key, range.start, range.end);
					console.log(data);
					nextData[key] = data;
					if (channels.indexOf(key) === channels.length - 1) {
						nextData["time"] = Array.from({ length: data?.length || 0 }, (_, i) => i);
					}
				}
			}
			setData(nextData);
			onChange({
				...visualization,
				ranges: ranges.map((range) => range.key),
				series: channels.map((ch) => ({
					label: ch,
					x: "time",
					y: ch,
					color: colors.visualization.palettes.default[channels.indexOf(ch)],
					axis: "y",
				})),
				axes: [
					{
						key: "x",
						location: "bottom",
						label: "x",
					},
					{
						key: "y",
						location: "left",
						label: "y",
					},
				],
				channels,
			} as Visualization);
		};
		fn();
	}, [client, channels, ranges]);

	return (
		<div className="delta-line-plot__container">
			<AutoSize className="delta-line-plot__plot__container" debounce={resizeDebounce}>
				{({ width, height }) => (
					<PlutoLinePlot
						width={width}
						height={height}
						data={data}
						axes={axes}
						series={series}
					/>
				)}
			</AutoSize>
			<LinePlotControls
				visualization={visualization}
				onChange={onChange}
				client={client}
			/>
		</div>
	);
};
