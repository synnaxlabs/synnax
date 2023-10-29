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
import { Color } from "@/color";
import { control } from "@/control/aether";
import { CSS } from "@/css";
import { useMemoDeepEqualProps } from "@/hooks";

import "@/control/Indicator.css";

export interface IndicatorProps
  extends Omit<z.input<typeof control.indicatorStateZ>, "status" | "color">,
    PropsWithChildren {}

export const Indicator = Aether.wrap<IndicatorProps>(
  control.Indicator.TYPE,
  ({ aetherKey, colorSource, statusSource }): ReactElement => {
    const memoProps = useMemoDeepEqualProps({ colorSource, statusSource });

    const [, { color }, setState] = Aether.use({
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

    return (
      <div
        className={CSS.B("indicator")}
        style={{
          backgroundColor: Color.cssString(color),
          flexGrow: 1,
        }}
      />
    );
  },
);
