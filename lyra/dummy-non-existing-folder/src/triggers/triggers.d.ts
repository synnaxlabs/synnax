import { type xy } from "@synnaxlabs/x";
import { z } from "zod";
/** All the mouse keys that can be used in a trigger */
export declare const MOUSE_KEYS: readonly ["MouseLeft", "MouseMiddle", "MouseRight"];
export declare const MOUSE_LEFT_NUMBER = 0;
export declare const MOUSE_MIDDLE_NUMBER = 1;
export declare const MOUSE_RIGHT_NUMBER = 2;
/** Zod schema for {@link MouseKey}. */
export declare const mouseKeyZ: z.ZodEnum<{
    MouseLeft: "MouseLeft";
    MouseMiddle: "MouseMiddle";
    MouseRight: "MouseRight";
}>;
export type MouseKey = z.infer<typeof mouseKeyZ>;
/** A list of all the alphanumeric keys that can be used in a trigger i.e. 0-9, A-Z */
export declare const ALPHANUMERIC_KEYS: readonly ["0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"];
export declare const ALPHANUMERIC_KEYS_SET: Set<"0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" | "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" | "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z">;
export type AlphanumericKey = (typeof ALPHANUMERIC_KEYS)[number];
/** @returns whether the key is a letter or a digit. */
export declare const isAlphanumericKey: (key: Key) => key is AlphanumericKey;
/** @returns whether pressing the key types a character. */
export declare const isTextEntryKey: (key: Key) => boolean;
/** The set of all possible keyboard and mouse inputs that can be used in a trigger */
export declare const KEYS: readonly ["MouseLeft", "MouseMiddle", "MouseRight", "0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z", "Backspace", "Tab", "Enter", "Shift", "Control", "Alt", "CapsLock", "Escape", "Space", "Equal", "Minus", "PageUp", "PageDown", "End", "Home", "ArrowLeft", "ArrowUp", "ArrowRight", "ArrowDown", "Insert", "Delete", "ContextMenu", "F1", "F2", "F3", "F4", "F5", "F6", "F7", "F8", "F9", "F10", "F11", "F12", "F13", "F14", "F15", "F16", "F17", "F18", "F19", "F20", "F21", "F22", "F23", "F24", "NumLock", "ScrollLock", "AudioVolumeMute", "AudioVolumeDown", "AudioVolumeUp", "AudioTrackNext", "AudioTrackPrevious", "AudioStop", "AudioPlay", "AudioPause", "AudioRewind", "AudioForward", "AudioRepeat", "AudioRandomPlay", "AudioSelect", "LaunchMail", "LaunchApp2", "LaunchApp1", "SelectTask", "LaunchScreenSaver", "BrowserSearch", "BrowserHome", "BrowserBack", "BrowserForward", "BrowserStop", "BrowserRefresh", "BrowserFavorites", "ZoomToggle", "Clear", "Power", "Eject"];
/** Zod schema for {@link Key}. */
export declare const keyZ: z.ZodEnum<{
    0: "0";
    1: "1";
    2: "2";
    3: "3";
    4: "4";
    5: "5";
    6: "6";
    7: "7";
    8: "8";
    9: "9";
    A: "A";
    Alt: "Alt";
    ArrowDown: "ArrowDown";
    ArrowLeft: "ArrowLeft";
    ArrowRight: "ArrowRight";
    ArrowUp: "ArrowUp";
    AudioForward: "AudioForward";
    AudioPause: "AudioPause";
    AudioPlay: "AudioPlay";
    AudioRandomPlay: "AudioRandomPlay";
    AudioRepeat: "AudioRepeat";
    AudioRewind: "AudioRewind";
    AudioSelect: "AudioSelect";
    AudioStop: "AudioStop";
    AudioTrackNext: "AudioTrackNext";
    AudioTrackPrevious: "AudioTrackPrevious";
    AudioVolumeDown: "AudioVolumeDown";
    AudioVolumeMute: "AudioVolumeMute";
    AudioVolumeUp: "AudioVolumeUp";
    B: "B";
    Backspace: "Backspace";
    BrowserBack: "BrowserBack";
    BrowserFavorites: "BrowserFavorites";
    BrowserForward: "BrowserForward";
    BrowserHome: "BrowserHome";
    BrowserRefresh: "BrowserRefresh";
    BrowserSearch: "BrowserSearch";
    BrowserStop: "BrowserStop";
    C: "C";
    CapsLock: "CapsLock";
    Clear: "Clear";
    ContextMenu: "ContextMenu";
    Control: "Control";
    D: "D";
    Delete: "Delete";
    E: "E";
    Eject: "Eject";
    End: "End";
    Enter: "Enter";
    Equal: "Equal";
    Escape: "Escape";
    F: "F";
    F1: "F1";
    F10: "F10";
    F11: "F11";
    F12: "F12";
    F13: "F13";
    F14: "F14";
    F15: "F15";
    F16: "F16";
    F17: "F17";
    F18: "F18";
    F19: "F19";
    F2: "F2";
    F20: "F20";
    F21: "F21";
    F22: "F22";
    F23: "F23";
    F24: "F24";
    F3: "F3";
    F4: "F4";
    F5: "F5";
    F6: "F6";
    F7: "F7";
    F8: "F8";
    F9: "F9";
    G: "G";
    H: "H";
    Home: "Home";
    I: "I";
    Insert: "Insert";
    J: "J";
    K: "K";
    L: "L";
    LaunchApp1: "LaunchApp1";
    LaunchApp2: "LaunchApp2";
    LaunchMail: "LaunchMail";
    LaunchScreenSaver: "LaunchScreenSaver";
    M: "M";
    Minus: "Minus";
    MouseLeft: "MouseLeft";
    MouseMiddle: "MouseMiddle";
    MouseRight: "MouseRight";
    N: "N";
    NumLock: "NumLock";
    O: "O";
    P: "P";
    PageDown: "PageDown";
    PageUp: "PageUp";
    Power: "Power";
    Q: "Q";
    R: "R";
    S: "S";
    ScrollLock: "ScrollLock";
    SelectTask: "SelectTask";
    Shift: "Shift";
    Space: "Space";
    T: "T";
    Tab: "Tab";
    U: "U";
    V: "V";
    W: "W";
    X: "X";
    Y: "Y";
    Z: "Z";
    ZoomToggle: "ZoomToggle";
}>;
/**
 * A union of literal string types representing all possible keyboard and mouse inputs
 * that can be used in a trigger. This includes all values from {@link KEYS} array.
 */
export type Key = (typeof KEYS)[number];
/** Zod schema for {@link Trigger}. */
export declare const triggerZ: z.ZodArray<z.ZodEnum<{
    0: "0";
    1: "1";
    2: "2";
    3: "3";
    4: "4";
    5: "5";
    6: "6";
    7: "7";
    8: "8";
    9: "9";
    A: "A";
    Alt: "Alt";
    ArrowDown: "ArrowDown";
    ArrowLeft: "ArrowLeft";
    ArrowRight: "ArrowRight";
    ArrowUp: "ArrowUp";
    AudioForward: "AudioForward";
    AudioPause: "AudioPause";
    AudioPlay: "AudioPlay";
    AudioRandomPlay: "AudioRandomPlay";
    AudioRepeat: "AudioRepeat";
    AudioRewind: "AudioRewind";
    AudioSelect: "AudioSelect";
    AudioStop: "AudioStop";
    AudioTrackNext: "AudioTrackNext";
    AudioTrackPrevious: "AudioTrackPrevious";
    AudioVolumeDown: "AudioVolumeDown";
    AudioVolumeMute: "AudioVolumeMute";
    AudioVolumeUp: "AudioVolumeUp";
    B: "B";
    Backspace: "Backspace";
    BrowserBack: "BrowserBack";
    BrowserFavorites: "BrowserFavorites";
    BrowserForward: "BrowserForward";
    BrowserHome: "BrowserHome";
    BrowserRefresh: "BrowserRefresh";
    BrowserSearch: "BrowserSearch";
    BrowserStop: "BrowserStop";
    C: "C";
    CapsLock: "CapsLock";
    Clear: "Clear";
    ContextMenu: "ContextMenu";
    Control: "Control";
    D: "D";
    Delete: "Delete";
    E: "E";
    Eject: "Eject";
    End: "End";
    Enter: "Enter";
    Equal: "Equal";
    Escape: "Escape";
    F: "F";
    F1: "F1";
    F10: "F10";
    F11: "F11";
    F12: "F12";
    F13: "F13";
    F14: "F14";
    F15: "F15";
    F16: "F16";
    F17: "F17";
    F18: "F18";
    F19: "F19";
    F2: "F2";
    F20: "F20";
    F21: "F21";
    F22: "F22";
    F23: "F23";
    F24: "F24";
    F3: "F3";
    F4: "F4";
    F5: "F5";
    F6: "F6";
    F7: "F7";
    F8: "F8";
    F9: "F9";
    G: "G";
    H: "H";
    Home: "Home";
    I: "I";
    Insert: "Insert";
    J: "J";
    K: "K";
    L: "L";
    LaunchApp1: "LaunchApp1";
    LaunchApp2: "LaunchApp2";
    LaunchMail: "LaunchMail";
    LaunchScreenSaver: "LaunchScreenSaver";
    M: "M";
    Minus: "Minus";
    MouseLeft: "MouseLeft";
    MouseMiddle: "MouseMiddle";
    MouseRight: "MouseRight";
    N: "N";
    NumLock: "NumLock";
    O: "O";
    P: "P";
    PageDown: "PageDown";
    PageUp: "PageUp";
    Power: "Power";
    Q: "Q";
    R: "R";
    S: "S";
    ScrollLock: "ScrollLock";
    SelectTask: "SelectTask";
    Shift: "Shift";
    Space: "Space";
    T: "T";
    Tab: "Tab";
    U: "U";
    V: "V";
    W: "W";
    X: "X";
    Y: "Y";
    Z: "Z";
    ZoomToggle: "ZoomToggle";
}>>;
/**
 * A sequence of unordered keyboard and mouse inputs that can be used to fire
 * a trigger. Repeated keys that represent double presses are allowed, but must be
 * placed next to each other.
 */
export type Trigger = Key[];
/**
 * The stage of a trigger. The 'start' event fires when the trigger is first activated.
 * The 'during' state is only fired in hooks that track cursor movement. The 'end' state
 * is fired when the trigger is released.
 */
export type Stage = "start" | "during" | "end";
/** An event fired when a trigger is activated in Triggers.use. */
export interface Event {
    /** The target element that the trigger was fired on. */
    target: HTMLElement;
    /** Previously held triggers that are now released. */
    prev: Trigger[];
    /** Triggers that were not previously held that are now activated. */
    next: Trigger[];
    /** The current cursor position. */
    cursor: xy.XY;
    /**
     * Prevents the event from being dispatched to any remaining subscribers with a
     * lower priority than the current subscriber. Subscribers at the same priority
     * level that have not yet been notified still receive the event.
     */
    stopPropagation: () => void;
}
/** A callback that is fired when a trigger is activated. */
export type Callback = (e: Event) => void;
/** Parses the TriggerKey from the provided KeyboardEvent or MouseEvent. */
export declare const eventKey: (e: KeyboardEvent | MouseEvent | PointerEvent | React.KeyboardEvent | React.MouseEvent | React.PointerEvent) => Key;
export declare const MODIFIER_KEYS: Key[];
/**
 * Parses the TriggerKey from the provided KeyboardEvent.
 * @returns the TriggerKey.
 */
export declare const keyboardKey: (e: KeyboardEvent | React.KeyboardEvent<HTMLElement>) => Key;
/**
 * Converts a mouse button number to a TriggerKey.
 * @returns the TriggerKey.
 */
export declare const mouseKey: (button: number) => Key;
export interface MatchOptions {
    /**
     * If true, triggers in actual that are a superset of those in expected will still be
     * considered a match i.e. if expected is [["Control"]] and actual is [["Control", "A"]],
     * then match will return true.
     */
    loose?: boolean;
    /**
     * If true, triggers in actual that are a double press of those in expected will still
     * be considered a match i.e. if expected is [["Control", "W"]] and actual is
     * [["Control", "W", "W"]], then match will return true.
     */
    double?: boolean;
}
/**
 * @param expected - The reference triggers to match the actual triggers against.
 * @param actual - The actual triggers that were fired.
 * @param loose - If true, triggers in actual that are a superset of those in expected
 * will still be considered a match i.e. if expected is [["Control"]] and actual is
 * [["Control", "A"]], then match will return true.
 * @returns true if any triggers in expected match those in actual.
 */
export declare const match: (expected: Trigger[], actual: Trigger[], opts?: MatchOptions) => boolean;
/** Wraps a handler so it runs only when the event's key is one of the expected ones. */
export declare const matchCallback: <E extends KeyboardEvent | MouseEvent | React.KeyboardEvent | React.MouseEvent>(expect: Trigger[], callback: (e: E) => void) => ((e: E) => void);
/**
 * Filter compares the expected triggers against the actual triggers and returns an
 * array of triggers in expected that match those in actual.
 * @param actual - The actual triggers that were fired.
 * @param loose - If true, triggers in actual that are a superset of those in expected
 * will still be considered a match i.e. if expected is [["Control"]] and actual is
 * [["Control", "A"]], then filter will return [["Control"]].
 */
export declare const filter: (expected: Trigger[], actual: Trigger[], opts?: MatchOptions) => Trigger[];
/**
 * Removes all triggers from the source that strongly match those in toPurge.
 * @returns the source triggers with all triggers in toPurge removed.
 */
export declare const purge: (source: Trigger[], toPurge: Trigger[]) => Trigger[];
/**
 * Finds the difference between two sets of triggers.
 * @returns a tuple, where the first element is the triggers in a that are not in b, and
 * the second element is the triggers in b that are not in a.
 */
export declare const diff: (a: Trigger[], b: Trigger[]) => [Trigger[], Trigger[]];
/** ModeConfig is a mapping of modes to triggers along with a default mode. */
export interface ModeConfig<M extends string | number | symbol> {
    /** The mode to fall back on when no mode's triggers match. */
    defaultMode: M;
    /** The triggers that select each mode. */
    modes: Record<M, Trigger[]>;
}
/**
 * DetermineMode determines the mode that should be used given the provided triggers.
 * It's important to note that this object uses Object.entries to iterate over the
 * config, so the order of modes is guaranteed by the insertion order of modes into the
 * config object, or, in the case of numeric keys, the order of the keys.
 * @param loose - If true, if the triggers are a superset of the triggers in a mode,
 * then that mode will be used.
 */
export declare const determineMode: <K extends string | number | symbol>(config: ModeConfig<K>, triggers: Trigger[], opts?: MatchOptions) => K;
/**
 * A useMemoCompare function that compares two ModeConfigs.
 * @returns true if the two ModeConfigs are equal.
 */
export declare const compareModeConfigs: <K extends string | number | symbol>([a]: Array<ModeConfig<K> | undefined | null>, [b]: Array<ModeConfig<K> | undefined | null>) => boolean;
/**
 * Flattens the given ModeConfig into a list of triggers, excluding the default mode.
 * @returns a list of triggers.
 */
export declare const flattenConfig: <K extends string | number | symbol>(config: ModeConfig<K>) => Trigger[];
/** @returns a memoized flattened config, only recomputing when the config changes. */
export declare const useFlattenedMemoConfig: <K extends string | number | symbol>(config: ModeConfig<K>) => Trigger[];
/** Purges all mouse keys from the given triggers. If the resulting trigger is empty,
 * it will be removed from the list of triggers. */
export declare const purgeMouse: (triggers: Trigger[]) => Trigger[];
/** The standard undo shortcut. Command maps to Control, so it covers both platforms. */
export declare const UNDO: Trigger;
/** The standard redo shortcut. */
export declare const REDO: Trigger;
/** The standard cut shortcut. */
export declare const CUT: Trigger;
/** The standard copy shortcut. */
export declare const COPY: Trigger;
/** The standard paste shortcut. */
export declare const PASTE: Trigger;
/** The standard group shortcut. */
export declare const GROUP: Trigger;
/** The standard ungroup shortcut. */
export declare const UNGROUP: Trigger;
/** The escape key on its own. */
export declare const ESCAPE: Trigger;
//# sourceMappingURL=triggers.d.ts.map