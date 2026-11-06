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

PREFIX = "client/cpp/thing/thing.h synnax::thing"
EXPECTED = [
    f'{PREFIX}:: const std::string URL = "http://localhost"',
    f"{PREFIX}:: enum class Mode : uint8_t {{}}",
    f"{PREFIX}::Mode:: First",
    f"{PREFIX}::Mode:: Second = 2",
    f"{PREFIX}:: struct Thing {{}}",
    f"{PREFIX}::Thing:: int count = 1",
    f'{PREFIX}::Thing:: std::string name{{"thing"}}',
    f"{PREFIX}::Thing:: [[nodiscard]] int doubled() const",
    f"{PREFIX}:: class Client {{}}",
    f"{PREFIX}::Client:: Client(int a, int b)",
    f"{PREFIX}::Client:: [[nodiscard]] std::pair<Thing, x::errors::Error> "
    "retrieve(const std::string &name, int limit = {}) const",
    f"{PREFIX}::Client:: friend std::ostream &operator<<(std::ostream &os, "
    "const Client &c)",
]


def run(tmp_path: Path, header: str, *, name: str = "thing/thing.h") -> str:
    path = tmp_path / "client" / "cpp" / name
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(header)
    return report(tmp_path)


class TestReport:
    def test_lists_public_declarations_with_their_scope(self, tmp_path: Path) -> None:
        assert run(tmp_path, HEADER).splitlines() == EXPECTED

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

    def test_ignores_wrapping_inside_templates_and_braces(self, tmp_path: Path) -> None:
        wrapped = "const std::vector<\n    int\n> VALUES = {\n    1,\n    2\n};"
        one_line = "const std::vector<int> VALUES = {1, 2};"
        assert run(tmp_path, wrapped) == run(tmp_path, one_line)

    def test_reports_reordered_enum_entries(self, tmp_path: Path) -> None:
        edited = HEADER.replace(
            "First,\n    Second = 2,", "Second = 2,\n    First,"
        ).replace("/// @brief the first mode.\n", "")
        assert run(tmp_path, edited) != run(tmp_path, HEADER)

    def test_reports_reordered_members(self, tmp_path: Path) -> None:
        edited = HEADER.replace(
            'int count = 1;\n    std::string name{"thing"};',
            'std::string name{"thing"};\n    int count = 1;',
        )
        assert run(tmp_path, edited) != run(tmp_path, HEADER)

    def test_reports_changed_spacing_inside_a_string(self, tmp_path: Path) -> None:
        one = run(tmp_path, 'const std::string PREFIX = "Bearer ";')
        two = run(tmp_path, 'const std::string PREFIX = "Bearer  ";')
        assert one != two

    def test_reports_a_friend_function_in_a_private_section(
        self, tmp_path: Path
    ) -> None:
        text = run(
            tmp_path,
            "class A {\n    friend bool operator==(A a, A b) { return true; }\n};",
        )
        assert "A:: friend bool operator==(A a, A b)" in text

    def test_skips_a_friend_type(self, tmp_path: Path) -> None:
        assert "Other" not in run(
            tmp_path, "class A {\npublic:\n    friend class Other;\n};"
        )

    def test_skips_an_initializer_with_a_brace_inside_parentheses(
        self, tmp_path: Path
    ) -> None:
        text = run(
            tmp_path,
            'struct A {\n    A(int a): b{a}, c(Bar{a}), d("{") {}\n'
            "    int get() const { return 1; }\n};",
        )
        assert text.splitlines() == [
            "client/cpp/thing/thing.h struct A {}",
            "client/cpp/thing/thing.h A:: A(int a)",
            "client/cpp/thing/thing.h A:: int get() const",
        ]

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
