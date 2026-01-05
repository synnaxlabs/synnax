// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { trace } from "@opentelemetry/api";

import { Instrumentation } from "@/instrumentation";
import { Logger } from "@/log";
import { Tracer } from "@/trace";

export const instrumentation = (): Instrumentation => {
  const serviceName = "synnax";
  return new Instrumentation({
    key: "",
    serviceName,
    logger: new Logger(),
    tracer: new Tracer(trace.getTracerProvider().getTracer("synnax")),
  });
};
