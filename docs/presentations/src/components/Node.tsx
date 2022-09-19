import { HexagonBar, Text } from "@synnaxlabs/pluto";
import "./Node.css";

export type NodeProps = {
  number?: number;
  width?: number;
};

export default function Node({ number, width = 250 }: NodeProps) {
  return (
    <div style={{ position: "relative", width }}>
      {number && (
        <Text level="h1" className="node__number">
          {number}
        </Text>
      )}
      <HexagonBar
        width={width}
        strokeWidth={5}
        metrics={[
          {
            name: "CPU",
            value: 90,
            max: 100,
            units: "%",
          },
          {
            name: "Memory",
            value: 90,
            max: 100,
            units: "%",
          },
          {
            name: "Disk",
            value: 90,
            max: 100,
            units: "%",
          },
          {
            name: "Net",
            value: 90,
            max: 100,
            units: "%",
          },
        ]}
      />
    </div>
  );
}
