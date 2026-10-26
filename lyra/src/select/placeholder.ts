// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { caseconv } from "@synnaxlabs/x";
import pluralize from "pluralize";

import { Input } from "@/input";

/**
 * @returns what an empty trigger reads. Beside a visible label the text prompts the
 * user to act; with no label it is the field's only name, so it reads as one.
 */
export const usePlaceholder = (resourceName: string): string =>
  Input.useLabelled() ? `Select ${resourceName}` : caseconv.capitalize(resourceName);

const ACRONYM = /^[A-Z]{2,}$/;
const IDENTIFIER = / [A-Za-z0-9]$/;

/**
 * @returns the plural of a resource name. A name ending in an acronym takes a lowercase
 * "s" ("PDOs"), where pluralize would give "PDOS". A name ending in a one-character
 * identifier drops it ("edge 1" gives "edges").
 */
export const plural = (resourceName: string): string => {
  if (IDENTIFIER.test(resourceName)) return plural(resourceName.slice(0, -2));
  if (ACRONYM.test(resourceName.split(" ").at(-1) ?? "")) return `${resourceName}s`;
  return pluralize.plural(resourceName);
};

/** @returns what the search field of a selection reads. */
export const searchPlaceholder = (resourceName: string): string =>
  `Search ${plural(resourceName)}...`;
