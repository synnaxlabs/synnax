#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import asyncio
import inspect
import warnings
from collections.abc import Callable

import numpy as np
import pytest

from x.exceptions import Error
from x.params import deprecate_positional, require_named_params
from x.telem import Density, MultiSeries, Rate, Series, Size, TimeRange, TimeSpan
from x.telem.telem import convert_time_units
from x.timing import Loop, poll, sleep


class TestRequireNamedParams:
    def test_allows_named_params(self) -> None:
        @require_named_params
        def greet(name: str = "") -> str:
            return f"hello {name}"

        assert greet(name="world") == "hello world"

    def test_raises_on_positional(self) -> None:
        @require_named_params
        def greet() -> str:
            return "hello"

        with pytest.raises(TypeError, match="only accepts named") as exc:
            greet("a", "b")  # type: ignore[call-arg]
        assert isinstance(exc.value, Error)

    def test_with_example_params(self) -> None:
        @require_named_params(example_params=("user_id", "12345"))
        def lookup() -> str:
            return ""

        with pytest.raises(TypeError, match="user_id='12345'"):
            lookup("a", "b")  # type: ignore[call-arg]

    def test_non_positional_type_error_propagates(self) -> None:
        @require_named_params
        def bad(*, n: int) -> int:
            return n + "oops"  # type: ignore[operator]

        with pytest.raises(TypeError, match="unsupported operand"):
            bad(n=1)


RELEASED_POSITIONAL_FORMS: dict[str, Callable[[], object]] = {
    "Series": lambda: Series([1, 2], "float32"),  # type: ignore[call-arg]
    "Series.to_numpy": lambda: Series([1, 2]).to_numpy(None, True),  # type: ignore[call-overload]
    "MultiSeries.to_numpy": lambda: MultiSeries([Series([1])]).to_numpy(None, True),  # type: ignore[call-arg]
    "Rate.byte_size": lambda: Rate(1).byte_size(TimeSpan.SECOND, Density.BIT64),  # type: ignore[call-arg]
    "Rate.size_span": lambda: Rate(1).size_span(Size(64), Density.BIT64),  # type: ignore[call-arg]
    "TimeRange.overlaps_with": lambda: TimeRange(start=0, end=10).overlaps_with(
        TimeRange(start=5, end=15),
        1,  # type: ignore[call-arg]
    ),
    "convert_time_units": lambda: convert_time_units(np.array([1]), "s", "ms"),  # type: ignore[call-arg]
    "sleep": lambda: sleep(0, False),  # type: ignore[call-arg]
    "Loop": lambda: Loop(TimeSpan.MILLISECOND, False),  # type: ignore[call-arg]
    "poll": lambda: poll(lambda: 1, TimeSpan.SECOND, TimeSpan.MILLISECOND),  # type: ignore[call-arg]
}


class TestDeprecatePositional:
    def test_keyword_calls_do_not_warn(self) -> None:
        @deprecate_positional("b")
        def f(a: int, *, b: int = 0) -> tuple[int, int]:
            return a, b

        with warnings.catch_warnings():
            warnings.simplefilter("error")
            assert f(1, b=2) == (1, 2)
            assert f(1) == (1, 0)

    def test_positional_calls_warn_and_pass_by_keyword(self) -> None:
        @deprecate_positional("b", "c")
        def f(a: int, *, b: int = 0, c: int = 0) -> tuple[int, int, int]:
            return a, b, c

        with pytest.warns(DeprecationWarning, match="Passing b, c to .*f\\(\\)"):
            assert f(1, 2, 3) == (1, 2, 3)  # type: ignore[call-arg]
        with pytest.warns(DeprecationWarning, match="Passing b to"):
            assert f(1, 2, c=3) == (1, 2, 3)  # type: ignore[call-arg]

    def test_too_many_positional_arguments_raise(self) -> None:
        @deprecate_positional("b")
        def f(a: int, *, b: int = 0) -> int:
            return a + b

        with pytest.raises(TypeError, match="positional"):
            f(1, 2, 3)  # type: ignore[call-arg]

    def test_positional_and_keyword_value_raise(self) -> None:
        @deprecate_positional("b")
        def f(a: int, *, b: int = 0) -> int:
            return a + b

        with (
            pytest.warns(DeprecationWarning, match="Passing b"),
            pytest.raises(TypeError),
        ):
            f(1, 2, b=2)  # type: ignore[misc]

    def test_unknown_name_raises(self) -> None:
        with pytest.raises(ValueError, match="keyword-only"):

            @deprecate_positional("a")
            def f(a: int) -> int:
                return a

    def test_positional_name_after_keyword_only_name_reorders(self) -> None:
        @deprecate_positional("parent", "name", "key")
        def f(name: str, *, parent: str = "", key: int = 0) -> tuple[str, str, int]:
            return name, parent, key

        assert f("n") == ("n", "", 0)
        with pytest.warns(DeprecationWarning, match="Passing parent, name to"):
            assert f("p", "n") == ("n", "p", 0)  # type: ignore[call-arg]
        with pytest.warns(DeprecationWarning, match="Passing parent, name, key to"):
            assert f("p", "n", 1) == ("n", "p", 1)  # type: ignore[call-arg]
        with pytest.warns(DeprecationWarning, match="Passing parent to"):
            assert f("p", name="n") == ("n", "p", 0)  # type: ignore[misc]
        assert f(name="n", parent="p") == ("n", "p", 0)

    def test_positional_name_without_keyword_only_name_raises(self) -> None:
        with pytest.raises(ValueError, match="keyword-only"):

            @deprecate_positional("a", "b")
            def f(a: int, *, b: int = 0) -> int:
                return a + b

    def test_names_pass_through_kwargs_without_none(self) -> None:
        @deprecate_positional("a", "b")
        def f(
            *, a: int | None = None, **kwargs: int
        ) -> tuple[int | None, dict[str, int]]:
            return a, kwargs

        with pytest.warns(DeprecationWarning, match="Passing a, b to"):
            assert f(None, 2) == (None, {"b": 2})  # type: ignore[call-arg]
        with pytest.warns(DeprecationWarning, match="Passing a, b to"):
            assert f(1, None) == (1, {})  # type: ignore[call-arg]

    def test_unknown_name_without_kwargs_raises(self) -> None:
        with pytest.raises(ValueError, match="keyword-only"):

            @deprecate_positional("b")
            def f(a: int) -> int:
                return a

    def test_coroutine_functions_stay_coroutine_functions(self) -> None:
        @deprecate_positional("b")
        async def f(a: int, *, b: int = 0) -> int:
            return a + b

        assert inspect.iscoroutinefunction(f)
        with pytest.warns(DeprecationWarning, match="Passing b"):
            assert asyncio.run(f(1, 2)) == 3  # type: ignore[call-arg]

    @pytest.mark.parametrize(
        "call",
        RELEASED_POSITIONAL_FORMS.values(),
        ids=RELEASED_POSITIONAL_FORMS.keys(),
    )
    def test_released_positional_forms_warn(self, call: Callable[[], object]) -> None:
        """Positional forms that 0.60 accepted should still work and warn."""
        with pytest.warns(DeprecationWarning, match="by position is deprecated"):
            call()
