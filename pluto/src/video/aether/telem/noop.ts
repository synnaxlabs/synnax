// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor, type observe } from "@synnaxlabs/x";

import { telem } from "@/telem/aether";
import { type VideoSource, type VideoSourceSpec } from "@/video/aether/telem/types";

export class NoopVideoSource implements VideoSource {
  static readonly TYPE = "noop-video-source";

  value(): telem.Result<VideoFrame | null> {
    return telem.success(null);
  }

  cleanup(): void {}

  onChange(_handler: observe.Handler<void>): destructor.Destructor {
    return () => {};
  }
}

export const noopVideoSourceSpec: VideoSourceSpec = {
  type: NoopVideoSource.TYPE,
  props: {},
  variant: "source",
  valueType: "video",
};
