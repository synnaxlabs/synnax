// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useEffect, useRef, type PropsWithChildren } from "react";

import { TimeStamp } from "@synnaxlabs/x";
import { type z } from "zod";

import { Aether } from "@/aether";
import { type Color } from "@/color";
import { control } from "@/control/aether";
import { CSS } from "@/css";
import { useMemoDeepEqualProps } from "@/hooks";
import { Text } from "@/text";
import { Tooltip } from "@/tooltip";

import "@/control/Indicator.css";

import { Align } from "..";

export interface IndicatorProps
  extends Omit<z.input<typeof control.indicatorStateZ>, "status" | "color">,
    PropsWithChildren {}

export const Indicator = Aether.wrap<IndicatorProps>(
  control.Indicator.TYPE,
  ({ aetherKey, colorSource, statusSource, children }): ReactElement => {
    const memoProps = useMemoDeepEqualProps({ colorSource, statusSource });

    const [, { status, color }, setState] = Aether.use({
      aetherKey,
      type: control.Indicator.TYPE,
      initialState: {
        ...memoProps,
        status: {
          key: "no_chip",
          variant: "warning",
          message: "No chip connected.",
          time: TimeStamp.now(),
        },
        color: "#000000",
      },
      schema: control.indicatorStateZ,
    });

    useEffect(() => {
      setState((p) => ({ ...p, ...memoProps }));
    }, [memoProps, setState]);

    const prevColor = useRef<Color.Crude>(color);
    const cycleCount = useRef(0);
    useEffect(() => {
      if (color.equals(prevColor.current)) return;
      prevColor.current = color;
      cycleCount.current += 1;
    }, [color]);

    return (
      // <div
      //   className={CSS.B("indicator")}
      //   style={{
      //     // borderColor: Color.cssString(color),
      //     // // backgroundImage: `linear-gradient(to ${bgImageStart}, ${Color.cssString(
      //     // //   prevColor.current,
      //     // // )} 50%, ${Color.cssString(color)} 50%)`,
      //     // // : `-${(cycleCount.current % 2) * 100}% 0`,
      //     // // flexGrow: 1,
      //   }}
      // >
      <>
        {children}
        <Text.Text level="small">{status.key}</Text.Text>
      </>
      // </div>
    );
  },
);
