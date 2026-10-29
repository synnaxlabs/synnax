// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/triggers/Text.css";

import { runtime } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { type Generic } from "@/generic";
import { Icon } from "@/icon";
import { Text as Base } from "@/text";
import { useReserved } from "@/triggers/Provider";
import { type Key, type Trigger } from "@/triggers/triggers";

/** Props for {@link Text}. */
export type TextProps<E extends Generic.ElementType = "p"> = Base.TextProps<E> & {
  trigger: Trigger;
  /** The platform whose keys to print. Defaults to the platform the page runs on. */
  os?: runtime.OS;
};

// Windows and Linux spell modifiers out. An unknown platform prints macOS symbols.
const isMac = (os?: runtime.OS): boolean => os !== "Windows" && os !== "Linux";

const SHARED_TEXT: Partial<Record<Key, ReactElement | string>> = {
  MouseLeft: <Icon.Click key="mouse" />,
  MouseRight: <Icon.Click key="mouse" />,
  MouseMiddle: <Icon.Click key="mouse" />,
  Enter: <Icon.Keyboard.Return key="enter" />,
  ArrowUp: <Icon.Keyboard.Arrow.Up key="arrow-up" />,
  ArrowDown: <Icon.Keyboard.Arrow.Down key="arrow-down" />,
  ArrowLeft: <Icon.Keyboard.Arrow.Left key="arrow-left" />,
  ArrowRight: <Icon.Keyboard.Arrow.Right key="arrow-right" />,
  Escape: "Esc",
  Delete: "Delete",
  Tab: "Tab",
  Space: "Space",
  CapsLock: "Caps Lock",
  Home: "Home",
  End: "End",
  PageUp: "Page Up",
  PageDown: "Page Down",
  Equal: "=",
  Minus: "-",
};

const MAC_TEXT: Partial<Record<Key, ReactElement | string>> = {
  ...SHARED_TEXT,
  Control: <Icon.Keyboard.Command key="control" />,
  Alt: <Icon.Keyboard.Option key="alt" />,
  Shift: <Icon.Keyboard.Shift key="shift" />,
  // Apple labels the backspace key "delete".
  Backspace: "Delete",
};

const PC_TEXT: Partial<Record<Key, ReactElement | string>> = {
  ...SHARED_TEXT,
  Control: "Ctrl",
  Alt: "Alt",
  Shift: "Shift",
  Backspace: "Backspace",
};

const isMouseTrigger = (trigger: Key): boolean =>
  trigger === "MouseLeft" || trigger === "MouseRight" || trigger === "MouseMiddle";

// macOS modifiers lead in Apple's order, ⌃ ⌥ ⇧ ⌘, where Control prints as ⌘. Windows
// and Linux keep the declared order. Mouse buttons always trail.
const MAC_MODIFIERS: Key[] = ["Alt", "Shift", "Control"];

const rank = (key: Key, mac: boolean): number => {
  if (isMouseTrigger(key)) return MAC_MODIFIERS.length + 1;
  const i = mac ? MAC_MODIFIERS.indexOf(key) : -1;
  return i === -1 ? MAC_MODIFIERS.length : i;
};

/** @returns the icons and letters a trigger reads as on the platform. */
export const toSymbols = (
  trigger: Trigger,
  os: runtime.OS | undefined = runtime.getOS(),
): (ReactElement | string)[] => {
  const mac = isMac(os);
  const text = mac ? MAC_TEXT : PC_TEXT;
  return [...trigger]
    .sort((a, b) => rank(a, mac) - rank(b, mac))
    .map((key) => text[key] ?? key);
};

/**
 * Renders a label followed by its shortcut as one chip of keys, with the platform's
 * own modifier symbols. A trigger the {@link Provider} reserves shows only its label,
 * or nothing.
 *
 * @example <Triggers.Text trigger={["Control", "S"]}>Save</Triggers.Text>
 */
export const Text = <E extends Generic.ElementType = "p">({
  trigger,
  children,
  level,
  os = runtime.getOS(),
  ...rest
}: TextProps<E>): ReactElement | null => {
  const reserved = useReserved(trigger);
  if (reserved && children == null) return null;
  return (
    <Base.Text level={level} gap="small" {...rest}>
      {children}
      {!reserved && (
        <Base.Text
          level={level}
          className={CSS.BE("triggers", "keys")}
          el="span"
          gap="tiny"
        >
          {toSymbols(trigger, os).map((symbol, i) => (
            <Base.Text key={i} el="span" variant="keyboard" level={level}>
              {symbol}
            </Base.Text>
          ))}
        </Base.Text>
      )}
    </Base.Text>
  );
};
