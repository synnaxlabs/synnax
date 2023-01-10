import { useCallback } from "react";

import { Space } from "@synnaxlabs/pluto";

import { SelectAxisInputItem, SelectMultipleAxesInputItem } from "../../../components";
import { AxisKey } from "../../../types";
import { LinePlotV, LinePlotVS } from "../../types";

import { useSelectRanges, SelectMultipleRangesInputItem } from "@/features/workspace";
import { DeepPartial } from "@/util/types";

export interface LinePlotDataControlsProps {
  vis: LinePlotVS;
  onChange: (vis: DeepPartial<LinePlotV>) => void;
}

export const LinePlotDataControls = ({
  vis,
  onChange,
}: LinePlotDataControlsProps): JSX.Element | null => {
  const ranges = useSelectRanges();

  const handleChannelSelect = useCallback(
    (key: AxisKey, value: readonly string[] | string): void => {
      onChange({ channels: { [key]: value } });
    },
    [onChange]
  );

  const handleRangeSelect = useCallback(
    (value: readonly string[]): void => {
      onChange({ ranges: { x1: value } });
    },
    [onChange]
  );

  return (
    <Space style={{ padding: "2rem", maxWidth: "100%" }}>
      <SelectMultipleAxesInputItem
        axis={"y1"}
        onChange={handleChannelSelect}
        value={vis.channels.y1}
        grow
      />
      <Space direction="horizontal" grow>
        <SelectAxisInputItem
          axis={"x1"}
          onChange={handleChannelSelect}
          value={vis.channels.x1}
        />
        <SelectMultipleRangesInputItem
          data={ranges}
          onChange={handleRangeSelect}
          value={vis.ranges.x1.map((v) => v.key)}
        />
      </Space>
    </Space>
  );
};
