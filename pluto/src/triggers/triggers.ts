// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Compare, CompareF, XY } from "@synnaxlabs/x";
import { z } from "zod";

export const MOUSE_TRIGGER_KEYS = ["MouseLeft", "MouseMiddle", "MouseRight"] as const;

export const mouseTriggerKeyZ = z.enum(MOUSE_TRIGGER_KEYS);
export type MouseKeyTrigger = z.infer<typeof mouseTriggerKeyZ>;

export const TRIGGER_KEYS = [
  ...MOUSE_TRIGGER_KEYS,
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

export const triggerKeyZ = z.enum(TRIGGER_KEYS);
export type TriggerKey = z.infer<typeof triggerKeyZ>;

export const triggerZ = z.array(triggerKeyZ);
export type Trigger = z.infer<typeof triggerZ>;

export type Stage = "start" | "during" | "end";

export interface TriggerEvent {
  target: HTMLElement;
  prev: Trigger[];
  next: Trigger[];
  cursor: XY;
}

export type TriggerCallback = (e: TriggerEvent) => void;

export const eventTriggerKey = (e: KeyboardEvent | MouseEvent): TriggerKey =>
  e instanceof KeyboardEvent ? keyboardTriggerKey(e) : mouseButtonTriggerKey(e.button);

// Tracks a list of keys that have an opinionated location i.e. "Left"  or "Right"
// as Triggers is location agnostic.
const INCLUDES_KEYS: TriggerKey[] = ["Control", "Alt", "Shift"];

/**
 * Parses the TriggerKey from the provided KeyboardEvent.
 *
 * @param e - The KeyboardEvent to parse.
 * @returns the TriggerKey.
 */
export const keyboardTriggerKey = (e: KeyboardEvent): TriggerKey => {
  if (["Digit", "Key"].some((k) => e.code.startsWith(k)))
    return e.code.slice(-1) as TriggerKey;
  if (e.code.includes("Meta")) return "Control";
  const includeKey = INCLUDES_KEYS.find((k) => e.code.includes(k));
  if (includeKey != null) return includeKey;
  return e.code as TriggerKey;
};

/**
 * Converts a mouse button number to a TriggerKey.
 *
 * @param button - The mouse button number.
 * @returns the TriggerKey.
 */
export const mouseButtonTriggerKey = (button: number): TriggerKey => {
  if (button === 1) return "MouseMiddle";
  if (button === 2) return "MouseRight";
  return "MouseLeft";
};

export const match = (
  options: Trigger[],
  triggers: Trigger[],
  loose = false
): boolean => filter(options, triggers, loose).length > 0;

export const filter = (
  options: Trigger[],
  triggers: Trigger[],
  loose = false
): Trigger[] => {
  const f = compareF(loose);
  const res = options.filter((o) => triggers.some((t) => f(o, t) === 0));
  return res;
};

export const purge = (source: Trigger[], toPurge: Trigger[]): Trigger[] =>
  source.filter(
    (t) =>
      !toPurge.some((t2) => Compare.unorderedPrimitiveArrays(t, t2) === Compare.EQUAL)
  );

export const diff = (
  a: Trigger[],
  b: Trigger[],
  loose = false
): [Trigger[], Trigger[]] => {
  const f = compareF(loose);
  const added = a.filter((t) => !b.some((t2) => f(t, t2) === 0));
  const removed = b.filter((t) => !a.some((t2) => f(t, t2) === 0));
  return [added, removed];
};

const compareF = (loose: boolean): CompareF<Trigger> =>
  loose
    ? (a: Trigger, b: Trigger) => {
        const aCounts: Record<TriggerKey[number], number> = {};
        a.forEach((k) => (aCounts[k] = (aCounts[k] ?? 0) + 1));
        const bCounts: Record<TriggerKey[number], number> = {};
        b.forEach((k) => (bCounts[k] = (bCounts[k] ?? 0) + 1));
        return a.every((k) => (aCounts[k] = bCounts[k])) ? 0 : -1;
      }
    : Compare.unorderedPrimitiveArrays;

export type TriggerConfig<K extends string | number | symbol> = Record<K, Trigger[]> & {
  defaultMode: K;
};

export const determineTriggerMode = <K extends string | number | symbol>(
  config: TriggerConfig<K>,
  triggers: Trigger[],
  loose = false
): K => {
  const e = Object.entries(config).filter(
    ([k]) => k !== "defaultMode"
  ) as unknown as Array<[K, Trigger[]]>;
  const flat = e.map(([k, v]) => v.map((t) => [k, t])).flat() as Array<[K, Trigger]>;
  const complexitySorted = flat.sort(([, a], [, b]) => b.length - a.length);
  const match_ = complexitySorted.find(([, v]) => match([v], triggers, loose));
  if (match_ != null) return match_[0];
  return config.defaultMode;
};

export const compareTriggerConfigs = <K extends string | number | symbol>(
  [a]: Array<TriggerConfig<K> | undefined | null>,
  [b]: Array<TriggerConfig<K> | undefined | null>
): boolean => {
  if (a == null && b == null) return true;
  if (a == null || b == null) return false;
  if (a.defaultMode !== b.defaultMode) return false;
  const aKeys = Object.keys(a) as K[];
  const bKeys = Object.keys(b) as K[];
  if (aKeys.length !== bKeys.length) return false;
  if (a.defaultMode !== b.defaultMode) return false;
  return aKeys.every((k) => Compare.unorderedPrimitiveArrays(a[k], b[k]) === 0);
};

export const reduceTriggerConfig = <K extends string | number | symbol>(
  config: TriggerConfig<K>
): Trigger[] => {
  const e = Object.entries(config).filter(
    ([k]) => k !== "defaultMode"
  ) as unknown as Array<[K, Trigger[]]>;
  return e.map(([, v]) => v).flat();
};
