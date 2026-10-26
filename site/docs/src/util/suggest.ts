// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { normalizeRoute } from "@/util/route";

// The lowest score of a likely match: a typing error or a moved page scores above it,
// an unrelated route below it.
const THRESHOLD = 0.62;
// The last segment carries most of what a reader asked for.
const LEAF_WEIGHT = 0.6;

const distance = (a: string, b: string): number => {
  let prev = Array.from({ length: b.length + 1 }, (_, j) => j);
  for (let i = 1; i <= a.length; i++) {
    const row = [i];
    for (let j = 1; j <= b.length; j++)
      row.push(
        Math.min(
          prev[j] + 1,
          row[j - 1] + 1,
          prev[j - 1] + (a[i - 1] === b[j - 1] ? 0 : 1),
        ),
      );
    prev = row;
  }
  return prev[b.length];
};

const similarity = (a: string, b: string): number =>
  1 - distance(a, b) / Math.max(a.length, b.length, 1);

const leaf = (route: string): string => route.slice(route.lastIndexOf("/") + 1);

/**
 * Returns at most limit routes that are likely what a reader meant by a route that has
 * no page, best first. The list is empty when no route is close.
 */
export const suggest = (route: string, routes: string[], limit = 3): string[] => {
  const target = normalizeRoute(route.toLowerCase());
  return routes
    .map((candidate) => ({
      route: candidate,
      score:
        LEAF_WEIGHT * similarity(leaf(target), leaf(candidate)) +
        (1 - LEAF_WEIGHT) * similarity(target, candidate),
    }))
    .filter(({ score }) => score >= THRESHOLD)
    .sort((a, b) => b.score - a.score || a.route.length - b.route.length)
    .slice(0, limit)
    .map(({ route }) => route);
};
