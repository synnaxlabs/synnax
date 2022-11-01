import { WindowProps } from "@synnaxlabs/drift";
import { Component, ComponentProps, ComponentType } from "react";

export type LayoutPlacementLocation = "window" | "mosaic";

export type Layout = {
  key: string;
  type: string;
  title: string;
  location: LayoutPlacementLocation;
  window?: LayoutWindowProps;
};

export type LayoutRendererProps = {
  layoutKey: string;
  onClose: () => void;
};

export type LayoutRenderer = ComponentType<LayoutRendererProps>;

export type LayoutWindowProps = Omit<WindowProps, "key" | "url"> & {
  navTop?: boolean;
};
