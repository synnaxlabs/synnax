import uptrace
from opentelemetry.propagate import get_global_textmap
from opentelemetry.trace import get_tracer_provider

from alamos import Instrumentation, Tracer

DSN = "http://synnax_dev@localhost:14317/2"


def instrumentation() -> Instrumentation:
    uptrace.configure_opentelemetry(
        dsn=DSN,
        service_name="dog",
        deployment_environment="dev",
    )
    return Instrumentation(
        key="alamos",
        service_name="alamos",
        tracer=Tracer(
            otel_provider=get_tracer_provider(),
            otel_propagator=get_global_textmap(),
        )
    )
