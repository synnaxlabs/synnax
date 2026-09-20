// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { runtime } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { type Generic } from "@/generic";
import { Icon } from "@/icon";
import { Text as Base } from "@/text";
import { type Key, type Trigger } from "@/triggers/triggers";

/** Props for {@link Text}. */
export type TextProps<E extends Generic.ElementType = "p"> = Base.TextProps<E> & {
  trigger: Trigger;
};

const isWindows = runtime.getOS() == "Windows";

const Control = isWindows ? Icon.Keyboard.Control : Icon.Keyboard.Command;
const Alt = isWindows ? () => "Alt" : Icon.Keyboard.Option;

// A keycap prints what the physical key prints, and the two platforms disagree: Apple
// sets its word keys in lowercase, Windows capitalizes them.
const word = (mac: string, windows: string): string => (isWindows ? windows : mac);

const CUSTOM_TEXT: Partial<Record<Key, ReactElement | string>> = {
  Control: <Control key="control" />,
  Alt: <Alt key="alt" />,
  Shift: <Icon.Keyboard.Shift key="shift" />,
  MouseLeft: <Icon.Click key="mouse" />,
  MouseRight: <Icon.Click key="mouse" />,
  MouseMiddle: <Icon.Click key="mouse" />,
  Enter: <Icon.Keyboard.Return key="enter" />,
  Escape: word("esc", "Esc"),
  Delete: word("delete", "Delete"),
  Backspace: word("delete", "Backspace"),
  Tab: word("tab", "Tab"),
  Space: word("space", "Space"),
  CapsLock: word("caps lock", "Caps Lock"),
  Home: word("home", "Home"),
  End: word("end", "End"),
  PageUp: word("page up", "Page Up"),
  PageDown: word("page down", "Page Down"),
  Equal: "=",
  Minus: "-",
};

const getCustomText = (trigger: Key): ReactElement | string =>
  CUSTOM_TEXT[trigger] ?? trigger;

const isMouseTrigger = (trigger: Key): boolean =>
  trigger === "MouseLeft" || trigger === "MouseRight" || trigger === "MouseMiddle";

const sortTriggers = (trigger: Trigger): Trigger =>
  [...trigger].sort((a, b) => {
    const aIsMouse = isMouseTrigger(a);
    const bIsMouse = isMouseTrigger(b);
    if (aIsMouse && !bIsMouse) return 1;
    if (!aIsMouse && bIsMouse) return -1;
    return 0;
  });

/** @returns the icons and letters a trigger reads as, per the current platform. */
export const toSymbols = (trigger: Trigger): (ReactElement | string)[] =>
  trigger.map((t) => getCustomText(t));

/**
 * Renders a shortcut as keycaps followed by its label, with the platform's own modifier
 * symbols.
 *
 * @example <Triggers.Text trigger={["Control", "S"]}>Save</Triggers.Text>
 */
export const Text = <E extends Generic.ElementType = "p">({
  trigger,
  children,
  level,
  ...rest
}: TextProps<E>): ReactElement => (
  <Base.Text level={level} gap="small" {...rest}>
    <Base.Text level={level} gap="tiny" el="span">
      {sortTriggers(trigger).map((t) => (
        <Base.Text key={t} el="span" variant="keyboard" level={level}>
          {getCustomText(t)}
        </Base.Text>
      ))}
    </Base.Text>
    {children}
  </Base.Text>
);
