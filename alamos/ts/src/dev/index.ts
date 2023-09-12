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
import uptrace from "@uptrace/web";

import { Instrumentation } from "@/instrumentation";
import { Logger } from "@/log";
import { Tracer } from "@/trace";

const configureOpentelemetry = (config: uptrace.Config): void => {
  if (runtime.RUNTIME === "browser") {
    uptrace.configureOpentelemetry(config);
  } else {
    // eslint-disable-next-line @typescript-eslint/no-var-requires
    const uptrace = require("@uptrace/node");
    uptrace.configureOpentelemetry(config);
  }
};

const DEV_DSN = "http://synnax_dev@localhost:14318/2";

export const instrumentation = (): Instrumentation => {
  const serviceName = "synnax";
  configureOpentelemetry({
    dsn: DEV_DSN,
    serviceName,
    deploymentEnvironment: "dev",
  });
  return new Instrumentation({
    key: "",
    serviceName,
    logger: new Logger(),
    tracer: new Tracer(trace.getTracerProvider().getTracer("synnax")),
  });
};
