// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Store } from "@/server/db/db";
import { event, type EventKind } from "@/server/db/schema";

export interface RecordArgs {
  kind: Extract<EventKind, "thread" | "feedback">;
  actor: string;
  organization?: string;
  detail: Record<string, unknown>;
}

/** record logs a support action so the rate limiter can count it. */
export const record = async (
  store: Store,
  { kind, actor, organization, detail }: RecordArgs,
): Promise<void> => {
  await store.query.insert(event).values({ kind, actor, organization, detail });
};
