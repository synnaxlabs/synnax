#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import uptrace
from opentelemetry.propagate import get_global_textmap
from opentelemetry.trace import get_tracer_provider

from alamos import Instrumentation, Tracer

DEV_DSN = "http://synnax_dev@localhost:14317/2"


def instrumentation() -> Instrumentation:
    """:returns: development instrumentation that connects to a locally hosted uptrace
    server (see alamos/dev/README.md). It's best to only call this function once within
    a testing environment, such as using a session scoped fixture in conftest.py.
    """
    uptrace.configure_opentelemetry(
        dsn=DEV_DSN,
        service_name="synnax",
        deployment_environment="dev",
    )
    return Instrumentation(
        key="dev",
        service_name="synnax",
        tracer=Tracer(
            otel_provider=get_tracer_provider(),
            otel_propagator=get_global_textmap(),
        ),
    )
