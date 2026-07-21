// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { errors } from "@synnaxlabs/x";

export class FluxError extends errors.createTyped("flux") {}

/**
 * Thrown to the error boundary when a query's cached answer is a deletion.
 * Carries the last value of the deleted record.
 */
export class DeletedError<D = unknown> extends FluxError.sub("deleted") {
  readonly corpse: D;

  constructor(message: string, corpse: D) {
    super(message);
    this.corpse = corpse;
  }
}
