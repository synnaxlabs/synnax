import { List, Space } from "@synnaxlabs/pluto";
import { useSelectRanges } from "../store";

export const RangesAccordionEntry = () => {
	const ranges = useSelectRanges();
	console.log(ranges);
	return (
		<Space style={{ height: "100%" }}>
			<List
				data={ranges.map((range) => {
					const start = new Date(range.start / 1000000);
					const end = new Date(range.end / 1000000);
					return {
						key: range.key,
						name: range.name,
						start: start.toISOString().substring(0, 16),
						end: end.toISOString().substring(0, 16),
					};
				})}
			>
				<List.Column.Header
					columns={[
						{
							key: "name",
							label: "Name",
						},
						{
							key: "start",
							label: "Start",
						},
						{
							key: "end",
							label: "End",
						},
					]}
				/>
				<List.Core.Virtual itemHeight={30} style={{ height: "100%" }}>
					{(props) => <List.Column.Item {...props} />}
				</List.Core.Virtual>
			</List>
		</Space>
	);
};
