// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type TimeStamp } from "@synnaxlabs/x";

// Brands instances through the global symbol registry so matches() holds
// across duplicated module instances (dual CJS/ESM builds), where
// instanceof does not.
const BRAND = Symbol.for("synnaxlabs.client.query.deleted");

/**
 * The last value of a deleted record, delivered wherever a cached answer
 * would be, so deletion can never be mistaken for live data. One instance
 * exists per deletion: repeated reads of a deleted answer return the same
 * reference.
 */
export class Deleted<D = unknown> {
  /** The record's value at the moment it was deleted. */
  readonly corpse: D;
  /** When the deletion was applied. */
  readonly deletedAt: TimeStamp;
  protected readonly [BRAND] = true;

  constructor(corpse: D, deletedAt: TimeStamp) {
    this.corpse = corpse;
    this.deletedAt = deletedAt;
  }

  /** Narrows a cached answer to the deleted case, carrying its record type
   *  onto the corpse. */
  static matches<D>(value: D | Deleted<D> | undefined): value is Deleted<D> {
    return typeof value === "object" && value !== null && BRAND in value;
  }
}
