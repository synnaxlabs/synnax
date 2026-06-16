import { readFileSync, writeFileSync } from "node:fs";

const esc = (s) => s.replace(/\//g, "\\/");

const fix = (file, pkgPath, qual) => {
  let s = readFileSync(file, "utf8");
  // Remove the self-import line (optional alias before the quoted path).
  s = s.replace(new RegExp(`^\\s*(?:\\w+\\s+)?"${esc(pkgPath)}"\\s*\\n`, "m"), "");
  // Strip the package qualifier so the symbols resolve locally.
  s = s.replace(new RegExp(`\\b${qual}\\.`, "g"), "");
  writeFileSync(file, s);
};

const SP = "github.com/synnaxlabs/synnax/pkg/service/status";
const LP = "github.com/synnaxlabs/synnax/pkg/service/label";
for (const f of ["migrate", "retrieve", "status", "service"])
  fix(`core/pkg/service/status/${f}.go`, SP, "status");
fix("core/pkg/service/label/retrieve.go", LP, "label");
console.log("fixed self-imports");
