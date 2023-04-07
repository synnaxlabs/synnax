#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Callable

from opentelemetry.propagators.textmap import CarrierT, Setter, Getter

from alamos.instrumentation import Instrumentation


class _Setter(Setter):
    f: Callable[[CarrierT, str, str], None]

    def __init__(self, f: Callable[[CarrierT, str, str], None]):
        self.f = f

    def set(self, carrier: CarrierT, key: str, value: str) -> None:
        self.f(carrier, key, value)


def propagate(
    ins: Instrumentation,
    carrier: CarrierT,
    setter: Callable[[CarrierT, str, str], None],
):
    ins.t.propagator.inject(carrier, setter=_Setter(setter))


def depropagate(
    ins: Instrumentation,
    carrier: CarrierT,
    getter: Getter[CarrierT],
):
    ins.t.propagator.extract(carrier)
