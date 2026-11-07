#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

import synnax as sy
from synnax.util import request_where, returns_one, sole, sole_or_none


class TestSole:
    """Tests for the helpers that return the one record of a single retrieve."""

    where = sy.label.Where(name=sy.filter.String(eq="a"))

    def test_sole(self) -> None:
        """Should return the only record."""
        assert sole("label", self.where, ["x"]) == "x"

    def test_sole_not_found(self) -> None:
        """Should describe the filter when no record matched."""
        with pytest.raises(
            sy.NotFoundError, match=r'^Found no label where name = "a"$'
        ):
            sole("label", self.where, [])

    def test_sole_not_found_without_filter(self) -> None:
        """Should leave out the filter when there is none."""
        with pytest.raises(sy.NotFoundError, match=r"^Found no label$"):
            sole("label", None, [])

    def test_sole_multiple_found(self) -> None:
        """Should describe the filter when more than one record matched."""
        with pytest.raises(
            sy.MultipleFoundError,
            match=r'^Expected one label where name = "a", but found 2$',
        ):
            sole("label", self.where, ["x", "y"])

    def test_sole_or_none(self) -> None:
        """Should return the only record, or None when no record matched."""
        assert sole_or_none("label", self.where, ["x"]) == "x"
        assert sole_or_none("label", self.where, []) is None

    def test_sole_or_none_multiple_found(self) -> None:
        """Should raise MultipleFoundError when more than one record matched."""
        with pytest.raises(sy.MultipleFoundError):
            sole_or_none("label", self.where, ["x", "y"])


class TestRequestWhere:
    """Tests for request_where."""

    def test_no_fields(self) -> None:
        """Should return the tree as it is when there are no fields."""
        tree = sy.label.Where(name=sy.filter.String(eq="a"))
        assert request_where(sy.label.Where, tree, {}) is tree
        assert request_where(sy.label.Where, None, {}) is None

    def test_fields(self) -> None:
        """Should read the fields as a partial record."""
        assert request_where(sy.label.Where, None, {"name": ["a"]}) == sy.label.Where(
            name=sy.filter.String(in_=["a"])
        )

    def test_join(self) -> None:
        """Should join the tree and the fields with an and."""
        tree = sy.label.Where(name=sy.filter.String(eq="a"))
        got = request_where(sy.label.Where, tree, {"name": {"starts_with": "b"}})
        assert got == sy.label.Where(
            and_=[tree, sy.label.Where(name=sy.filter.String(starts_with="b"))]
        )

    def test_empty_tree(self) -> None:
        """Should return the fields alone when the tree has no lines."""
        got = request_where(sy.label.Where, sy.label.Where(), {"name": "a"})
        assert got == sy.label.Where(name=sy.filter.String(eq="a"))

    def test_none(self) -> None:
        """Should reject a field set to None."""
        with pytest.raises(sy.ValidationError, match="name is None"):
            request_where(sy.label.Where, None, {"name": None})


class TestReturnsOne:
    """Tests for returns_one."""

    def test_plain(self) -> None:
        """Should return one record for a plain key or name."""
        assert returns_one({"key": 1}, False, None, 0)
        assert returns_one({"name": "a", "key": [1]}, False, None, 0)

    def test_list_or_operator(self) -> None:
        """Should return a list for a list or an operator."""
        assert not returns_one({"name": ["a"]}, False, None, 0)
        assert not returns_one({"name": {"starts_with": "a"}}, False, None, 0)
        assert not returns_one({"key": sy.filter.Set[int](eq=1)}, False, None, 0)

    def test_one(self) -> None:
        """Should return one record for any filter when one is set."""
        assert returns_one({}, True, None, 0)

    def test_page(self) -> None:
        """Should reject one record beside a limit or an offset."""
        with pytest.raises(sy.ValidationError, match="limit or offset"):
            returns_one({"key": 1}, False, 1, 0)
        with pytest.raises(sy.ValidationError, match="limit or offset"):
            returns_one({}, True, None, 2)
