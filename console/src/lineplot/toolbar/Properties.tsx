// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot } from "@synnaxlabs/client";
import { Flex, Input, LinePlot } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/css";

export const Properties = (): ReactElement => {
  const name = LinePlot.useSelectName({});
  const title = LinePlot.useSelectTitle({});
  const legend = LinePlot.useSelectLegend({});
  const dispatch = LinePlot.useSingleDispatch();

  const handleRename = (name: string): void => dispatch(lineplot.rename({ name }));

  const handleTitleVisibilityChange = (visible: boolean): void =>
    dispatch(lineplot.setTitleVisible({ visible }));

  const handleLegendVisibilityChange = (visible: boolean): void =>
    dispatch(lineplot.setLegendVisible({ visible }));

  return (
    <Flex.Box x className={CSS.BE("line-plot", "toolbar", "properties")}>
      <Input.Item label="Title" grow>
        <Input.Text
          value={name}
          onChange={handleRename}
          selectOnFocus
          resetOnBlurIfEmpty
        />
      </Input.Item>
      <Input.Item label="Show Title">
        <Input.Switch value={title.visible} onChange={handleTitleVisibilityChange} />
      </Input.Item>
      <Input.Item label="Show Legend">
        <Input.Switch value={legend.visible} onChange={handleLegendVisibilityChange} />
      </Input.Item>
    </Flex.Box>
  );
};
