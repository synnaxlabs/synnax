// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/log/log.css";

import { Icon } from "@synnaxlabs/media";
import { box, Optional } from "@synnaxlabs/x";
import { ReactElement, useCallback, useEffect, useRef } from "react";
import { z } from "zod";

import { Aether } from "@/aether";
import { Align } from "@/align";
import { Button } from "@/button";
import { CSS } from "@/css";
import { useCombinedRefs } from "@/hooks";
import { useMemoDeepEqualProps } from "@/memo";
import { Canvas } from "@/vis/canvas";
import { log } from "@/vis/log/aether";

export interface LogProps
  extends Optional<
      Omit<z.input<typeof log.logState>, "region" | "scrollPosition" | "scrollback">,
      "visible"
    >,
    Omit<Align.SpaceProps, "color"> {}

export const Log = Aether.wrap<LogProps>(
  "Log",
  ({
    aetherKey,
    font,
    className,
    visible = true,
    color,
    telem,
    ...props
  }): ReactElement | null => {
    const memoProps = useMemoDeepEqualProps({ font, color, telem, visible });
    const [, { scrollback }, setState] = Aether.use({
      aetherKey,
      type: log.Log.TYPE,
      schema: log.logState,
      initialState: {
        region: box.ZERO,
        scrollback: false,
        scrollPosition: 0,
        ...memoProps,
      },
    });

    useEffect(() => {
      setState((s) => ({ ...s, ...memoProps }));
    }, [memoProps, setState]);

    const resizeRef = Canvas.useRegion(
      useCallback(
        (b) => {
          setState((s) => ({ ...s, region: b }));
        },
        [setState],
      ),
    );

    const logRef = useRef<HTMLDivElement | null>(null);
    const combinedRef = useCombinedRefs(logRef, resizeRef);

    return (
      <div
        ref={combinedRef}
        className={CSS(CSS.B("log"), className)}
        onWheel={(e) => {
          setState((s) => ({
            ...s,
            scrollPosition: s.scrollPosition - e.deltaY,
          }));
        }}
        {...props}
      >
        <Button.Icon
          className={CSS(CSS.BE("log", "live"), CSS.visible(scrollback))}
          variant="outlined"
          onClick={() => {
            setState((s) => ({ ...s, scrollback: false }));
          }}
          tooltip="Return to Live"
        >
          <Icon.Dynamic style={{ color: "var(--pluto-error-p1)" }} />
        </Button.Icon>
      </div>
    );
  },
);
