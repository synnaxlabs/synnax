// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/lineplot/Viewport.css";

import { box, xy } from "@synnaxlabs/x";
import {
  forwardRef,
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useLayoutEffect,
} from "react";

import { CSS } from "@/css";
import { Viewport as Core } from "@/viewport";
import { type UseRefValue } from "@/viewport/use";
import { useContext } from "@/vis/lineplot/LinePlot";

export interface ViewportProps extends PropsWithChildren, Core.UseProps {}

export const selectViewportEl = (el: HTMLElement | null): Element | null =>
  el == null
    ? null
    : document.querySelectorAll(".pluto-line-plot__viewport")[0] ?? null;

export const Viewport = forwardRef<UseRefValue | undefined, ViewportProps>(
  (
    { children, initial = box.DECIMAL, onChange, ...props }: ViewportProps,
    ref,
  ): ReactElement => {
    const { setViewport } = useContext("Viewport");

    useLayoutEffect(() => {
      setViewport({ box: initial, mode: "zoom", cursor: xy.ZERO, stage: "start" });
    }, [setViewport, initial]);

    const handleChange = useCallback(
      (e: Core.UseEvent): void => {
        setViewport(e);
        onChange?.(e);
      },
      [onChange, setViewport],
    );

    const maskProps = Core.use({
      onChange: handleChange,
      initial,
      ref,
      ...props,
    });

    return (
      <Core.Mask className={CSS.BE("line-plot", "viewport")} {...maskProps}>
        {children}
      </Core.Mask>
    );
  },
);
Viewport.displayName = "Viewport";
