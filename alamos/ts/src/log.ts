// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { InstrumentationMeta } from "./meta";

export class Logger {
  noop: boolean;

  constructor(noop: boolean = false) {
    this.noop = noop;
  }

  child(_: InstrumentationMeta): Logger {
    if (this.noop) return NOOP_LOGGER;
    return new Logger();
  }


  debug(msg: string): void {
    if (this.noop) return;
    console.log(msg);
  }

  info(msg: string): void {
    if (this.noop) return;
    console.log(msg);
  }

  warn(msg: string): void {
    if (this.noop) return;
    console.warn(msg);
  }

  error(msg: string): void {
    if (this.noop) return;
    console.error(msg);
  }
}

const NOOP_LOGGER = new Logger(true);
