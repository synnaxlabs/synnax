// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Logger } from "@/log";
import { Meta } from "@/meta";
import { Tracer } from "@/trace";

export interface InstrumentationOptions {
  key?: string;
  serviceName?: string;
  logger?: Logger;
  tracer?: Tracer;
  noop?: boolean;
  __meta?: Meta;
}

export class Instrumentation {
  private readonly meta: Meta;
  readonly T: Tracer;
  readonly L: Logger;

  constructor({
    key = "",
    serviceName = "",
    logger = Logger.NOOP,
    tracer = Tracer.NOOP,
    noop = false,

    __meta,
  }: InstrumentationOptions) {
    this.meta = __meta ?? new Meta(key, key, serviceName, noop);
    this.T = tracer.child(this.meta);
    this.L = logger.child(this.meta);
  }

  child(key: string): Instrumentation {
    const __meta = this.meta.child(key);
    return new Instrumentation({ __meta, tracer: this.T, logger: this.L });
  }

  static readonly NOOP = new Instrumentation({ noop: true });
}

export const NOOP = Instrumentation.NOOP;
