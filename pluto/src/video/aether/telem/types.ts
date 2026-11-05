// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { type Source, sourceSpecZ } from "@/telem/aether/telem";

/** A source of the newest decoded picture of a video channel, or null before one. */
export interface VideoSource extends Source<VideoFrame | null> {}

export const videoSourceSpecZ = sourceSpecZ.extend({ valueType: z.literal("video") });
export type VideoSourceSpec = z.infer<typeof videoSourceSpecZ>;
