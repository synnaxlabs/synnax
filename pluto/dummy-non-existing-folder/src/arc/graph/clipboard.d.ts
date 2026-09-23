import { arc } from "@synnaxlabs/client";
import { type RefObject } from "react";
import { Diagram } from "../../vis/diagram";
export interface UseClipboardParams {
    key: arc.Key;
    selected?: string[];
    onCut?: (remaining: string[]) => void;
    onPaste?: (newKeys: string[]) => void;
    container?: RefObject<HTMLDivElement | null>;
}
export declare const useClipboard: ({ key, selected, onCut, onPaste, container, }: UseClipboardParams) => Diagram.UseClipboardReturn;
//# sourceMappingURL=clipboard.d.ts.map