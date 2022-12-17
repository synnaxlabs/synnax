import { ComponentType } from "react";

import type { WindowProps } from "@synnaxlabs/drift";

export type LayoutPlacementLocation = "window" | "mosaic";

export interface Layout {
  key: string;
  type: string;
  title: string;
  location: LayoutPlacementLocation;
  window?: LayoutWindowProps;
}

export interface LayoutRendererProps {
  layoutKey: string;
  onClose: () => void;
}

export type LayoutRenderer = ComponentType<LayoutRendererProps>;

export type LayoutWindowProps = Omit<WindowProps, "key" | "url"> & {
  navTop?: boolean;
};
