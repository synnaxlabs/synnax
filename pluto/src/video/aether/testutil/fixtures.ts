// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type VideoFrame as EncodedFrame } from "@synnaxlabs/x";

// The frames come from ffmpeg's libx264 (-preset ultrafast -tune zerolatency -bf 0)
// encoding a solid color, with the SEI units removed. Each keyframe carries its SPS and
// PPS.

const frame = (keyframe: boolean, base64: string): EncodedFrame => ({
  keyframe,
  codec: "h264",
  data: Uint8Array.from(atob(base64), (c) => c.charCodeAt(0)),
});

/** Eight 64x64 frames with keyframes at 0 and 4. */
export const STREAM: EncodedFrame[] = [
  frame(
    true,
    "AAAAAWdCwAraEJsBEAAAAwAQAAADA8jxImoAAAABaM4PyAAAAAFliIQ6EYoAAg0xwABBejgACBTJycnXXXXXXXXXXXXg",
  ),
  frame(false, "AAAAAUGaIDaCMA=="),
  frame(false, "AAAAAUGaQDqCMA=="),
  frame(false, "AAAAAUGaYDqCMA=="),
  frame(
    true,
    "AAAAAWdCwAraEJsBEAAAAwAQAAADA8jxImoAAAABaM4PyAAAAAFliIIDqEYoAAjvxwABG2jgACHbJycnXXXXXXXXXXXXgA==",
  ),
  frame(false, "AAAAAUGaIDqCMA=="),
  frame(false, "AAAAAUGaQDqCMA=="),
  frame(false, "AAAAAUGaYDqCMA=="),
];

/** Two 32x32 frames with a keyframe at 0. */
export const SMALL: EncodedFrame[] = [
  frame(
    true,
    "AAAAAWdCwAraJbARAAADAAEAAAMAPI8SJqAAAAABaM4PyAAAAAFliIQ6EYoAAg0xwABBejgACBTJ114=",
  ),
  frame(false, "AAAAAUGaICaL"),
];
