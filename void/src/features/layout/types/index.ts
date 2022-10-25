import { Component, ReactNode } from "react";

export type LayoutContent<S, P> = {
  key: string;
  type: string;
  title: string;
  props: P;
  state: S;
};

export type LayoutPlacementLocation = "window" | "mosaic";

export type LayoutPlacement = {
  winKey?: string;
  location: LayoutPlacementLocation;
  contentKey: string;
};

export type LayoutRendererProps<S, P> = LayoutContent<S, P> &
  Omit<LayoutPlacement, "contentKey">;

export type LayoutRenderer<S, P> = Component<LayoutRendererProps<S, P>>;
