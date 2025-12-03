// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/lineplot/Viewport.css";

import { box, xy } from "@synnaxlabs/x";
import {
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useLayoutEffect,
} from "react";

import { CSS } from "@/css";
import { useContext } from "@/lineplot/LinePlot";
import { Viewport as Core } from "@/viewport";

export interface ViewportProps extends PropsWithChildren, Core.UseProps {}

export const selectViewportEl = (el: HTMLElement | null): Element | null =>
  el == null ? null : (el.closest(".pluto-line-plot__viewport") ?? null);

export const Viewport = ({
  ref,
  children,
  initial = box.DECIMAL,
  onChange,
  ...rest
}: ViewportProps): ReactElement => {
  const { setViewport } = useContext("LinePlot.Viewport");

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

  const maskProps = Core.use({ onChange: handleChange, initial, ref, ...rest });

  return (
    <Core.Mask className={CSS.BE("line-plot", "viewport")} {...maskProps}>
      {children}
    </Core.Mask>
  );
};
