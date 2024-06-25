// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { type ReactElement } from "react";

import { Align } from "@/align";
import { Text as Core } from "@/text";
import { type Key, type Trigger } from "@/triggers/triggers";

export type TextProps<L extends Core.Level> = Core.KeyboardProps<L> & {
  trigger: Trigger;
};

const CUSTOM_TEXT: Partial<Record<Key, (() => ReactElement) | string>> = {
  Control: () => <Core.Symbols.Meta key="control" />,
  Alt: () => <Core.Symbols.Alt key="alt" />,
  Shift: () => <Icon.Keyboard.Shift key="shift" />,
  MouseLeft: "Left Click",
  MouseRight: "Right Click",
  MouseMiddle: "Middle Click",
  Enter: () => <Icon.Keyboard.Return key="enter" />,
};

const getCustomText = (trigger: Key): ReactElement | string => {
  const t = CUSTOM_TEXT[trigger];
  if (t != null) return typeof t === "function" ? t() : t;
  return trigger;
};

export const toSymbols = (trigger: Trigger): (ReactElement | string)[] =>
  trigger.map((t) => getCustomText(t));

export const Text = <L extends Core.Level>({
  className,
  style,
  trigger,
  children,
  ...props
}: TextProps<L>): ReactElement => {
  return (
    <>
      {trigger.map((t) => (
        // @ts-expect-error - issues with generic element types
        <Core.Keyboard<L> key={t} {...props}>
          {getCustomText(t)}
        </Core.Keyboard>
      ))}
    </>
  );
};
