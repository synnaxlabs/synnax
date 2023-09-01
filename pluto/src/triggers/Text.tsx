// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { Align } from "@/align";
import { Text as Core } from "@/text";
import { Trigger } from "@/triggers/triggers";

export type TextProps<L extends Core.Level> = Core.KeyboardProps<L> & {
  trigger: Trigger;
};

export const Text = <L extends Core.Level>({
  className,
  style,
  trigger,
  ...props
}: TextProps<L>): ReactElement => (
  <Align.Space className={className} style={style}>
    {trigger.map((t) => (
      // @ts-expect-error
      <Core.Keyboard<L> key={t} {...props}>
        {t}
      </Core.Keyboard>
    ))}
  </Align.Space>
);
