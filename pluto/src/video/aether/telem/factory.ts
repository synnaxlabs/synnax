// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type telem } from "@/telem/aether";
import { NoopVideoSource } from "@/video/aether/telem/noop";
import { PlayVideo, StreamVideo } from "@/video/aether/telem/sources";

export class VideoFactory implements telem.Factory {
  type = "video";

  constructor(private readonly client: telem.Client | null) {}

  create(spec: telem.Spec, options?: telem.CreateOptions): telem.Telem | null {
    if (spec.type === NoopVideoSource.TYPE) return new NoopVideoSource();
    if (spec.type === StreamVideo.TYPE)
      return new StreamVideo(this.client, spec.props, options);
    if (spec.type === PlayVideo.TYPE)
      return new PlayVideo(this.client, spec.props, options);
    return null;
  }
}
