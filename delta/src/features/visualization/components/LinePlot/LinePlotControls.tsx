import { Select, Space } from "@synnaxlabs/pluto";
import { LinePlotVisualization } from "../../types";

export interface LinePlotControlsProps {
  visualization: LinePlotVisualization;
  onChange: (vis: LinePlotVisualization) => void;
}

export const LinePlotControls = () => {
  return (
    <Space direction="vertical">
      <Select.Multiple listPosition="top"></Select.Multiple>
    </Space>
  );
};
