import "./Editor.css";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Menu } from "@synnaxlabs/lyra/menu";
import { type ReactNode, type Ref } from "react";
import { type EditorExtension } from "./language";
import { type Monaco } from "./Provider";
/** EditorHandle is the imperative surface an Editor exposes through its ref, for consumers
 * that own the editor's text and update it programmatically rather than through React
 * state. */
export interface EditorHandle {
    /** setValue reconciles the editor's model to next as a minimal edit, preserving the
     * local cursor and selection. It is a no-op when the model already holds next, and does
     * not fire onChange, since the edit did not originate from the user. */
    setValue: (next: string) => void;
}
interface UseProps {
    initialValue?: string;
    onValueChange?: (value: string) => void;
    onEdit?: (changes: readonly Monaco.editor.IModelContentChange[]) => void;
    language: string;
    isBlock?: boolean;
    scrollBeyondLastLine?: boolean;
    openContextMenu?: Menu.ContextMenuProps["open"];
    extensions?: EditorExtension[];
    /** placeholder is ghost text shown on the first line while the document is empty. It
     * is injected, so it never enters the model or the user's selection. Read when the
     * editor is created. */
    placeholder?: string;
    /** autoFocus places the cursor in the editor once it is created. Read when the editor
     * is created. */
    autoFocus?: boolean;
}
export interface EditorProps extends Omit<Flex.BoxProps, "value" | "onChange" | "ref">, UseProps {
    ref?: Ref<EditorHandle>;
    loading?: ReactNode;
    /** extraMenuItems are appended to the context menu, so a consumer can add
     * app-specific entries (e.g. "Reload Console"). */
    extraMenuItems?: ReactNode;
}
/** Editor is a Monaco-backed code editor. It suspends while the Monaco runtime
 * initializes on first use, rendering `loading` until ready, and surfaces an
 * initialization failure to the boundary it provides. */
export declare const Editor: ({ loading, ...props }: EditorProps) => import("react").JSX.Element;
export {};
//# sourceMappingURL=Editor.d.ts.map