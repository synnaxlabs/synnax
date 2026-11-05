// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Base, type BaseProps } from "@/video/Base";
import { useChannel } from "@/video/queries";

export interface VideoProps extends Omit<BaseProps, "channel"> {}

/**
 * Shows the channel of the video view in the surrounding Video scope. It suspends until
 * the view is in the cache, so render it within a Video.Suspended boundary.
 */
export const Video = (props: VideoProps): ReactElement => {
  const channel = useChannel();
  return <Base channel={channel} {...props} />;
};
