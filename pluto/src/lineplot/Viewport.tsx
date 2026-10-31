// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/lineplot/Viewport.css";

import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { box, xy } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement, useLayoutEffect } from "react";

import { CSS } from "@/css";
import { useContext, useFetchContext } from "@/lineplot/Frame";
import { Viewport as Base } from "@/viewport";

export interface ViewportProps
  extends
    PropsWithChildren,
    Base.UseProps,
    Pick<
      Base.MaskProps,
      "className" | "onDragOver" | "onDrop" | "onDragEnter" | "onDragLeave"
    > {}

export const selectViewportEl = (el: HTMLElement | null): Element | null =>
  el == null ? null : (el.closest(".pluto-line-plot__viewport") ?? null);

export const Viewport = ({
  ref,
  children,
  initial = box.DECIMAL,
  onChange,
  className,
  onDragOver,
  onDrop,
  onDragEnter,
  onDragLeave,
  ...rest
}: ViewportProps): ReactElement => {
  const { setViewport, setViewportElement } = useContext("LinePlot.Viewport");
  const { loading, fetching, loadingMessage } = useFetchContext("LinePlot.Viewport");

  useLayoutEffect(() => {
    setViewport({ box: initial, mode: "zoom", cursor: xy.ZERO, stage: "start" });
  }, [setViewport, initial]);

  const handleChange = (e: Base.UseEvent): void => {
    setViewport(e);
    onChange?.(e);
  };

  const maskProps = Base.use({ onChange: handleChange, initial, ref, ...rest });

  const { ref: maskRef } = maskProps;
  useLayoutEffect(() => {
    setViewportElement(maskRef.current);
    return () => setViewportElement(null);
  }, [maskRef, setViewportElement]);

  return (
    <Base.Mask
      className={CSS.cls(CSS.BE("line-plot", "viewport"), className)}
      onDragOver={onDragOver}
      onDrop={onDrop}
      onDragEnter={onDragEnter}
      onDragLeave={onDragLeave}
      {...maskProps}
    >
      {loading && (
        <Status.Loading className={CSS.BE("line-plot", "loading")}>
          <Status.Orbital />
          {loadingMessage != null && (
            <Text.Text className={CSS.BE("line-plot", "loading-message")}>
              {loadingMessage}
            </Text.Text>
          )}
        </Status.Loading>
      )}
      {!loading && fetching && (
        <Icon.Loading className={CSS.BE("line-plot", "fetching")} />
      )}
      {children}
    </Base.Mask>
  );
};
