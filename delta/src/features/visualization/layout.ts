// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { nanoid } from "nanoid";

import { setVisualization as storeCreateVizualization } from "./store";
import { Visualization } from "./types";

import { Layout, LayoutCreator, LayoutCreatorProps } from "@/features/layout";

export const createVisualization =
  <V extends Visualization>(initial: Partial<V>): LayoutCreator =>
  ({ dispatch }: LayoutCreatorProps): Layout => {
    const key = initial.key ?? nanoid();
    dispatch(
      storeCreateVizualization({
        ...initial,
        key,
        variant: "linePlot",
      })
    );
    return {
      key,
      location: "mosaic",
      type: "visualization",
      title: initial.key ?? "Plot",
    };
  };
