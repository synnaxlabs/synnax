// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { trace } from "@opentelemetry/api";
import { runtime } from "@synnaxlabs/x";

import { Instrumentation } from "@/instrumentation";
import { Logger } from "@/log";
import { Tracer } from "@/trace";

const DEV_DSN = "http://synnax_dev@localhost:14318/2";

export const instrumentation = (): Instrumentation => {
  const serviceName = "synnax";
  return new Instrumentation({
    key: "",
    serviceName,
    logger: new Logger(),
    tracer: new Tracer(trace.getTracerProvider().getTracer("synnax")),
  });
};
