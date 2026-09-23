import { table } from "@synnaxlabs/client";
import { type ClipboardEventHandler } from "react";
export interface UseClipboardParams {
    key: table.Key;
    selected?: string[];
    onPaste?: (overwrittenKeys: string[]) => void;
}
export interface UseClipboardReturn {
    onCopy: ClipboardEventHandler;
    onPaste: ClipboardEventHandler;
}
export declare const useClipboard: ({ key, selected, onPaste, }: UseClipboardParams) => UseClipboardReturn;
//# sourceMappingURL=clipboard.d.ts.map