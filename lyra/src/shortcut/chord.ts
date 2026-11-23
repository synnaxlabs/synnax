// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { runtime } from "@synnaxlabs/x";

/**
 * One key plus the modifiers held with it. Mod is ⌘ on macOS and Ctrl elsewhere.
 * Control is ⌃, so a chord that holds it matches only on macOS.
 */
export interface Chord {
  key: string;
  mod: boolean;
  control: boolean;
  alt: boolean;
  shift: boolean;
}

/** One chord, or chords pressed one after another. */
export type Binding = Chord[];

const MODIFIERS = ["Mod", "Control", "Alt", "Shift"] as const;
type Modifier = (typeof MODIFIERS)[number];
const MODIFIER_FIELDS: Record<Modifier, keyof Omit<Chord, "key">> = {
  Mod: "mod",
  Control: "control",
  Alt: "alt",
  Shift: "shift",
};

const FUNCTION_KEYS = Array.from({ length: 24 }, (_, i) => `F${i + 1}`);

const NAMED_KEYS = new Set([
  "Enter",
  "Escape",
  "Tab",
  "Space",
  "Backspace",
  "Delete",
  "Insert",
  "Home",
  "End",
  "PageUp",
  "PageDown",
  "ArrowUp",
  "ArrowDown",
  "ArrowLeft",
  "ArrowRight",
  ...FUNCTION_KEYS,
]);

const CODE_PUNCTUATION: Record<string, string> = {
  Backquote: "`",
  Minus: "-",
  Equal: "=",
  BracketLeft: "[",
  BracketRight: "]",
  Backslash: "\\",
  Semicolon: ";",
  Quote: "'",
  Comma: ",",
  Period: ".",
  Slash: "/",
};

const isLetter = (key: string): boolean => /^[A-Z]$/.test(key);

// Printable ASCII, less the space and the "+" that joins a chord.
const isCharacter = (key: string): boolean => /^[!-*,-~]$/.test(key);

const parseChord = (binding: string, text: string): Chord => {
  const parts = text.split("+");
  const key = parts.pop() ?? "";
  const chord: Chord = {
    key: "",
    mod: false,
    control: false,
    alt: false,
    shift: false,
  };
  for (const part of parts) {
    if (!MODIFIERS.includes(part as Modifier))
      throw new Error(`Shortcut "${binding}" has an unknown modifier "${part}"`);
    const field = MODIFIER_FIELDS[part as Modifier];
    if (chord[field]) throw new Error(`Shortcut "${binding}" repeats "${part}"`);
    chord[field] = true;
  }
  const upper = key.toUpperCase();
  if (isCharacter(upper)) chord.key = upper;
  else if (NAMED_KEYS.has(key)) chord.key = key;
  else throw new Error(`Shortcut "${binding}" has an unknown key "${key}"`);
  return chord;
};

/**
 * Parses a binding such as "Mod+Shift+P", or a sequence such as "G G". Modifiers are
 * Mod, Control, Alt, and Shift. A key is a printable character or a named key such as
 * Enter, Escape, ArrowUp, or F3.
 * @throws when the binding names an unknown modifier or key.
 */
export const parse = (binding: string): Binding =>
  binding
    .split(" ")
    .filter((text) => text.length > 0)
    .map((text) => parseChord(binding, text));

const formatChord = (chord: Chord): string =>
  [...MODIFIERS.filter((m) => chord[MODIFIER_FIELDS[m]]), chord.key].join("+");

/** @returns the canonical text of a binding, with modifiers in a fixed order. */
export const format = (binding: Binding): string => binding.map(formatChord).join(" ");

const isMac = (os?: runtime.OS): boolean => os !== "Windows" && os !== "Linux";

const typedKey = (key: string): string | null => {
  if (key === " ") return "Space";
  const upper = key.toUpperCase();
  if (isCharacter(upper)) return upper;
  return NAMED_KEYS.has(key) ? key : null;
};

// The key at the event's position on a US keyboard.
const positionKey = (code: string): string | null => {
  if (/^Key[A-Z]$/.test(code)) return code.slice(3);
  if (/^Digit[0-9]$/.test(code)) return code.slice(5);
  return CODE_PUNCTUATION[code] ?? null;
};

/**
 * @returns the chords a keydown can stand for, best match first. A letter keeps the
 * layout's character, so Mod+Z follows the Z on an AZERTY keyboard. The US position
 * stands in when the character is not ASCII, when Shift typed a symbol, and on the
 * digit row. A symbol that Shift typed, such as "?", also matches without Shift.
 */
export const eventChords = (e: KeyboardEvent, os: runtime.OS | undefined): Chord[] => {
  // The input method owns a key while it composes. Safari marks the first such key
  // only with the keyCode.
  // eslint-disable-next-line @typescript-eslint/no-deprecated
  if (e.isComposing || e.keyCode === 229) return [];
  if (e.getModifierState("AltGraph")) return [];
  const mac = isMac(os);
  if (e.metaKey && !mac) return [];
  const mods = {
    mod: mac ? e.metaKey : e.ctrlKey,
    control: mac && e.ctrlKey,
    alt: e.altKey,
    shift: e.shiftKey,
  };
  const typed = typedKey(e.key);
  const position = positionKey(e.code);
  const typedSymbol = typed != null && typed.length === 1 && !isLetter(typed);
  const chords: Chord[] = [];
  if (typed != null) {
    chords.push({ ...mods, key: typed });
    if (e.shiftKey && typedSymbol && typed !== position)
      chords.push({ ...mods, key: typed, shift: false });
  }
  const standsIn =
    typed == null || (e.shiftKey && typedSymbol) || e.code.startsWith("Digit");
  if (position != null && position !== typed && standsIn)
    chords.push({ ...mods, key: position });
  return chords;
};
