#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pathlib import Path

from surface_cpp import report

HEADER = """
#pragma once
#include <string>

namespace synnax::thing {
struct Thing;

/// @brief the URL. // not a marker
const std::string URL = "http://localhost";

enum class Mode : uint8_t {
    /// @brief the first mode.
    First,
    Second = 2,
};

struct Thing {
    int count = 1;
    std::string name{"thing"};
    [[nodiscard]] int doubled() const { return this->count * 2; }
};

/// @brief a client.
class Client {
    int implicit = 0;

public:
    Client(
        int a,
        int b
    ):
        a(a), b{b}, c(a + b) {
        this->open();
    }

    [[nodiscard]] std::pair<Thing, x::errors::Error>
    retrieve(const std::string &name, int limit = {}) const;

    friend std::ostream &operator<<(std::ostream &os, const Client &c);

private:
    int a;
    void open() {}
};
}
"""

EXPECTED = """\
client/cpp/thing/thing.h synnax::thing:: const std::string URL = "http://localhost"
client/cpp/thing/thing.h synnax::thing:: enum class Mode : uint8_t {}
client/cpp/thing/thing.h synnax::thing:: struct Thing {}
client/cpp/thing/thing.h synnax::thing:: class Client {}
client/cpp/thing/thing.h synnax::thing::Client:: Client(int a, int b)
client/cpp/thing/thing.h synnax::thing::Client:: [[nodiscard]] std::pair<Thing, \
x::errors::Error> retrieve(const std::string &name, int limit = {}) const
client/cpp/thing/thing.h synnax::thing::Mode:: First
client/cpp/thing/thing.h synnax::thing::Mode:: Second = 2
client/cpp/thing/thing.h synnax::thing::Thing:: [[nodiscard]] int doubled() const
client/cpp/thing/thing.h synnax::thing::Thing:: int count = 1
client/cpp/thing/thing.h synnax::thing::Thing:: std::string name{"thing"}
"""


def run(tmp_path: Path, header: str, *, name: str = "thing/thing.h") -> str:
    path = tmp_path / "client" / "cpp" / name
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(header)
    return report(tmp_path)


class TestReport:
    def test_lists_public_declarations_with_their_scope(self, tmp_path: Path) -> None:
        lines = run(tmp_path, HEADER).splitlines()
        assert sorted(lines) == sorted(EXPECTED.splitlines())

    def test_ignores_bodies_comments_private_members_and_wrapping(
        self, tmp_path: Path
    ) -> None:
        edited = (
            HEADER.replace(
                "return this->count * 2;", "return this->count + this->count;"
            )
            .replace("the first mode", "the mode that comes first")
            .replace("int a;", "int a;\n    int d;")
            .replace("this->open();", "")
            .replace("    retrieve(const", "    retrieve(\n        const")
        )
        assert run(tmp_path, edited) == run(tmp_path, HEADER)

    def test_reports_a_changed_signature(self, tmp_path: Path) -> None:
        edited = HEADER.replace("int limit = {}", "int limit = 5")
        assert run(tmp_path, edited) != run(tmp_path, HEADER)

    def test_reports_a_new_member(self, tmp_path: Path) -> None:
        edited = HEADER.replace("int count = 1;", "int count = 1;\n    int total;")
        assert "Thing:: int total\n" in run(tmp_path, edited)

    def test_skips_codecs_and_test_utilities(self, tmp_path: Path) -> None:
        run(tmp_path, "struct Codec {};", name="thing/proto.gen.h")
        run(tmp_path, "struct Codec {};", name="thing/json.gen.h")
        assert run(tmp_path, "struct Fake {};", name="testutil/testutil.h") == ""

    def test_keeps_generated_types(self, tmp_path: Path) -> None:
        assert "struct Thing" in run(
            tmp_path, "struct Thing {};", name="thing/types.gen.h"
        )
