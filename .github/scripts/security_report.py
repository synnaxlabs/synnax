#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Builds the private security report from the scans that security_scan.sh wrote and,
with --agents, the encrypted agent reports.

Usage: security_report.py <out> [--agents] [--publish]

--agents reads AGENT_REPORTS, the JSON of the agent jobs' results and outputs, and
decrypts each report with the GPG private key in REPORT_PRIVATE_KEY. Writes the report
to <out>/report.md. --publish also files it as a Linear issue in the SY team, when it
holds a finding or a failed scan, and posts the count and the issue link to Slack.
Prints no finding, because logs on a public repository are public. Exits non-zero when a
scan or an agent failed.

Environment for --publish: LINEAR_API_KEY, SLACK_WEBHOOK_URL, and RUN_URL.
"""

import argparse
import json
import os
import re
import subprocess
import sys
import tempfile
import urllib.request
from collections import defaultdict
from collections.abc import Callable
from dataclasses import dataclass
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

LINEAR_URL = "https://api.linear.app/graphql"
TEAM = "SY"
# govulncheck marks a vulnerable module that no package imports as a note.
COUNTED = {"error", "warning"}
# Findings listed per rule, which keeps the report within a Linear description.
LISTED = 25
# Characters in the whole report, which keeps it within a Linear description.
LIMIT = 60_000
SEVERITY = re.compile(r"^### (Critical|High|Medium|Low): (.+)$", re.MULTILINE)

# Sends a JSON payload to a URL with headers and returns the response body.
Send = Callable[[str, dict[str, Any], dict[str, str]], bytes]


@dataclass(frozen=True)
class Finding:
    rule: str
    level: str
    location: str
    message: str


@dataclass
class Scan:
    name: str
    exit_code: int
    findings: list[Finding]
    log: str
    body: str = ""


def read_findings(sarif: dict[str, Any]) -> list[Finding]:
    """Returns the counted results of a SARIF log, without duplicates."""
    findings: dict[Finding, None] = {}
    for run in sarif.get("runs") or []:
        for result in run.get("results") or []:
            level = result.get("level", "warning")
            if level not in COUNTED:
                continue
            location = ""
            if result.get("locations"):
                physical = result["locations"][0].get("physicalLocation", {})
                location = physical.get("artifactLocation", {}).get("uri", "")
                if line := physical.get("region", {}).get("startLine"):
                    location += f":{line}"
            finding = Finding(
                rule=result.get("ruleId", ""),
                level=level,
                location=location,
                message=" ".join(result.get("message", {}).get("text", "").split()),
            )
            findings[finding] = None
    return list(findings)


def read_scan(out: Path, name: str) -> Scan:
    exit_code = int((out / f"{name}.exit").read_text())
    log = (out / f"{name}.log").read_text().strip()
    findings = []
    if exit_code == 0:
        findings = read_findings(json.loads((out / f"{name}.sarif").read_text()))
    return Scan(name, exit_code, findings, log)


def read_scans(out: Path) -> list[Scan]:
    return [read_scan(out, p.stem) for p in sorted(out.glob("*.exit"))]


def render_scan(scan: Scan) -> list[str]:
    if scan.exit_code != 0:
        return [
            f"## {scan.name}: failed with exit code {scan.exit_code}",
            "",
            "```",
            scan.log[-4000:],
            "```",
            "",
        ]
    lines = [f"## {scan.name}: {len(scan.findings)} findings", ""]
    if scan.body:
        return [*lines, scan.body, ""]
    by_rule: dict[tuple[str, str], list[Finding]] = defaultdict(list)
    for f in scan.findings:
        by_rule[(f.rule, f.level)].append(f)
    for (rule, level), findings in sorted(by_rule.items()):
        lines += [f"### `{rule}` ({level}): {len(findings)}", ""]
        lines += [f"- `{f.location}`: {f.message[:300]}" for f in findings[:LISTED]]
        if len(findings) > LISTED:
            lines.append(f"- And {len(findings) - LISTED} more.")
        lines.append("")
    return lines


def gpg(home: str, *args: str, text: str) -> str:
    cmd = ["gpg", "--homedir", home, "--batch", "--quiet", *args]
    done = subprocess.run(cmd, input=text, capture_output=True, text=True)
    if done.returncode != 0:
        raise RuntimeError(f"gpg failed: {done.stderr.strip()}")
    return done.stdout


def read_agents(jobs: dict[str, Any], key: str) -> list[Scan]:
    """Decrypts the report of each agent job, keyed by area, with the private key."""
    scans = []
    # A short home, since the gpg-agent socket path has a length limit.
    with tempfile.TemporaryDirectory(dir="/tmp") as home:
        try:
            gpg(home, "--import", text=key)
            imported = ""
        except RuntimeError as e:
            imported = str(e)
        for area, job in sorted(jobs.items()):
            name = f"agent-{area}"
            armor = job.get("outputs", {}).get("report", "")
            if imported:
                scans.append(Scan(name, 1, [], imported))
                continue
            if job["result"] != "success" or not armor:
                log = f"The agent job ended with {job['result']} and no report."
                scans.append(Scan(name, 1, [], log))
                continue
            try:
                body = gpg(home, "--decrypt", text=armor).strip()
            except RuntimeError as e:
                scans.append(Scan(name, 1, [], str(e)))
                continue
            findings = [
                Finding(m[1], m[1].lower(), "", m[2]) for m in SEVERITY.finditer(body)
            ]
            scans.append(Scan(name, 0, findings, "", body))
        subprocess.run(["gpgconf", "--homedir", home, "--kill", "all"], check=True)
    return scans


def render(scans: list[Scan], run_url: str) -> str:
    total = sum(len(s.findings) for s in scans)
    lines = [f"Findings: {total}.", ""]
    if run_url:
        lines += [f"Run: {run_url}", ""]
    for scan in scans:
        lines += render_scan(scan)
    report = "\n".join(lines)
    if len(report) <= LIMIT:
        return report
    cut = report.rfind("\n", 0, LIMIT - 200)
    omitted = report[cut:].count("\n- ")
    return (
        f"{report[:cut]}\n\nThe report stops at the size limit of a Linear issue. "
        f"It omits {len(report) - cut} characters and {omitted} listed findings."
    )


def post(url: str, payload: dict[str, Any], headers: dict[str, str]) -> bytes:
    request = urllib.request.Request(
        url,
        data=json.dumps(payload).encode(),
        headers={"Content-Type": "application/json", **headers},
    )
    with urllib.request.urlopen(request, timeout=30) as response:
        body: bytes = response.read()
    return body


def linear(
    send: Send, key: str, query: str, variables: dict[str, Any]
) -> dict[str, Any]:
    payload = {"query": query, "variables": variables}
    body = json.loads(send(LINEAR_URL, payload, {"Authorization": key}))
    if body.get("errors"):
        # The errors can quote the report, so they stay out of the log.
        raise RuntimeError("the Linear API returned errors")
    data: dict[str, Any] = body["data"]
    return data


def file_issue(send: Send, key: str, title: str, description: str) -> str:
    """Files an issue in the SY team and returns its URL."""
    teams = linear(
        send,
        key,
        "query($key: String!) { teams(filter: {key: {eq: $key}}) { nodes { id } } }",
        {"key": TEAM},
    )["teams"]["nodes"]
    created = linear(
        send,
        key,
        "mutation($input: IssueCreateInput!) "
        "{ issueCreate(input: $input) { issue { url } } }",
        {
            "input": {
                "teamId": teams[0]["id"],
                "title": title,
                "description": description,
            }
        },
    )
    url: str = created["issueCreate"]["issue"]["url"]
    return url


def publish(send: Send, scans: list[Scan], report: str, env: dict[str, str]) -> None:
    total = sum(len(s.findings) for s in scans)
    failed = sum(s.exit_code != 0 for s in scans)
    text = f"Weekly security audit: {total} findings, {failed} failed scans."
    if total or failed:
        date = datetime.now(UTC).date().isoformat()
        title = f"Security audit {date}: {total} findings"
        text += " " + file_issue(send, env["LINEAR_API_KEY"], title, report)
    send(env["SLACK_WEBHOOK_URL"], {"text": text}, {})


def main(argv: list[str], send: Send = post) -> int:
    p = argparse.ArgumentParser()
    p.add_argument("out", type=Path)
    p.add_argument("--agents", action="store_true")
    p.add_argument("--publish", action="store_true")
    args = p.parse_args(argv)
    scans = read_scans(args.out)
    if not scans:
        sys.exit(f"{args.out} holds no scan")
    if args.agents:
        jobs = json.loads(os.environ["AGENT_REPORTS"])
        scans = read_agents(jobs, os.environ["REPORT_PRIVATE_KEY"]) + scans
    report = render(scans, os.environ.get("RUN_URL", ""))
    (args.out / "report.md").write_text(report)
    if args.publish:
        publish(send, scans, report, dict(os.environ))
    failed = [s.name for s in scans if s.exit_code != 0]
    if failed:
        print(f"Failed scans: {', '.join(failed)}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
