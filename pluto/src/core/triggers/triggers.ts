// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Compare, CompareF, XY } from "@synnaxlabs/x";

export const MOUSE_KEYS = ["MouseLeft", "MouseMiddle", "MouseRight"] as const;

export type MouseKey = (typeof MOUSE_KEYS)[number];

export const KEYS = [
  ...MOUSE_KEYS,
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
  "Meta",
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

export type Key = (typeof KEYS)[number];

export type Trigger = Key[];

export type Stage = "start" | "during" | "end";

export interface TriggerEvent {
  target: HTMLElement;
  prev: Trigger[];
  next: Trigger[];
  cursor: XY;
}

export type TriggerCallback = (e: TriggerEvent) => void;

export const parseEventKey = (e: KeyboardEvent | MouseEvent): Key =>
  e instanceof KeyboardEvent ? keyboardToKey(e) : mouseButtonToKey(e.button);

// Tracks a list of keys that have an opinionated location i.e. "Left"  or "Right"
// as Triggers is location agnostic.
const INCLUDES_KEYS: Key[] = ["Meta", "Control", "Alt", "Shift"];

export const keyboardToKey = (e: KeyboardEvent): Key => {
  if (["Digit", "Key"].some((k) => e.code.startsWith(k)))
    return e.code.slice(-1) as Key;
  const includeKey = INCLUDES_KEYS.find((k) => e.code.includes(k));
  if (includeKey != null) return includeKey;
  return e.code as Key;
};

export const mouseButtonToKey = (button: number): Key => {
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
  return options.filter((o) => triggers.some((t) => f(o, t) === 0));
};

export const purge = (source: Trigger[], toPurge: Trigger[]): Trigger[] =>
  source.filter(
    (t) => !toPurge.some((t2) => Compare.unorderedPrimitiveArrays(t, t2) == 0)
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
        const aCounts: Record<Key[number], number> = {};
        a.forEach((k) => (aCounts[k] = (aCounts[k] ?? 0) + 1));
        const bCounts: Record<Key[number], number> = {};
        b.forEach((k) => (bCounts[k] = (bCounts[k] ?? 0) + 1));
        return a.every((k) => (aCounts[k] = bCounts[k])) ? 0 : -1;
      }
    : Compare.unorderedPrimitiveArrays;
