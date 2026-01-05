// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { compare, type xy } from "@synnaxlabs/x";
import { z } from "zod";

import { useMemoCompare } from "@/memo";

/** All the mouse keys that can be used in a trigger */
export const MOUSE_KEYS = ["MouseLeft", "MouseMiddle", "MouseRight"] as const;
export const MOUSE_LEFT_NUMBER = 0;
export const MOUSE_MIDDLE_NUMBER = 1;
export const MOUSE_RIGHT_NUMBER = 2;

export const mouseKeyZ = z.enum(MOUSE_KEYS);
export type MouseKey = z.infer<typeof mouseKeyZ>;

/** A list of all the alphanumeric keys that can be used in a trigger i.e. 0-9, A-Z */
export const ALPHANUMERIC_KEYS = [
  "0",
  "1",
  "2",
  "3",
  "4",
  "5",
  "6",
  "7",
  "8",
  "9",
  "A",
  "B",
  "C",
  "D",
  "E",
  "F",
  "G",
  "H",
  "I",
  "J",
  "K",
  "L",
  "M",
  "N",
  "O",
  "P",
  "Q",
  "R",
  "S",
  "T",
  "U",
  "V",
  "W",
  "X",
  "Y",
  "Z",
] as const;

export const ALPHANUMERIC_KEYS_SET = new Set(ALPHANUMERIC_KEYS);

export type AlphanumericKey = (typeof ALPHANUMERIC_KEYS)[number];

export const isAlphanumericKey = (key: Key): key is AlphanumericKey =>
  ALPHANUMERIC_KEYS_SET.has(key as AlphanumericKey);

/** The set of all possible keyboard and mouse inputs that can be used in a trigger */
export const KEYS = [
  ...MOUSE_KEYS,
  ...ALPHANUMERIC_KEYS,
  "Backspace",
  "Tab",
  "Enter",
  "Shift",
  "Control",
  "Alt",
  "CapsLock",
  "Escape",
  "Space",
  "PageUp",
  "PageDown",
  "End",
  "Home",
  "ArrowLeft",
  "ArrowUp",
  "ArrowRight",
  "ArrowDown",
  "Insert",
  "Delete",
  "ContextMenu",
  "F1",
  "F2",
  "F3",
  "F4",
  "F5",
  "F6",
  "F7",
  "F8",
  "F9",
  "F10",
  "F11",
  "F12",
  "F13",
  "F14",
  "F15",
  "F16",
  "F17",
  "F18",
  "F19",
  "F20",
  "F21",
  "F22",
  "F23",
  "F24",
  "NumLock",
  "ScrollLock",
  "AudioVolumeMute",
  "AudioVolumeDown",
  "AudioVolumeUp",
  "AudioTrackNext",
  "AudioTrackPrevious",
  "AudioStop",
  "AudioPlay",
  "AudioPause",
  "AudioRewind",
  "AudioForward",
  "AudioRepeat",
  "AudioRandomPlay",
  "AudioSelect",
  "LaunchMail",
  "LaunchApp2",
  "LaunchApp1",
  "SelectTask",
  "LaunchScreenSaver",
  "BrowserSearch",
  "BrowserHome",
  "BrowserBack",
  "BrowserForward",
  "BrowserStop",
  "BrowserRefresh",
  "BrowserFavorites",
  "ZoomToggle",
  "Clear",
  "Power",
  "Eject",
] as const;

export const keyZ = z.enum(KEYS);
/**
 * A union of literal string types representing all possible keyboard and mouse inputs
 * that can be used in a trigger. This includes all values from {@link KEYS} array.
 */
export type Key = (typeof KEYS)[number];

export const triggerZ = z.array(keyZ);
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

/**
 * An event fired when a trigger is activated in Triggers.use.
 */
export interface Event {
  /** The target element that the trigger was fired on. */
  target: HTMLElement;
  /** Previously held triggers that are now released. */
  prev: Trigger[];
  /** Triggers that were not previously held that are now activated. */
  next: Trigger[];
  /** The current cursor position. */
  cursor: xy.XY;
}

/** A callback that is fired when a trigger is activated. */
export type Callback = (e: Event) => void;

/** Parses the TriggerKey from the provided KeyboardEvent or MouseEvent. */
export const eventKey = (
  e:
    | KeyboardEvent
    | MouseEvent
    | PointerEvent
    | React.KeyboardEvent
    | React.MouseEvent
    | React.PointerEvent,
): Key => {
  if (e.type.includes("key")) return keyboardKey(e as KeyboardEvent);
  return mouseKey((e as MouseEvent).button);
};

/* Tracks a list of keys that have an opinionated location i.e. "Left"  or "Right"
 as Triggers is location agnostic. */
const INCLUDES_KEYS: Key[] = ["Control", "Alt", "Shift"];

/**
 * Parses the TriggerKey from the provided KeyboardEvent.
 * @param e - The KeyboardEvent to parse.
 * @returns the TriggerKey.
 */
export const keyboardKey = (
  e: KeyboardEvent | React.KeyboardEvent<HTMLElement>,
): Key => {
  if (["Digit", "Key"].some((k) => e.code.startsWith(k)))
    return e.code.slice(-1) as Key;
  if (e.code.includes("Meta")) return "Control";
  const includeKey = INCLUDES_KEYS.find((k) => e.code.includes(k));
  if (includeKey != null) return includeKey;
  return e.code as Key;
};

const MOUSE_BUTTONS: Record<number, MouseKey> = {
  0: "MouseLeft",
  1: "MouseMiddle",
  2: "MouseRight",
};

/**
 * Converts a mouse button number to a TriggerKey.
 * @param button - The mouse button number.
 * @returns the TriggerKey.
 */
export const mouseKey = (button: number): Key => MOUSE_BUTTONS[button] ?? "MouseLeft";

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
 * Match compares the expected triggers against the actual triggers.
 *
 * @param expected - The reference triggers to match the actual triggers against.
 * @param actual - The actual triggers that were fired.
 * @param loose - If true, triggers in actual that are a superset of those in expected
 * will still be considered a match i.e. if expected is [["Control"]] and actual is
 * [["Control", "A"]], then match will return true.
 * @returns true if any triggers in expected match those in actual.
 *
 */
export const match = (
  expected: Trigger[],
  actual: Trigger[],
  opts?: MatchOptions,
): boolean => filter(expected, actual, opts).length > 0;

export const matchCallback =
  <E extends KeyboardEvent | MouseEvent | React.KeyboardEvent | React.MouseEvent>(
    expect: Trigger[],
    callback: (e: E) => void,
  ): ((e: E) => void) =>
  (e) => {
    if (match(expect, [[eventKey(e)]])) return callback(e);
  };

/**
 * Filter compares the expected triggers against the actual triggers and returns
 * an array of triggers in expected that match those in actual.
 *
 * @param expected - The reference triggers to match the actual triggers against.
 * @param actual - The actual triggers that were fired.
 * @param loose - If true, triggers in actual that are a superset of those in expected
 * will still be considered a match i.e. if expected is [["Control"]] and actual is
 * [["Control", "A"]], then filter will return [["Control"]].
 */
export const filter = (
  expected: Trigger[],
  actual: Trigger[],
  opts?: MatchOptions,
): Trigger[] => {
  const f = compareF(opts);
  return expected.filter((o) => actual.some((t) => f(o, t) === compare.EQUAL));
};

/**
 * Removes all triggers from the source that strongly match those in toPurge.
 *
 * @param source - The source triggers to purge from.
 * @param toPurge - The triggers to purge from the source.
 * @returns the source triggers with all triggers in toPurge removed.
 */
export const purge = (source: Trigger[], toPurge: Trigger[]): Trigger[] =>
  source.filter(
    (t) =>
      !toPurge.some((t2) => compare.unorderedPrimitiveArrays(t, t2) === compare.EQUAL),
  );

/**
 *  Finds the difference between two sets of triggers.
 *
 * @param a - The first set of triggers.
 * @param b - The second set of triggers.
 * @returns a tuple, where the first element is the triggers in a that are not in b,
 * and the second element is the triggers in b that are not in a.
 */
export const diff = (a: Trigger[], b: Trigger[]): [Trigger[], Trigger[]] => {
  const f = compareF();
  const added = a.filter((ta) => !b.some((tb) => f(ta, tb) === compare.EQUAL));
  const removed = b.filter((tb) => !a.some((ta) => f(tb, ta) === compare.EQUAL));
  return [added, removed];
};

/**
 * Determines if two triggers are semantically equal.
 * @param loose - If true, if the second trigger is a superset of the first, then
 * the triggers will be considered equal.
 * @returns a comparison function that determines if two triggers are semantically equal.
 */
const compareF = (opts?: MatchOptions): compare.Comparator<Trigger> => {
  if (opts?.loose === true) return _looseCompare;
  if (opts?.double === true) return compare.uniqueUnorderedPrimitiveArrays;
  return compare.unorderedPrimitiveArrays;
};

const _looseCompare: compare.Comparator<Trigger> = (a, b) =>
  a.every((k) => b.includes(k)) ? compare.EQUAL : compare.LESS_THAN;

/** ModeConfig is a mapping of modes to triggers along with a default mode. */
export type ModeConfig<M extends string | number | symbol> = Record<M, Trigger[]> & {
  defaultMode: M;
};

/**
 * DetermineMode determines the mode that should be used given the provided triggers.
 * It's important to note that this object uses Object.entries to iterate over the
 * config, so the order of modes is guaranteed by the insertion order of modes into
 * the config object, or, in the case of numeric keys, the order of the keys.
 *
 * @param config - The mode config to use.
 * @param triggers - The triggers to use to determine the mode.
 * @param loose - If true, if the triggers are a superset of the triggers in a mode,
 * then that mode will be used.
 */
export const determineMode = <K extends string | number | symbol>(
  config: ModeConfig<K>,
  triggers: Trigger[],
  opts?: MatchOptions,
): K => {
  const e = Object.entries(config).filter(
    ([k]) => k !== "defaultMode",
  ) as unknown as Array<[K, Trigger[]]>;
  const flat = e.map(([k, v]) => v.map((t) => [k, t])).flat() as Array<[K, Trigger]>;
  const complexitySorted = flat.sort(([, a], [, b]) => b.length - a.length);
  const match_ = complexitySorted.find(([, v]) => match([v], triggers, opts));
  if (match_ != null) return match_[0];
  return config.defaultMode;
};

/**
 * A useMemoCompare function that compares two ModeConfigs.
 * @returns true if the two ModeConfigs are equal.
 */
export const compareModeConfigs = <K extends string | number | symbol>(
  [a]: Array<ModeConfig<K> | undefined | null>,
  [b]: Array<ModeConfig<K> | undefined | null>,
): boolean => {
  if (a == null && b == null) return true;
  if (a == null || b == null) return false;
  if (a.defaultMode !== b.defaultMode) return false;
  const aKeys = Object.keys(a) as K[];
  const bKeys = Object.keys(b) as K[];
  if (aKeys.length !== bKeys.length) return false;
  if (a.defaultMode !== b.defaultMode) return false;
  return aKeys.every((k) => compare.unorderedPrimitiveArrays(a[k], b[k]) === 0);
};

/**
 * Flattens the given ModeConfig into a list of triggers, excluding the default mode.
 * @param config - The ModeConfig to flatten.
 * @returns a list of triggers.
 */
export const flattenConfig = <K extends string | number | symbol>(
  config: ModeConfig<K>,
): Trigger[] => {
  const e = Object.entries(config).filter(
    ([k]) => k !== "defaultMode",
  ) as unknown as Array<[K, Trigger[]]>;
  return e.map(([, v]) => v).flat();
};

/**
 * @returns a memoized flattened config, only recomputing when the config changes.
 */
export const useFlattenedMemoConfig = <K extends string | number | symbol>(
  config: ModeConfig<K>,
): Trigger[] =>
  useMemoCompare(() => flattenConfig(config), compareModeConfigs, [config]);

/** Purges all mouse keys from the given triggers. If the resulting trigger is empty,
 * it will be removed from the list of triggers. */
export const purgeMouse = (triggers: Trigger[]): Trigger[] =>
  triggers
    .map((t) => t.filter((k) => !k.startsWith("Mouse")))
    .filter((t) => t.length > 0);
