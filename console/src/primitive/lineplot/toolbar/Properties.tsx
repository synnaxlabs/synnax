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
import { type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { CSS } from "@/primitive/css";
import { Session } from "@/session";

export const Properties = (): ReactElement => {
  const key = LinePlot.useKey();
  const title = LinePlot.useSelectTitle();
  const legend = LinePlot.useSelectLegend();
  const name = LinePlot.useSelectName();
  const reduxDispatch = useDispatch();
  const dispatch = LinePlot.useSingleDispatch();
  const { update: rename } = LinePlot.useRename({});

  const handleTitleRename = useCallback(
    (name: string): void => {
      reduxDispatch(Session.Layout.rename({ key, name }));
      rename({ key, name });
    },
    [reduxDispatch, rename],
  );

  const handleTitleVisibilityChange = useCallback(
    (visible: boolean): void => dispatch(lineplot.setTitleVisible({ visible })),
    [dispatch],
  );

  const handleLegendVisibilityChange = useCallback(
    (visible: boolean): void =>
      dispatch(lineplot.setLegendHidden({ hidden: !visible })),
    [dispatch],
  );

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
