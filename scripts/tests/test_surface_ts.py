#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import subprocess
from pathlib import Path

SCRIPT = (
    Path(__file__).resolve().parents[2] / "client" / "ts" / "scripts" / "surface.sh"
)

TSCONFIG = {
    "compilerOptions": {
        "declaration": True,
        "emitDeclarationOnly": True,
        "module": "esnext",
        "moduleResolution": "bundler",
        "rootDir": ".",
        "strict": True,
        "target": "es2022",
    },
    "include": ["src/**/*"],
}

TIME = """
export class TimeStamp { readonly ns = 0; }
export class TimeSpan { readonly span = 0; }
"""

INDEX = """
import { TimeStamp as T } from "./time";

/** Opens a valve. */
export const open = (at: T): boolean => at !== undefined;

export class Valve {
    private secret = 1;
    close(): void {}
}
"""


def report(root: Path, index: str = INDEX) -> str:
    (root / "src").mkdir(exist_ok=True)
    (root / "tsconfig.build.json").write_text(json.dumps(TSCONFIG))
    (root / "src" / "time.ts").write_text(TIME)
    (root / "src" / "index.ts").write_text(index)
    return subprocess.run(
        [str(SCRIPT), str(root)], capture_output=True, text=True, check=True
    ).stdout


class TestSurfaceTS:
    def test_prints_public_declarations(self, tmp_path: Path) -> None:
        out = report(tmp_path)
        assert "# src/index.d.ts\n" in out
        assert "export declare const open: (at: T) => boolean;" in out
        assert "close(): void;" in out
        assert "secret" not in out

    def test_ignores_comments_and_private_members(self, tmp_path: Path) -> None:
        before = report(tmp_path)
        after = report(
            tmp_path,
            INDEX.replace("Opens a valve.", "Opens it.").replace(
                "secret = 1", "hidden = 2"
            ),
        )
        assert after == before

    def test_changes_when_an_import_alias_changes(self, tmp_path: Path) -> None:
        before = report(tmp_path)
        after = report(tmp_path, INDEX.replace("TimeStamp as T", "TimeSpan as T"))
        assert after != before
