#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import types
import warnings

import pytest

from x.deprecation import deprecated_getattr


def _make_module(
    deprecated: dict[str, str | tuple[str, str]],
    **globals_entries: object,
) -> types.ModuleType:
    """Create a fake module with deprecated_getattr configured."""
    mod = types.ModuleType("test_module")
    mod.__dict__.update(globals_entries)
    mod.__dict__["__getattr__"] = deprecated_getattr(
        "test_module", deprecated, mod.__dict__
    )
    return mod


class TestDeprecatedGetattr:
    def test_emits_deprecation_warning(self) -> None:
        mod = _make_module({"OldName": "NewName"}, NewName="value")
        with warnings.catch_warnings(record=True) as w:
            warnings.simplefilter("always", DeprecationWarning)
            result = mod.OldName
        assert result == "value"
        assert len(w) == 1
        assert issubclass(w[0].category, DeprecationWarning)
        assert "OldName is deprecated, use NewName instead" in str(w[0].message)

    def test_returns_correct_value(self) -> None:
        sentinel = object()
        mod = _make_module({"Old": "New"}, New=sentinel)
        with warnings.catch_warnings(record=True):
            warnings.simplefilter("always", DeprecationWarning)
            assert mod.Old is sentinel

    def test_caches_after_first_access(self) -> None:
        mod = _make_module({"Old": "New"}, New="value")
        with warnings.catch_warnings(record=True):
            warnings.simplefilter("always", DeprecationWarning)
            _ = mod.Old
        with warnings.catch_warnings(record=True) as w:
            warnings.simplefilter("always", DeprecationWarning)
            _ = mod.Old
        assert len(w) == 0

    def test_raises_attribute_error_for_unknown(self) -> None:
        mod = _make_module({}, NewName="value")
        with pytest.raises(AttributeError, match="test_module"):
            _ = mod.NonExistent

    def test_tuple_form_custom_display_name(self) -> None:
        mod = _make_module(
            {"OldName": ("package.module.NewName", "_internal")},
            _internal="value",
        )
        with warnings.catch_warnings(record=True) as w:
            warnings.simplefilter("always", DeprecationWarning)
            result = mod.OldName
        assert result == "value"
        assert len(w) == 1
        assert "use package.module.NewName instead" in str(w[0].message)

    def test_tuple_form_caches(self) -> None:
        mod = _make_module(
            {"OldName": ("pkg.New", "_internal")},
            _internal="value",
        )
        with warnings.catch_warnings(record=True):
            warnings.simplefilter("always", DeprecationWarning)
            _ = mod.OldName
        with warnings.catch_warnings(record=True) as w:
            warnings.simplefilter("always", DeprecationWarning)
            _ = mod.OldName
        assert len(w) == 0

    def test_multiple_deprecated_names(self) -> None:
        mod = _make_module(
            {"OldA": "NewA", "OldB": "NewB"},
            NewA="a",
            NewB="b",
        )
        with warnings.catch_warnings(record=True) as w:
            warnings.simplefilter("always", DeprecationWarning)
            assert mod.OldA == "a"
            assert mod.OldB == "b"
        assert len(w) == 2
        assert "OldA" in str(w[0].message)
        assert "OldB" in str(w[1].message)

    def test_non_deprecated_access_unaffected(self) -> None:
        mod = _make_module({"Old": "New"}, New="value", Other="other")
        assert mod.Other == "other"
        assert mod.New == "value"
