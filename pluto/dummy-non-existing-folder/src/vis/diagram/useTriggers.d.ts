import { Triggers } from "@synnaxlabs/lyra/triggers";
import { type xy } from "@synnaxlabs/x";
export interface UseTriggersProps {
    onUndo?: () => void;
    onRedo?: () => void;
    onCopy?: (cursor: xy.XY) => void;
    onPaste?: (cursor: xy.XY) => void;
    onClearSelection?: () => void;
    onSelectAll?: () => void;
    onGroup?: () => void;
    onUngroup?: () => void;
    enabled?: Triggers.Condition;
    /** Withholds the shortcuts that change the diagram. Copying and the selection
     * shortcuts stay live, so a read-only diagram is still navigable. Defaults to true.
     * */
    editable?: boolean;
}
export declare const useTriggers: ({ onCopy, onPaste, onClearSelection: onClear, onSelectAll, onUndo, onRedo, onGroup, onUngroup, enabled, editable, }: UseTriggersProps) => void;
//# sourceMappingURL=useTriggers.d.ts.map