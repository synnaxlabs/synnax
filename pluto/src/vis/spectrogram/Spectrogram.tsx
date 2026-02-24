// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, xy } from "@synnaxlabs/x";
import {
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useEffect,
  useRef,
} from "react";
import { z } from "zod";

import { Aether } from "@/aether";
import { CSS } from "@/css";
import { useUniqueKey } from "@/hooks/useUniqueKey";
import { useMemoDeepEqual } from "@/memo";
import { Canvas } from "@/vis/canvas";
import { ColorBar as AetherColorBar } from "@/vis/spectrogram/aether/colorBar";
import { FreqAxis as AetherFreqAxis } from "@/vis/spectrogram/aether/freqAxis";
import { Spectrogram as AetherSpectrogram } from "@/vis/spectrogram/aether/spectrogram";
import { TimeAxis as AetherTimeAxis } from "@/vis/spectrogram/aether/timeAxis";
import {
  Tooltip as AetherTooltip,
  tooltipStateZ,
} from "@/vis/spectrogram/aether/tooltip";

export interface SpectrogramProps
  extends Omit<z.input<typeof AetherSpectrogram.z>, "box">,
    PropsWithChildren,
    Aether.ComponentProps {}

const emptyZ = z.object({});

export const Spectrogram = ({
  aetherKey,
  children,
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
  const [{ path }, , setState] = Aether.use({
    aetherKey,
    type: AetherSpectrogram.TYPE,
    schema: AetherSpectrogram.z,
    initialState: { box: box.ZERO, ...memoProps },
  });

  useEffect(() => setState((prev) => ({ ...prev, ...memoProps })), [memoProps]);

  const resizeRef = Canvas.useRegion(
    useCallback(
      (b: box.Box) => setState((prev) => ({ ...prev, box: b })),
      [setState],
    ),
  );

  return (
    <div
      ref={resizeRef}
      className={CSS.B("spectrogram")}
      style={{ width: "100%", height: "100%" }}
    >
      <Aether.Composite path={path}>{children}</Aether.Composite>
    </div>
  );
};

export interface FreqAxisProps extends Aether.ComponentProps {}

export const FreqAxis = ({ aetherKey }: FreqAxisProps): ReactElement | null => {
  const cKey = useUniqueKey(aetherKey);
  Aether.use({
    aetherKey: cKey,
    type: AetherFreqAxis.TYPE,
    schema: emptyZ,
    initialState: {},
  });
  return null;
};

export interface TimeAxisProps extends Aether.ComponentProps {}

export const TimeAxis = ({ aetherKey }: TimeAxisProps): ReactElement | null => {
  const cKey = useUniqueKey(aetherKey);
  Aether.use({
    aetherKey: cKey,
    type: AetherTimeAxis.TYPE,
    schema: emptyZ,
    initialState: {},
  });
  return null;
};

export interface ColorBarProps extends Aether.ComponentProps {}

export const ColorBar = ({ aetherKey }: ColorBarProps): ReactElement | null => {
  const cKey = useUniqueKey(aetherKey);
  Aether.use({
    aetherKey: cKey,
    type: AetherColorBar.TYPE,
    schema: emptyZ,
    initialState: {},
  });
  return null;
};

export interface TooltipProps extends Aether.ComponentProps {}

export const Tooltip = ({ aetherKey }: TooltipProps): ReactElement | null => {
  const cKey = useUniqueKey(aetherKey);
  const [, , setState] = Aether.use({
    aetherKey: cKey,
    type: AetherTooltip.TYPE,
    schema: tooltipStateZ,
    initialState: { cursorPosition: null },
  });

  const ref = useRef<HTMLSpanElement>(null);

  const handleMove = useCallback(
    (e: MouseEvent): void => {
      const canvas = document.querySelector(".pluto-canvas-container");
      if (canvas == null) return;
      const topLeft = box.topLeft(canvas);
      setState({ cursorPosition: xy.translation(topLeft, xy.construct(e)) });
    },
    [setState],
  );

  const handleLeave = useCallback(
    (): void => setState({ cursorPosition: null }),
    [setState],
  );

  useEffect(() => {
    if (ref.current === null) return;
    const parent = ref.current.parentElement;
    if (parent == null) return;
    parent.addEventListener("mousemove", handleMove);
    parent.addEventListener("mouseleave", handleLeave);
    return () => {
      parent.removeEventListener("mousemove", handleMove);
      parent.removeEventListener("mouseleave", handleLeave);
    };
  }, [handleMove]);

  return <span ref={ref} />;
};
