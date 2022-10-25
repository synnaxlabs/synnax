import { Component, ReactNode } from "react";

export type LayoutContent<P> = {
  key: string;
  type: string;
  title: string;
  /**
   * Props is the set of props that will be passed to the renderer from the
   * layout
   */
  props: P;
};

export type LayoutPlacementLocation = "window" | "mosaic";

export type LayoutPlacement = {
  winKey?: string;
  location: LayoutPlacementLocation;
  contentKey: string;
};

export type LayoutRenderer<P> = Component<P>;
