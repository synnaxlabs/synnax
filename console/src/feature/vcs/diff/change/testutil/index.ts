// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Change } from "@/feature/vcs/diff/change";

/** Formats a property as one line of text, such as "Sample rate: 1000 -> 2500". */
export const format = (property: Change.Property): string => {
  const path = Change.formatPath(property);
  const { before, after } = property;
  switch (property.kind) {
    case "added":
      return after == null ? `${path}: added` : `${path}: ${Change.text(after)}`;
    case "removed":
      return before == null ? `${path}: removed` : `${path}: ${Change.text(before)}`;
    case "changed":
      if (before == null || after == null) return `${path}: changed`;
      return `${path}: ${Change.text(before)} -> ${Change.text(after)}`;
  }
};
