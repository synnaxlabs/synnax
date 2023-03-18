// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Compare, XY } from "@synnaxlabs/x";

export const KEYS = [
  "MouseMiddle",
  "MouseLeft",
  "MouseRight",
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

export type Key = typeof KEYS[number];

export type Trigger = Key[];

export type Stage = "start" | "during" | "end";

export interface TriggerEvent {
  target: HTMLElement;
  triggers: Trigger[];
  stage: Stage;
  cursor: XY;
}

export type TriggerCallback = (e: TriggerEvent) => void;

export const parseEventKey = (e: KeyboardEvent | MouseEvent): Key =>
  e instanceof KeyboardEvent ? keyboardToKey(e.key) : mouseButtonToKey(e.button);

export const keyboardToKey = (key: string): Key =>
  (key[0].toUpperCase() + key.slice(1)) as Key;

export const mouseButtonToKey = (button: number): Key => {
  if (button === 1) return "MouseMiddle";
  if (button === 2) return "MouseRight";
  return "MouseLeft";
};

export const match = (options: Trigger[], triggers: Trigger[]): boolean =>
  options.some((o) =>
    triggers.some((t) => Compare.unorderedPrimitiveArrays(o, t) === 0)
  );
