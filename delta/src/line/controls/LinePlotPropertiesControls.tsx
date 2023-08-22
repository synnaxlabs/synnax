// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { Input, Align, componentRenderProp } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { renameLayout, useSelectRequiredLayout } from "@/layout";

import { useSelectLinePlot } from "../store/selectors";
import { setLinePlotLegend, setLinePlotTitle } from "../store/slice";

export interface LinePlotPropertiesControlsProps {
  layoutKey: string;
}

export const LinePlotPropertiesControls = ({
  layoutKey,
}: LinePlotPropertiesControlsProps): ReactElement => {
  const plot = useSelectLinePlot(layoutKey);
  const { name } = useSelectRequiredLayout(layoutKey);
  const dispatch = useDispatch();

  const handleTitleRename = (value: string): void => {
    dispatch(renameLayout({ key: layoutKey, name: value }));
  };

  const handleTitleVisibilityChange = (value: boolean): void => {
    dispatch(setLinePlotTitle({ key: layoutKey, title: { visible: value } }));
  };

  const handleLegendVisibilityChange = (value: boolean): void => {
    dispatch(setLinePlotLegend({ key: layoutKey, legend: { visible: value } }));
  };

  return (
    <Align.Space direction="y" style={{ padding: "2rem" }}>
      <Input.Item<string> label="Title" value={name} onChange={handleTitleRename} />
      <Align.Space direction="x" size="small">
        <Input.Item<boolean>
          label="Show Title"
          value={plot.title.visible}
          onChange={handleTitleVisibilityChange}
        >
          {componentRenderProp(Input.Switch)}
        </Input.Item>
        <Input.Item<boolean>
          label="Show Legend"
          value={plot.legend.visible}
          onChange={handleLegendVisibilityChange}
        >
          {componentRenderProp(Input.Switch)}
        </Input.Item>
      </Align.Space>
    </Align.Space>
  );
};
