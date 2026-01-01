// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, Input } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";
import { useDispatch } from "react-redux";

import { Layout } from "@/layout";
import { useSelect } from "@/lineplot/selectors";
import { setLegend, setTitle } from "@/lineplot/slice";

export interface PropertiesProps {
  layoutKey: string;
}

export const Properties = ({ layoutKey }: PropertiesProps): ReactElement => {
  const plot = useSelect(layoutKey);
  const { name } = Layout.useSelectRequired(layoutKey);
  const dispatch = useDispatch();

  const handleTitleRename = (value: string): void => {
    dispatch(Layout.rename({ key: layoutKey, name: value }));
  };

  const handleTitleVisibilityChange = (value: boolean): void => {
    dispatch(setTitle({ key: layoutKey, title: { visible: value } }));
  };

  const handleLegendVisibilityChange = (value: boolean): void => {
    dispatch(setLegend({ key: layoutKey, legend: { visible: value } }));
  };

  return (
    <Flex.Box x style={{ padding: "2rem" }}>
      <Input.Item label="Title" grow>
        <Input.Text
          value={name}
          onChange={handleTitleRename}
          selectOnFocus
          resetOnBlurIfEmpty
        />
      </Input.Item>
      <Input.Item label="Show Title">
        <Input.Switch
          value={plot.title.visible}
          onChange={handleTitleVisibilityChange}
        />
      </Input.Item>
      <Input.Item label="Show Legend">
        <Input.Switch
          value={plot.legend.visible}
          onChange={handleLegendVisibilityChange}
        />
      </Input.Item>
    </Flex.Box>
  );
};
