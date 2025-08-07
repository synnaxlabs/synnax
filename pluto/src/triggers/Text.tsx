// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { type Generic } from "@/generic";
import { Icon } from "@/icon";
import { Text as Core } from "@/text";
import { type Key, type Trigger } from "@/triggers/triggers";

export type TextProps<E extends Generic.ElementType = "p"> = Core.TextProps<E> & {
  trigger: Trigger;
};

const CUSTOM_TEXT: Partial<Record<Key, (() => ReactElement) | string>> = {
  Control: () => <Core.Symbols.Meta key="control" />,
  Alt: () => <Core.Symbols.Alt key="alt" />,
  Shift: () => <Icon.Keyboard.Shift key="shift" />,
  MouseLeft: "Click",
  MouseRight: "Click",
  MouseMiddle: "Click",
  Enter: () => <Icon.Keyboard.Return key="enter" />,
};

const getCustomText = (trigger: Key): ReactElement | string => {
  const t = CUSTOM_TEXT[trigger];
  if (t != null) return typeof t === "function" ? t() : t;
  return trigger;
};

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

export const toSymbols = (trigger: Trigger): (ReactElement | string)[] =>
  trigger.map((t) => getCustomText(t));

export const Text = <E extends Generic.ElementType = "p">({
  trigger,
  children,
  level,
  ...rest
}: TextProps<E>): ReactElement => (
  <Core.Text level={level} gap="small" {...rest}>
    {sortTriggers(trigger).map((t) => (
      <Core.Text key={t} el="span" variant="keyboard" level={level}>
        {getCustomText(t)}
      </Core.Text>
    ))}
    {children}
  </Core.Text>
);
