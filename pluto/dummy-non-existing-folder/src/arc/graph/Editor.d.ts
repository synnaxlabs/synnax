import "./Editor.css";
import { type Component } from "@synnaxlabs/lyra/component";
import { Menu } from "@synnaxlabs/lyra/menu";
import { type Triggers } from "@synnaxlabs/lyra/triggers";
import { type ReactElement } from "react";
import { Diagram as BaseDiagram } from "../../vis/diagram";
export interface EditorProps extends Omit<BaseDiagram.DiagramProps, "nodes" | "edges" | "onNodesChange" | "onEdgesChange" | "onChange"> {
    enableTriggers?: Triggers.Condition;
    extraMenuItems?: Component.RenderProp<Menu.ContextMenuMenuProps>;
}
export declare const Editor: ({ viewport, className, selected, editable, onSelectionChange, enableTriggers, extraMenuItems, children, ...props }: EditorProps) => ReactElement;
//# sourceMappingURL=Editor.d.ts.map