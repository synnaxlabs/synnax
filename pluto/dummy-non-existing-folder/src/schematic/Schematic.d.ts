import "./Schematic.css";
import { type Component } from "@synnaxlabs/lyra/component";
import { Menu } from "@synnaxlabs/lyra/menu";
import { type Triggers } from "@synnaxlabs/lyra/triggers";
import { type ReactElement } from "react";
import { Diagram as BaseDiagram } from "../vis/diagram";
export interface SchematicProps extends Omit<BaseDiagram.DiagramProps, "dragHandleSelector" | "nodes" | "edges" | "onNodesChange" | "onEdgesChange" | "onChange"> {
    enableTriggers?: Triggers.Condition;
    extraMenuItems?: Component.RenderProp<Menu.ContextMenuMenuProps>;
    /** Rendered as a centered overlay when the schematic has no nodes. */
    emptyContent?: ReactElement;
}
export declare const Schematic: ({ className, viewport, onDoubleClick, onNodeDoubleClick, onSelectionChange, selected, enableTriggers, extraMenuItems, editable, emptyContent, children, ...props }: SchematicProps) => ReactElement;
//# sourceMappingURL=Schematic.d.ts.map