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
