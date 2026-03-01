import type { CSSProperties } from "react";

export type NodeIcon = React.ComponentType<{ style?: CSSProperties }>;

export interface NodeDef {
  id: string;
  x: number;
  y: number;
  w: number;
  h: number;
  label: string;
  icon: NodeIcon;
}

export interface EdgeDef {
  from: string;
  to: string;
}

export type DiagramVariant = "schematic" | "pill";

export interface DiagramDef {
  viewBox: string;
  nodes: NodeDef[];
  edges: EdgeDef[];
  variant?: DiagramVariant;
}
