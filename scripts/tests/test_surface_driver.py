#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pathlib import Path

from surface_driver import report

TASK = """
/// @brief configuration for the task manager.
struct ManagerConfig {
    /// @brief number of worker threads.
    size_t worker_count = 4;

    template<typename Parser>
    void override(Parser &p) {
        // the worker count is clamped.
        this->worker_count = p.field("worker_count", this->worker_count);
    }

    friend std::ostream &operator<<(std::ostream &os, const ManagerConfig &cfg);
};

struct Other {
    int ignored = 1;
};
"""

EXEC = """
int exec(int argc, char *argv[]) {
    auto args = x::args::Parser(argc, argv);
    if (args.flag("--debug")) set_debug();
    if (command == "start") {
        if (args.flag("--standalone", "-s")) return start(args);
    }
    auto path = args.field<std::string>("--state-file", "");
    return 1;
}
"""


def write(root: Path, task: str = TASK) -> str:
    (root / "driver" / "task").mkdir(parents=True, exist_ok=True)
    (root / "driver" / "cmd").mkdir(parents=True, exist_ok=True)
    (root / "driver" / "task" / "task.h").write_text(task)
    (root / "driver" / "cmd" / "exec.cpp").write_text(EXEC)
    (root / "driver" / "cmd" / "exec_test.cpp").write_text('a.flag("--test-only");')
    return report(root)


class TestReport:
    """Tests the report of a small Driver tree and how edits change it."""

    def test_lists_commands_flags_keys_and_members(self, tmp_path: Path) -> None:
        assert write(tmp_path) == (
            "driver/cmd/exec.cpp command start\n"
            "driver/cmd/exec.cpp field --state-file\n"
            "driver/cmd/exec.cpp flag --debug\n"
            "driver/cmd/exec.cpp flag --standalone -s\n"
            "driver/task/task.h ManagerConfig member size_t worker_count = 4\n"
            "driver/task/task.h field worker_count\n"
        )

    def test_ignores_comments(self, tmp_path: Path) -> None:
        before = write(tmp_path)
        after = write(tmp_path, TASK.replace("worker threads", "threads"))
        assert after == before

    def test_changes_when_a_default_changes(self, tmp_path: Path) -> None:
        before = write(tmp_path)
        after = write(tmp_path, TASK.replace("= 4", "= 8"))
        assert after != before

    def test_changes_when_a_brace_initialized_default_changes(
        self, tmp_path: Path
    ) -> None:
        braced = TASK.replace("worker_count = 4;", "worker_count{4};")
        before = write(tmp_path, braced)
        after = write(tmp_path, braced.replace("{4}", "{8}"))
        assert "ManagerConfig member size_t worker_count{4}" in before
        assert after != before

    def test_lists_the_defaults_resolve_sets(self, tmp_path: Path) -> None:
        rack = tmp_path / "driver" / "rack"
        rack.mkdir(parents=True)
        (rack / "rack.h").write_text(
            "static Config resolve(Parser &p) {\n"
            '    Config cfg{.connection = {.host = "localhost", .port = 9090}};\n'
            "    return cfg;\n"
            "}\n"
        )
        out = write(tmp_path)
        assert "driver/rack/rack.h resolve .port = 9090\n" in out
        assert 'driver/rack/rack.h resolve .host = "localhost"\n' in out

    def test_lists_built_integrations_and_their_conditions(
        self, tmp_path: Path
    ) -> None:
        rack = tmp_path / "driver" / "rack"
        rack.mkdir(parents=True)
        src = (
            "inline std::vector<std::string> built_integrations() {\n"
            "    std::vector<std::string> integrations = {\n"
            "        opcua::INTEGRATION_NAME,\n"
            "        http::INTEGRATION_NAME\n"
            "    };\n"
            "#ifndef SYNNAX_NILINUXRT\n"
            "    integrations.push_back(modbus::INTEGRATION_NAME);\n"
            "#endif\n"
            "    return integrations;\n"
            "}\n"
        )
        (rack / "rack.h").write_text(src)
        before = write(tmp_path)
        assert [line for line in before.splitlines() if "rack.h" in line] == [
            "driver/rack/rack.h built_integrations http::INTEGRATION_NAME",
            "driver/rack/rack.h built_integrations modbus::INTEGRATION_NAME"
            " #ifndef SYNNAX_NILINUXRT",
            "driver/rack/rack.h built_integrations opcua::INTEGRATION_NAME",
        ]
        (rack / "rack.h").write_text(src.replace(" integrations", " names"))
        assert write(tmp_path) == before

    def test_lists_the_integrations_of_each_platform(self, tmp_path: Path) -> None:
        rack = tmp_path / "driver" / "rack"
        rack.mkdir(parents=True)
        (rack / "rack.h").write_text(
            "std::vector<std::string> platform_integrations();\n"
        )
        (rack / "factories_windows.cpp").write_text(
            "std::vector<std::string> platform_integrations() {\n"
            "    return {ni::INTEGRATION_NAME, labjack::INTEGRATION_NAME};\n"
            "}\n"
        )
        assert [
            line for line in write(tmp_path).splitlines() if "platform" in line
        ] == [
            "driver/rack/factories_windows.cpp platform_integrations"
            " labjack::INTEGRATION_NAME",
            "driver/rack/factories_windows.cpp platform_integrations"
            " ni::INTEGRATION_NAME",
        ]
