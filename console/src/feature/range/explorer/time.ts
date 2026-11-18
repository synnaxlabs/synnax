// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { Ranger } from "@synnaxlabs/pluto";
import { TimeRange, TimeSpan, type TimeStamp } from "@synnaxlabs/x";

const hasFields = (where: ranger.Where): boolean =>
  Object.values(where).some((value) => value !== undefined);

/**
 * @returns the filter with its stages and recent spans read at the instant, so the
 * Core answers it as of that instant. A filter without them comes back as it is.
 */
export const pin = (where: ranger.Where, at: TimeStamp): ranger.Where => {
  const { stages, recent, and, or, not, ...rest } = where;
  const base: ranger.Where = { ...rest };
  if (and != null) base.and = and.map((node) => pin(node, at));
  if (or != null) base.or = or.map((node) => pin(node, at));
  if (not != null) base.not = pin(not, at);
  const pinned: ranger.Where[] = [];
  if (stages != null)
    pinned.push({ or: stages.map((stage) => Ranger.stageWhere(stage, at)) });
  if (recent != null)
    pinned.push({ overlaps: new TimeRange(at.sub(new TimeSpan(recent)), at) });
  if (pinned.length === 0) return base;
  return { and: hasFields(base) ? [base, ...pinned] : pinned };
};
