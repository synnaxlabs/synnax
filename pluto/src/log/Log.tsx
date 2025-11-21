// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/log/Log.css";

import { box, location, type optional } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useEffect } from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { Button } from "@/button";
import { CSS } from "@/css";
import { type Flex } from "@/flex";
import { Icon } from "@/icon";
import { log } from "@/log/aether";
import { useMemoDeepEqual } from "@/memo";
import { Status } from "@/status/core";
import { Canvas } from "@/vis/canvas";

export interface LogProps
  extends optional.Optional<
      Omit<
        z.input<typeof log.logState>,
        "region" | "scrollPosition" | "scrollback" | "empty" | "scrolling" | "wheelPos"
      >,
      "visible"
    >,
    Omit<Flex.BoxProps, "color">,
    Aether.ComponentProps {
  emptyContent?: ReactElement;
}

export const Log = ({
  aetherKey,
  font,
  className,
  visible = true,
  emptyContent = (
    <Status.Summary center level="h3" variant="disabled" hideIcon>
      Empty Log
    </Status.Summary>
  ),
  color,
  telem,
  ...rest
}: LogProps): ReactElement | null => {
  const memoProps = useMemoDeepEqual({ font, color, telem, visible });
  const [, { scrolling, empty }, setState] = Aether.use({
    type: log.Log.TYPE,
    schema: log.logState,
    initialState: {
      empty: true,
      region: box.ZERO,
      scrolling: false,
      wheelPos: 0,
      ...memoProps,
    },
  });

  useEffect(() => {
    setState((s) => ({ ...s, ...memoProps }));
  }, [memoProps, setState]);

  const resizeRef = Canvas.useRegion(
    useCallback((b) => setState((s) => ({ ...s, region: b })), [setState]),
  );

  return (
    <div
      ref={resizeRef}
      className={CSS(CSS.B("log"), className)}
      onWheel={(e) => {
        setState((s) => ({
          ...s,
          wheelPos: s.wheelPos - e.deltaY,
          scrolling: s.scrolling ? s.scrolling : e.deltaY < 0,
        }));
      }}
      {...rest}
    >
      {empty ? (
        emptyContent
      ) : (
        <Button.Button
          className={CSS(CSS.BE("log", "live"), scrolling && CSS.M("active"))}
          variant="outlined"
          onClick={() => setState((s) => ({ ...s, scrolling: !s.scrolling }))}
          tooltip={scrolling ? "Resume Scrolling" : "Pause Scrolling"}
          tooltipLocation={location.BOTTOM_LEFT}
        >
          <Icon.Dynamic />
        </Button.Button>
      )}
    </div>
  );
};
