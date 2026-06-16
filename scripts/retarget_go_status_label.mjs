// One-off: retarget LIVE x/go/status and x/go/label imports to core/pkg/service/*.
// Leaves migration snapshot imports (x/status/migrations, x/label/migrations) untouched.
import { readFileSync, writeFileSync } from "node:fs";
import { execSync } from "node:child_process";

const CORE = {
  status: "github.com/synnaxlabs/synnax/pkg/service/status",
  label: "github.com/synnaxlabs/synnax/pkg/service/label",
};
const X = {
  status: "github.com/synnaxlabs/x/status",
  label: "github.com/synnaxlabs/x/label",
};

const files = execSync(
  `grep -rln 'synnaxlabs/x/status"\\|synnaxlabs/x/label"' --include='*.go' core x/go 2>/dev/null || true`,
  { encoding: "utf8" },
)
  .trim()
  .split("\n")
  .filter(Boolean);

let changed = 0;
for (const file of files) {
  let src = readFileSync(file, "utf8");
  const orig = src;
  for (const kind of ["status", "label"]) {
    // Match an import spec line for the LIVE x package (exact path, not migrations).
    const xLine = new RegExp(
      `^\\s*(?:(\\w+)\\s+)?"${X[kind].replace(/\//g, "\\/")}"\\s*$`,
      "m",
    );
    const m = xLine.exec(src);
    if (m == null) continue;
    const xAlias = m[1] ?? kind; // default import name is the package name
    // Is the core package already imported? capture its local name.
    const coreLine = new RegExp(
      `^\\s*(?:(\\w+)\\s+)?"${CORE[kind].replace(/\//g, "\\/")}"\\s*$`,
      "m",
    );
    const cm = coreLine.exec(src);
    const canonical = cm != null ? (cm[1] ?? kind) : kind;

    if (cm != null) {
      // Core already imported: drop the x line entirely.
      src = src.replace(xLine, "").replace(/\n\n\n+/g, "\n\n");
    } else {
      // Repoint the x line to core, using the canonical (default) name.
      src = src.replace(xLine, `\t"${CORE[kind]}"`);
    }
    // Rename body references from the x alias to the canonical name.
    if (xAlias !== canonical)
      src = src.replace(new RegExp(`\\b${xAlias}\\.`, "g"), `${canonical}.`);
  }
  if (src !== orig) {
    writeFileSync(file, src);
    changed++;
  }
}
console.log(`retargeted ${changed} Go files`);
