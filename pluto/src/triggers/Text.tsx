// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Icon } from "@synnaxlabs/media";

import { Align } from "@/align";
import { Text as Core } from "@/text";
import { type Key, type Trigger } from "@/triggers/triggers";

export type TextProps<L extends Core.Level> = Core.KeyboardProps<L> & {
  trigger: Trigger;
};

export const Text = <L extends Core.Level>({
  className,
  style,
  trigger,
  ...props
}: TextProps<L>): ReactElement => {
  const CUSTOM_TEXT: Partial<Record<Key, ReactElement | string>> = {
    Control: <Core.Symbols.Meta />,
    Alt: <Core.Symbols.Alt />,
    Shift: <Icon.Keyboard.Shift />,
    MouseLeft: "Left Click",
    MouseRight: "Right Click",
    MouseMiddle: "Middle Click",
  };
  return (
    <Align.Space className={className} style={style} direction="x">
      {trigger.map((t) => (
        // @ts-expect-error - issues with generic element types
        <Core.Keyboard<L> key={t} {...props}>
          {CUSTOM_TEXT[t] ?? t}
        </Core.Keyboard>
      ))}
    </Align.Space>
  );
};
