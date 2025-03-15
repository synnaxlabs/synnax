// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/log/Log.css";

import { Icon } from "@synnaxlabs/media";
import { box, type Optional } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useEffect } from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { type Align } from "@/align";
import { Button } from "@/button";
import { CSS } from "@/css";
import { useMemoDeepEqualProps } from "@/memo";
import { Status } from "@/status";
import { Canvas } from "@/vis/canvas";
import { log } from "@/vis/log/aether";

export interface LogProps
  extends Optional<
      Omit<
        z.input<typeof log.logState>,
        "region" | "scrollPosition" | "scrollback" | "empty" | "scrolling" | "wheelPos"
      >,
      "visible"
    >,
    Omit<Align.SpaceProps, "color">,
    Aether.CProps {
  emptyContent?: ReactElement;
}

export const Log = ({
  aetherKey,
  font,
  className,
  visible = true,
  emptyContent = (
    <Status.Text.Centered level="h3" variant="disabled" hideIcon>
      Empty Log
    </Status.Text.Centered>
  ),
  color,
  telem,
  ...rest
}: LogProps): ReactElement | null => {
  const memoProps = useMemoDeepEqualProps({ font, color, telem, visible });
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
        <Button.Icon
          className={CSS(CSS.BE("log", "live"), scrolling && CSS.M("active"))}
          variant="outlined"
          onClick={() => setState((s) => ({ ...s, scrolling: !s.scrolling }))}
          tooltip={scrolling ? "Resume Scrolling" : "Pause Scrolling"}
        >
          <Icon.Dynamic />
        </Button.Icon>
      )}
    </div>
  );
};
