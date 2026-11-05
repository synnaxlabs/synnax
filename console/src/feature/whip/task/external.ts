// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Read, ReadSelectable } from "@/feature/whip/task/Read";
import { READ_TYPE } from "@/feature/whip/task/types";
import { FLAGS } from "@/flags";
import { type Selector } from "@/platform/selector";
import { type Task } from "@/platform/task";

export * from "@/feature/whip/task/commands";
export * from "@/feature/whip/task/Read";
export * from "@/feature/whip/task/types";

export const FORMS: Task.Forms = FLAGS.video ? { [READ_TYPE]: Read } : {};

export const SELECTABLES: Selector.Selectable[] = FLAGS.video ? [ReadSelectable] : [];
