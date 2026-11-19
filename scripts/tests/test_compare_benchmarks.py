#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

import compare_benchmarks as cb

PACKAGE = "example.com/pkg"


def table(unit: str, *rows: str, package: str = PACKAGE) -> str:
    """Returns one table in the CSV form that benchstat prints."""
    header = [f"pkg: {package}", ",base,,head,,,", f",{unit},CI,{unit},CI,vs base,P"]
    return "\n".join([*header, *rows, "geomean,1,,1,,+0.00%,", "", ""])


@pytest.mark.parametrize(
    ("unit", "row", "want"),
    [
        (
            "sec/op",
            "Write-8,1e-06,1%,1.06e-06,1%,+6.00%,p=0.000 n=10",
            [cb.Finding("slower", PACKAGE, "Write-8", "+6.00%")],
        ),
        ("sec/op", "Write-8,1e-06,9%,1.06e-06,9%,~,p=0.300 n=10", []),
        ("sec/op", "Write-8,1e-06,1%,1.04e-06,1%,+4.00%,p=0.000 n=10", []),
        ("sec/op", "Write-8,1e-06,1%,9e-07,1%,-10.00%,p=0.000 n=10", []),
        (
            "allocs/op",
            "Write-8,300,0%,301,0%,+0.33%,p=0.000 n=10",
            [cb.Finding("allocations", PACKAGE, "Write-8", "300 to 301 allocs/op")],
        ),
        (
            "allocs/op",
            "Write-8,0,0%,1,0%,?,p=0.000 n=10",
            [cb.Finding("allocations", PACKAGE, "Write-8", "0 to 1 allocs/op")],
        ),
        ("allocs/op", "Write-8,40,9%,42,9%,~,p=0.400 n=10", []),
        ("allocs/op", "Write-8,4,0%,3,0%,-25.00%,p=0.000 n=10", []),
        ("sec/op", "Gone-8,5e-09,0%", [cb.Finding("missing", PACKAGE, "Gone-8")]),
        ("sec/op", "New-8,,,5e-09,0%", [cb.Finding("new", PACKAGE, "New-8")]),
        ("allocs/op", "Gone-8,3,0%", []),
        ("B/op", "Write-8,512,0%,1024,0%,+100.00%,p=0.000 n=10", []),
        ("samples/sec", "Write-8,100,0%,200,0%,+100.00%,p=0.000 n=10", []),
    ],
)
def test_judge_applies_the_rule(unit: str, row: str, want: list[cb.Finding]) -> None:
    assert cb.judge(table(unit, row)) == want


def test_judge_names_the_package_of_each_table() -> None:
    slow = "Write-8,1e-06,1%,2e-06,1%,+100.00%,p=0.000 n=10"
    csv = (
        'goos: linux\n"cpu: A CPU, 2.5GHz"\n'
        + table("sec/op", slow, package="example.com/a")
        + table("allocs/op", "Write-8,1,0%,2,0%,+100.00%,p=0.000 n=10")
    )
    assert cb.judge(csv) == [
        cb.Finding("slower", "example.com/a", "Write-8", "+100.00%"),
        cb.Finding("allocations", PACKAGE, "Write-8", "1 to 2 allocs/op"),
    ]
