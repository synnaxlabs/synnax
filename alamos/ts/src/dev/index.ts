// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import uptrace from "@uptrace/web"

import { trace, propagation } from "@opentelemetry/api";
propagation.inject

import { Instrumentation } from "@/instrumentation";
import { Logger } from "@/log";
import { Tracer } from "@/trace";

const DEV_DSN = "http://synnax_dev@localhost:14317/2"

export const instrumentation = () => {
  const serviceName = "synnax"
  uptrace.configureOpentelemetry({
    dsn: DEV_DSN,
    serviceName,
    deploymentEnvironment: "dev"
  })
  return new Instrumentation({
    key: "",
    serviceName,
    logger: new Logger(
    ),
    tracer: new Tracer(trace.getTracerProvider().getTracer("synnax")),
  })
}
