// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box } from "@synnaxlabs/x";
import { ReactElement, useCallback, useEffect } from "react";
import { z } from "zod";

import { Aether } from "@/aether";
import { Align } from "@/align";
import { CSS } from "@/css";
import { useMemoDeepEqualProps } from "@/memo";
import { Canvas } from "@/vis/canvas";
import { log } from "@/vis/log/aether";

export interface LogProps extends Omit<z.input<typeof log.logState>, "region"> {}

export const Log = Aether.wrap<LogProps>(
  "Log",
  ({
    aetherKey,
    font,
    color,
    precision,
    minWidth,
    width,
    telem,
  }): ReactElement | null => {
    console.log(telem);
    const memoProps = useMemoDeepEqualProps({
      font,
      color,
      precision,
      minWidth,
      width,
      telem,
    });
    const [, , setState] = Aether.use({
      aetherKey,
      type: log.Log.TYPE,
      schema: log.logState,
      initialState: {
        region: box.ZERO,
        ...memoProps,
      },
    });

    useEffect(() => {
      setState((s) => ({ ...s, ...memoProps }));
    }, [setState, memoProps]);

    const resizeRef = Canvas.useRegion(
      useCallback(
        (b) => {
          setState((s) => ({ ...s, region: b }));
        },
        [setState],
      ),
    );

    return (
      <Align.Space
        ref={resizeRef}
        direction="y"
        className={CSS.B("log")}
        style={{ height: "100%" }}
      />
    );
  },
);
