// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Mutex as Core } from "async-mutex";

export type Mutex<G> = G & Core;

export class mutex<G> extends Core {
  constructor(guard: G) {
    super();
    Object.assign(this, guard);
  }
}

export const newMutex = <G>(guard: G): Mutex<G> => new mutex(guard) as Mutex<G>;
