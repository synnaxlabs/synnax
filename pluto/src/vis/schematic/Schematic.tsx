import "@/vis/schematic/Schematic.css";

import { type ReactElement } from "react";

import { CSS } from "@/css";
import { componentRenderProp } from "@/util/renderProp";
import { Diagram } from "@/vis/diagram";
import { ConnectionLine, Edge, type EdgeData } from "@/vis/schematic/edge";

export interface SchematicProps extends Diagram.DiagramProps {}

const edgeRenderer = componentRenderProp(Edge);

export const Schematic = ({
  className,
  children,
  ...props
}: SchematicProps): ReactElement => (
  <Diagram.Diagram className={CSS(CSS.B("schematic"), className)} {...props}>
    <Diagram.EdgeRenderer<EdgeData> connectionLineComponent={ConnectionLine}>
      {edgeRenderer}
    </Diagram.EdgeRenderer>
    {children}
  </Diagram.Diagram>
);
