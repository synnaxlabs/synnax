// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import pluralize from "pluralize";

const ACRONYM = /(^|\s)[A-Z\d]{2,}$/;
const IDENTIFIER = / [A-Za-z\d]$/;

/**
 * @returns the plural of a name. A name that ends in an acronym takes a lowercase "s":
 * "ICD" becomes "ICDs", where pluralize gives "ICDS". A name that ends in a
 * one-character identifier drops it: "edge 1" becomes "edges".
 */
export const plural = (name: string): string => {
  if (IDENTIFIER.test(name)) return plural(name.slice(0, -2));
  return ACRONYM.test(name) ? `${name}s` : pluralize.plural(name);
};
