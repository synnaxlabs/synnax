export declare const preventDefault: (e: {
    preventDefault: () => void;
}) => void;
export declare const stopPropagation: (e: {
    stopPropagation: () => void;
}) => void;
/** The keys whose keydown or keyup the browser turns into a click. */
export declare const ACTIVATION_KEYS: string[];
/**
 * Cancels the synthetic click for Space and Enter. Attach to both keydown and keyup:
 * Enter clicks on keydown, Space on keyup.
 */
export declare const blockActivation: (e: {
    key: string;
    preventDefault: () => void;
}) => void;
/** True when the target accepts text entry, so the keystroke belongs to it. */
export declare const isInputOrContentEditable: (e: {
    target: EventTarget | null;
}) => boolean;
//# sourceMappingURL=event.d.ts.map