// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { memo } from "react";

import { useDispatch } from "react-redux";

import { LinePlot } from "../line/LinePlot";
import { setVisualization, useSelectSugaredVisualization } from "../store";
import { SugaredLinePlotVisualization, Visualization } from "../types";

import { useClusterClient } from "@/features/cluster";
import { LayoutRenderer, LayoutRendererProps } from "@/features/layout";

export const VisualizationLayoutRenderer: LayoutRenderer = {
  Renderer: memo(({ layoutKey }: LayoutRendererProps) => {
    const vis = useSelectSugaredVisualization(layoutKey);
    const dispatch = useDispatch();
    const client = useClusterClient();
    if (vis == null || client == null) return <h1>No Client</h1>;

    const onChange = (vis: Visualization): void => {
      dispatch(setVisualization(vis));
    };

    switch (vis.variant) {
      case "linePlot":
        return (
          <LinePlot
            visualization={vis as SugaredLinePlotVisualization}
            client={client}
            onChange={onChange}
            resizeDebounce={100}
          />
        );
    }
    return <h1>No Visualization Found</h1>;
  }),
};
VisualizationLayoutRenderer.Renderer.displayName = "VisualizationLayoutRenderer";
