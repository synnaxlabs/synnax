// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, xy } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useEffect, useRef } from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { CSS } from "@/css";
import { useMemoDeepEqual } from "@/memo";
import { Canvas } from "@/vis/canvas";
import { Spectrogram } from "@/vis/spectrogram/aether/spectrogram";

export interface SpectrogramProps
  extends Omit<z.input<typeof Spectrogram.z>, "box">,
    Aether.ComponentProps {}

export const SpectrogramC = ({
  aetherKey,
  telem,
  sampleRate,
  fftSize,
  windowFunction,
  overlap,
  colorMap,
  dbMin,
  dbMax,
  freqMin,
  freqMax,
  visible,
}: SpectrogramProps): ReactElement => {
  const memoProps = useMemoDeepEqual({
    telem,
    sampleRate,
    fftSize,
    windowFunction,
    overlap,
    colorMap,
    dbMin,
    dbMax,
    freqMin,
    freqMax,
    visible,
  });
  const [, , setState] = Aether.use({
    aetherKey,
    type: Spectrogram.TYPE,
    schema: Spectrogram.z,
    initialState: { box: box.ZERO, ...memoProps },
  });

  useEffect(() => setState((prev) => ({ ...prev, ...memoProps })), [memoProps]);

  const divRef = useRef<HTMLDivElement>(null);

  const resizeRef = Canvas.useRegion(
    useCallback(
      (b: box.Box) => setState((prev) => ({ ...prev, box: b })),
      [setState],
    ),
  );

  const handleMove = useCallback(
    (e: MouseEvent): void => {
      const canvas = document.querySelector(".pluto-canvas-container");
      if (canvas == null) return;
      const topLeft = box.topLeft(canvas);
      setState((prev) => ({
        ...prev,
        cursorPosition: xy.translation(topLeft, xy.construct(e)),
      }));
    },
    [setState],
  );

  const handleLeave = useCallback(
    (): void => setState((prev) => ({ ...prev, cursorPosition: null })),
    [setState],
  );

  useEffect(() => {
    const el = divRef.current;
    if (el == null) return;
    el.addEventListener("mousemove", handleMove);
    el.addEventListener("mouseleave", handleLeave);
    return () => {
      el.removeEventListener("mousemove", handleMove);
      el.removeEventListener("mouseleave", handleLeave);
    };
  }, [handleMove, handleLeave]);

  const setRefs = useCallback(
    (el: HTMLDivElement | null) => {
      (divRef as React.MutableRefObject<HTMLDivElement | null>).current = el;
      resizeRef(el);
    },
    [resizeRef],
  );

  return (
    <div
      ref={setRefs}
      className={CSS.B("spectrogram")}
      style={{ width: "100%", height: "100%" }}
    />
  );
};
