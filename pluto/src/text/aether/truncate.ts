// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

const ELLIPSIS = "…";

const GRAPHEMES = new Intl.Segmenter(undefined, { granularity: "grapheme" });

/**
 * Shortens text to fit a width, and puts an ellipsis where it cuts. The cut never
 * splits a grapheme, such as an emoji or a letter with an accent.
 * @param label - The text to fit.
 * @param maxWidth - The width to fit, in the units that measure returns.
 * @param measure - Returns the width of a string.
 * @returns label when it fits, else the longest start of label that fits with the
 * ellipsis. An empty string when not even the ellipsis fits.
 */
export const truncate = (
  label: string,
  maxWidth: number,
  measure: (label: string) => number,
): string => {
  if (measure(label) <= maxWidth) return label;
  if (measure(ELLIPSIS) > maxWidth) return "";
  const graphemes = Array.from(GRAPHEMES.segment(label), ({ segment }) => segment);
  const withEllipsis = (count: number): string =>
    graphemes.slice(0, count).join("") + ELLIPSIS;
  let fits = 0;
  let overflows = graphemes.length;
  while (overflows - fits > 1) {
    const mid = Math.floor((fits + overflows) / 2);
    if (measure(withEllipsis(mid)) <= maxWidth) fits = mid;
    else overflows = mid;
  }
  return withEllipsis(fits);
};
