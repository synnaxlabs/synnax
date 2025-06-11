import "@/vis/schematic/Schematic.css";

import { type ReactElement } from "react";

import { CSS } from "@/css";
import { componentRenderProp } from "@/util/renderProp";
import { Diagram } from "@/vis/diagram";
import { ConnectionLine, Edge, type EdgeData } from "@/vis/schematic/edge";
import { DRAG_HANDLE_CLASS } from "@/vis/schematic/Grid";

export interface SchematicProps
  extends Omit<Diagram.DiagramProps, "dragHandleSelector"> {}

const edgeRenderer = componentRenderProp(Edge);

export const Schematic = ({
  className,
  children,
  ...props
}: SchematicProps): ReactElement => (
  <Diagram.Diagram
    className={CSS(CSS.B("schematic"), className)}
    dragHandleSelector={`.${DRAG_HANDLE_CLASS}`}
    {...props}
  >
    <Diagram.EdgeRenderer<EdgeData> connectionLineComponent={ConnectionLine}>
      {edgeRenderer}
    </Diagram.EdgeRenderer>
    {children}
  </Diagram.Diagram>
);
