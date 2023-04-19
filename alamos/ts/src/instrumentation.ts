/* eslint-disable prettier/prettier */
// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Logger } from "@/log";
import { Tracer } from "@/trace";

export class Instrumentation {
  readonly T: Tracer;
  readonly L: Logger;

  constructor(tracer: Tracer, logger: Logger) {
    this.T = tracer;
    this.L = logger;
  }
}
