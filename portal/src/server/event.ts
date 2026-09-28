// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { desc, eq } from "drizzle-orm";

import { type Store } from "@/server/db/db";
import { type Event, event } from "@/server/db/schema";

/** listForLicense returns a license's audit events, newest first. */
export const listForLicense = async (
  store: Store,
  licenseKey: string,
  limit = 50,
): Promise<Event[]> =>
  await store.query
    .select()
    .from(event)
    .where(eq(event.license, licenseKey))
    .orderBy(desc(event.at))
    .limit(limit);
