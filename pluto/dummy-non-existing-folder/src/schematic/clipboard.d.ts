import { type RefObject } from "react";
import { Diagram } from "../vis/diagram";
export interface UseClipboardParams {
    selected?: string[];
    onCut?: (remaining: string[]) => void;
    onPaste?: (newKeys: string[]) => void;
    container?: RefObject<HTMLDivElement | null>;
}
export declare const useClipboard: ({ selected, onCut, onPaste, container, }: UseClipboardParams) => Diagram.UseClipboardReturn;
//# sourceMappingURL=clipboard.d.ts.map