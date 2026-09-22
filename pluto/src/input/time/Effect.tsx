// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/time/Time.css";

import { type ReactElement, type ReactNode, useLayoutEffect, useRef } from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";

export interface EffectProps {
  /** What a commit would change, or null when it would change nothing else. */
  children: ReactNode;
  className?: string;
}

/**
 * A footer that says what a commit would change. It expands in when `children` turns
 * non-null and collapses out, still showing its last content, when they turn null.
 */
export const Effect = ({ children, className }: EffectProps): ReactElement => {
  const visible = children != null;
  const last = useRef<ReactNode>(children);
  useLayoutEffect(() => {
    if (visible) last.current = children;
  });
  return (
    <div
      className={CSS.cls(CSS.B("time-effect"), CSS.visible(visible), className)}
      aria-hidden={!visible}
    >
      <div className={CSS.BE("time-effect", "clip")}>
        <Flex.Box
          x
          wrap
          align="center"
          gap="small"
          className={CSS.BE("time-effect", "content")}
        >
          {visible ? children : last.current}
        </Flex.Box>
      </div>
    </div>
  );
};
