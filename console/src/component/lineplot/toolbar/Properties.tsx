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
import { useDispatch } from "react-redux";

import { CSS } from "@/component/css";

export const Properties = (): ReactElement => {
  const title = LinePlot.useSelectTitle({ key: layoutKey });
  const legend = LinePlot.useSelectLegend({ key: layoutKey });
  const name = LinePlot.useSelectName();
  const reduxDispatch = useDispatch();
  const { dispatch } = LinePlot.useDispatch();
  const { update: renameRemote } = LinePlot.useRename({});

  const handleTitleRename = (value: string): void => {
    reduxDispatch(Layout.rename({ key: layoutKey, name: value }));
    renameRemote({ key: layoutKey, name: value });
  };

  const handleTitleVisibilityChange = (visible: boolean): void => {
    dispatch({
      key: layoutKey,
      actions: [lineplot.setTitle({ title: { ...title, visible } })],
    });
  };

  const handleLegendVisibilityChange = (visible: boolean): void => {
    dispatch({
      key: layoutKey,
      actions: [lineplot.setLegendHidden({ hidden: !visible })],
    });
  };

  return (
    <Flex.Box x className={CSS.BE("line-plot", "toolbar", "properties")}>
      <Input.Item label="Title" grow>
        <Input.Text
          value={name}
          onChange={handleTitleRename}
          selectOnFocus
          resetOnBlurIfEmpty
        />
      </Input.Item>
      <Input.Item label="Show Title">
        <Input.Switch value={title.visible} onChange={handleTitleVisibilityChange} />
      </Input.Item>
      <Input.Item label="Show Legend">
        <Input.Switch value={!legend.hidden} onChange={handleLegendVisibilityChange} />
      </Input.Item>
    </Flex.Box>
  );
};
