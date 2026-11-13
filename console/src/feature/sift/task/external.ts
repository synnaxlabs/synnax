// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Stream } from "@/feature/sift/task/Stream";
import { STREAM_TYPE } from "@/feature/sift/task/types";
import { type Task } from "@/platform/task";

export * from "@/feature/sift/task/commands";
export * from "@/feature/sift/task/integration";
export * from "@/feature/sift/task/Stream";
export * from "@/feature/sift/task/types";

export const FORMS: Task.Forms = { [STREAM_TYPE]: Stream };
