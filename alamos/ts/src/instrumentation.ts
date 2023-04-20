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
import { InstrumentationMeta } from "@/meta";

export interface InstrumentationOptions {
  key: string;
  serviceName?: string;
  logger: Logger;
  tracer: Tracer;
}

export class Instrumentation {
  private meta: InstrumentationMeta;
  readonly T: Tracer;
  readonly L: Logger;

  constructor({ key, serviceName, logger, tracer }: InstrumentationOptions) {
    this.meta = new InstrumentationMeta(key, "", serviceName);
    this.T = tracer;
    this.L = logger;
  }

  child(key: string): Instrumentation {
    const meta = this.meta.child(key);
    const ins = new Instrumentation({
      key: meta.key,
      logger: this.L.child(meta),
      tracer: this.T.child(meta),
    })
    ins.meta = meta;
    return ins
  }

}


