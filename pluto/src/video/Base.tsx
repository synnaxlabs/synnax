// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/video/Video.css";

import { type channel } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/lyra/status";
import { box, type TimeRange } from "@synnaxlabs/x";
import {
  type ComponentPropsWithoutRef,
  type ReactElement,
  useEffect,
  useState,
} from "react";

import { Aether } from "@/aether";
import { CSS } from "@/css";
import { playVideo, streamVideo, video } from "@/video/aether";
import { noopVideoSourceSpec } from "@/video/aether/telem/noop";
import { clock } from "@/video/clock";
import { Controls } from "@/video/Controls";
import { Canvas } from "@/vis/canvas";

export interface BaseProps
  extends Aether.ComponentProps, Omit<ComponentPropsWithoutRef<"div">, "children"> {
  /** The video channel to show. Zero shows no video. */
  channel: channel.Key;
  /** The range to play, with controls. Absent shows live video. */
  range?: TimeRange;
  /** @default true */
  visible?: boolean;
}

const message = (text: string): ReactElement => (
  <Status.Summary center level="h3" variant="disabled" hideIcon>
    {text}
  </Status.Summary>
);

/**
 * Shows the newest picture of a live video channel, or plays a range of a stored one,
 * scaled to fit.
 */
export const Base = ({
  aetherKey,
  channel,
  range,
  visible = true,
  className,
  ...rest
}: BaseProps): ReactElement => {
  const rangeKey = range?.toString();
  const [playback, setPlayback] = useState<clock.Clock | null>(null);
  useEffect(() => {
    setPlayback(range == null ? null : clock.stopped(range.start));
  }, [rangeKey]);
  const c = range == null ? undefined : (playback ?? clock.stopped(range.start));
  // WebCodecs exists only in a secure context, such as HTTPS or localhost.
  const supported = typeof VideoDecoder !== "undefined";
  const playable = supported && channel !== 0;
  let telem = noopVideoSourceSpec;
  if (playable)
    telem = range == null ? streamVideo({ channel }) : playVideo({ channel });
  const [, { empty }, setState] = Aether.use({
    aetherKey,
    type: video.Video.TYPE,
    schema: video.videoStateZ,
    initialState: { region: box.ZERO, telem, visible, empty: true, clock: c },
  });
  useEffect(() => {
    setState((s) => ({ ...s, telem, visible, clock: c }));
  }, [channel, visible, rangeKey, playback, setState]);
  const resizeRef = Canvas.useRegion((b) => setState((s) => ({ ...s, region: b })));
  let content: ReactElement | null = null;
  if (!supported) content = message("Video needs HTTPS or the desktop app");
  else if (channel === 0) content = message("No video channel");
  else if (empty) content = message(range == null ? "Waiting for video" : "No video");
  return (
    <div className={CSS.cls(CSS.B("video"), className)} {...rest}>
      <div ref={resizeRef} className={CSS.BE("video", "picture")}>
        {content}
      </div>
      {playable && range != null && c != null && (
        <Controls range={range} clock={c} onChange={setPlayback} />
      )}
    </div>
  );
};
