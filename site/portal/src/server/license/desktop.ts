// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { and, desc, eq, isNull } from "drizzle-orm";

import { type Reader } from "@/server/db/db";
import { type Activation, activation, type License, license } from "@/server/db/schema";

export interface Machine {
  activation: Activation;
  license: License;
}

/**
 * machinesFor returns the Desktop machines of an organization that hold a seat, the
 * most recently seen first.
 */
export const machinesFor = async (
  db: Reader,
  organization: string,
): Promise<Machine[]> =>
  await db
    .select({ activation, license })
    .from(activation)
    .innerJoin(license, eq(activation.license, license.key))
    .where(
      and(
        eq(license.organization, organization),
        eq(license.edition, "desktop"),
        isNull(activation.releasedAt),
      ),
    )
    .orderBy(desc(activation.lastSeen));
