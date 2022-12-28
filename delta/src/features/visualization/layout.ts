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
