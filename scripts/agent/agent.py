#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Runs Claude Code or Codex headless on this repository and delivers the result.

Usage: scripts/agent/run (--prompt TEXT | --prompt-file PATH) [--agent claude|codex]
    [--model MODEL] [--max-turns N] [--timeout-minutes N]
    [--output pr|comment|report] [--pr NUMBER] [--title TITLE] [--labels a,b]
    [--image PATH]... [--draft] [--dry-run] [--recipient KEY]

Outputs:
    pr       Commit the agent's work and open a PR from `agent/<agent>/<slug>`. With
             --pr, behave as comment.
    comment  Push the agent's commits to the branch of --pr, then post its final
             message as a comment there.
    report   Write the final message to stdout, the step summary, and the `report`
             step output. The agent may push and open PRs itself, as the App.

--recipient, with report, keeps the report private on a public repository: it writes
only the report encrypted to the GPG public key file KEY, as the `report` step output.
The agent gets no GitHub token, and its stderr, CLI state, and output files are
discarded.

The last stdout line is the PR URL for pr and comment, and is also written as `pr-url`
to $GITHUB_OUTPUT when it is set. --dry-run runs the agent but pushes, comments, and
opens nothing.

Environment: AGENT_DEFAULT picks the agent when --agent is absent. GH_TOKEN is the
GitHub App installation token for `gh` and for Git pushes. AGENT_GIT_NAME and
AGENT_GIT_EMAIL are the bot identity for commits. A Claude Code or Codex login
authenticates the CLI. pr and comment need a clean checkout.
"""

import argparse
import base64
import os
import re
import secrets
import shutil
import signal
import subprocess
import sys
import tempfile
import urllib.parse
from collections.abc import Sequence
from dataclasses import dataclass, field
from pathlib import Path

AGENTS = ("claude", "codex")
OUTPUTS = ("pr", "comment", "report")
BASE = "main"
ASSETS_BRANCH = "agent-assets"
IMAGE_SUFFIXES = {".png", ".jpg", ".jpeg", ".gif", ".webp"}
TEMPLATE = Path(".github/pull_request_template.md")
LINEAR_URL = "https://linear.app/synnax/issue/"
# Claude Code otherwise appends a co-author trailer, which the repository rules forbid.
CLAUDE_SETTINGS = '{"attribution":{"commit":"","pr":""}}'


@dataclass(kw_only=True)
class Options:
    prompt: str
    agent: str
    output: str
    model: str | None = None
    max_turns: int | None = None
    timeout_minutes: int = 60
    pr: int | None = None
    title: str | None = None
    labels: list[str] = field(default_factory=list)
    images: list[Path] = field(default_factory=list)
    draft: bool = False
    dry_run: bool = False
    bin: str | None = None
    recipient: Path | None = None


def parse(argv: Sequence[str], *, env: dict[str, str]) -> Options:
    p = argparse.ArgumentParser(prog="scripts/agent/run")
    prompt = p.add_mutually_exclusive_group(required=True)
    prompt.add_argument("--prompt")
    prompt.add_argument("--prompt-file", type=Path)
    p.add_argument("--agent", choices=AGENTS, default=env.get("AGENT_DEFAULT") or None)
    p.add_argument("--model")
    p.add_argument("--max-turns", type=int)
    p.add_argument("--timeout-minutes", type=int, default=60)
    p.add_argument("--output", choices=OUTPUTS, default="pr")
    p.add_argument("--pr", type=int)
    p.add_argument("--title")
    p.add_argument("--labels", default="")
    p.add_argument("--image", type=Path, action="append", default=[])
    p.add_argument("--draft", action="store_true")
    p.add_argument("--dry-run", action="store_true")
    p.add_argument("--bin", help="path to the agent CLI, if not on PATH")
    p.add_argument("--recipient", type=Path, help="GPG public key file")
    a = p.parse_args(argv)
    if a.agent is None:
        p.error("--agent is required when AGENT_DEFAULT is not set")
    if a.agent not in AGENTS:
        p.error(f"AGENT_DEFAULT must be one of {AGENTS}, not {a.agent!r}")
    if a.output == "comment" and a.pr is None:
        p.error("--output comment needs --pr")
    if a.agent == "codex" and a.max_turns is not None:
        p.error("codex has no turn limit: use --timeout-minutes")
    if a.recipient is not None and a.output != "report":
        p.error("--recipient needs --output report")
    text = a.prompt if a.prompt is not None else a.prompt_file.read_text()
    if not text.strip():
        p.error("the prompt is empty")
    return Options(
        prompt=text.strip(),
        agent=a.agent,
        output=a.output,
        model=a.model,
        max_turns=a.max_turns,
        timeout_minutes=a.timeout_minutes,
        pr=a.pr,
        title=a.title,
        labels=split_labels(a.labels),
        images=[path.resolve() for path in a.image],
        draft=a.draft,
        dry_run=a.dry_run,
        bin=str(Path(a.bin).resolve()) if a.bin and os.sep in a.bin else a.bin,
        recipient=a.recipient.resolve() if a.recipient else None,
    )


def split_labels(raw: str) -> list[str]:
    return [label.strip() for label in re.split(r"[,\n]", raw) if label.strip()]


def slug(text: str, *, words: int = 6) -> str:
    tokens = re.findall(r"[a-z0-9]+", text.lower())
    return "-".join(tokens[:words]) or "task"


def branch_name(*, agent: str, title: str, suffix: str) -> str:
    return f"agent/{agent}/{slug(title)}-{suffix}"


def default_title(prompt: str) -> str:
    line = prompt.splitlines()[0].strip().lstrip("/")
    return f"[agent] {line}"[:72]


def skill_prompt(prompt: str, *, root: Path) -> str:
    """Spells out a slash skill for Codex, which does not read `.claude/skills`."""
    m = re.fullmatch(r"/([\w-]+)(?:\s+(.*))?", prompt, flags=re.S)
    if m is None:
        return prompt
    skill = Path(".claude/skills") / m[1] / "SKILL.md"
    if not (root / skill).is_file():
        return prompt
    args = f" Arguments: {m[2]}" if m[2] else ""
    return f"Follow the instructions in `{skill}`.{args}"


def instructions(opts: Options, *, out: Path) -> str:
    lines = ["You run headless in CI. Nobody answers questions: decide and proceed."]
    if opts.dry_run:
        lines.append("This is a dry run: never push, open a PR, or comment on GitHub.")
    elif opts.output != "report":
        lines += [
            "Never push, open a PR, or comment on GitHub: the wrapper does that.",
            f"Save screenshots worth showing (PNG, JPEG, GIF) to `{out}/images/`.",
        ]
    if opts.pr is not None and opts.output != "report":
        lines += [
            "Commit your work on the current branch: it is pushed to the PR.",
            "Your final message is posted as a PR comment. Keep it short.",
        ]
    elif opts.output == "pr":
        lines += [
            "Commit your work on the current branch with short messages.",
            f"Write a PR title to `{out}/title.txt` (`SY-####: Sentence case`).",
            f"Write the PR description to `{out}/description.md`: what changed and"
            " why, short.",
            f"Write the Linear issue key (SY-####), if any, to `{out}/issue.txt`.",
            f"Write one review tier label (see CONTRIBUTING.md) to `{out}/labels.txt`.",
        ]
    elif opts.recipient is not None:
        lines.append(
            "Your final message is a private report. Never commit, push, or write to"
            " GitHub or any other service."
        )
    else:
        lines.append("Your final message is the report. Keep it short.")
    return "\n".join(f"- {line}" for line in lines)


def agent_command(opts: Options, *, prompt: str, out: Path) -> list[str]:
    if opts.agent == "claude":
        cmd = [opts.bin or "claude", "-p", "--output-format", "text"]
        cmd += ["--dangerously-skip-permissions", "--settings", CLAUDE_SETTINGS]
        if opts.max_turns is not None:
            cmd += ["--max-turns", str(opts.max_turns)]
    else:
        cmd = [
            opts.bin or "codex",
            "exec",
            "--dangerously-bypass-approvals-and-sandbox",
        ]
        cmd += ["-c", 'project_doc_fallback_filenames=["CLAUDE.md"]']
        cmd += ["--output-last-message", str(out / "last-message.md")]
    if opts.model:
        cmd += ["--model", opts.model]
    return [*cmd, prompt]


def render_body(
    template: str, *, issue: str, description: str, labels: list[str]
) -> str:
    body = re.sub(r"<!--.*?-->\n?", "", template, flags=re.S)
    link = f"[{issue}]({LINEAR_URL}{issue})" if issue else "None"
    body = body.replace("[SY-####](<>)", link)
    body = body.replace("## Description\n", f"## Description\n\n{description}\n", 1)
    tiers = ", ".join(f"`{label}`" for label in labels if label.startswith("review/"))
    body = body.replace("## Review tier\n", f"## Review tier\n\n{tiers}\n", 1)
    return re.sub(r"\n{3,}", "\n\n", body).strip() + "\n"


def stored_names(paths: list[Path]) -> list[str]:
    """Names images uniquely, since two paths can share a base name."""
    return [f"{i + 1}-{re.sub(r'[^\w.-]', '_', p.name)}" for i, p in enumerate(paths)]


def image_markdown(urls: dict[str, str]) -> str:
    return "\n\n".join(f"![{name}]({url})" for name, url in urls.items())


def read(path: Path) -> str:
    return path.read_text().strip() if path.is_file() else ""


@dataclass(kw_only=True)
class Runner:
    """Executes commands, and only prints the ones that write to GitHub in a dry run."""

    dry_run: bool
    env: dict[str, str]

    def run(self, cmd: list[str], *, writes: bool = False) -> str:
        if writes and self.dry_run:
            print("dry run:", " ".join(cmd[:6]), file=sys.stderr)
            return ""
        done = subprocess.run(
            cmd, check=True, text=True, stdout=subprocess.PIPE, env=self.env
        )
        return done.stdout.strip()

    def ok(self, cmd: list[str]) -> bool:
        return subprocess.run(cmd, env=self.env, capture_output=True).returncode == 0


def run_agent(
    cmd: list[str], *, env: dict[str, str], seconds: float, quiet: bool = False
) -> str:
    """Runs the agent in its own process group and returns its stdout. quiet discards
    its stderr.

    On a timeout or an interrupt, kills the whole group, so no build or server it
    started outlives it.
    """
    proc = subprocess.Popen(
        cmd,
        env=env,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.DEVNULL if quiet else None,
        start_new_session=True,
    )
    try:
        stdout, _ = proc.communicate(timeout=seconds)
    except BaseException:
        if sys.platform == "win32":
            proc.kill()
        else:
            os.killpg(proc.pid, signal.SIGKILL)
        proc.communicate()
        raise
    if proc.returncode != 0:
        raise subprocess.CalledProcessError(proc.returncode, cmd[:1])
    return stdout


def private_env(env: dict[str, str], *, out: Path) -> dict[str, str]:
    """Returns env with no GitHub token or step output files, and with the CLI state
    of the agent in out, so no later job on the runner can read the session. The CLI
    logins of the runner carry over."""
    dirs = {
        "CLAUDE_CONFIG_DIR": (".claude", out / "claude", (".credentials.json",)),
        "CODEX_HOME": (".codex", out / "codex", ("auth.json", "config.toml")),
    }
    for var, (default, private, names) in dirs.items():
        private.mkdir()
        home = Path(env.get(var) or Path.home() / default)
        for name in names:
            if (home / name).is_file():
                shutil.copy(home / name, private / name)
    hidden = {"GH_TOKEN", "GITHUB_TOKEN", "GITHUB_OUTPUT", "GITHUB_STEP_SUMMARY"}
    kept = {k: v for k, v in env.items() if k not in hidden}
    return {**kept, **{var: str(private) for var, (_, private, _) in dirs.items()}}


def encrypt(text: str, *, recipient: Path) -> str:
    """Encrypts text to the GPG public key file recipient, as ASCII armor."""
    with tempfile.TemporaryDirectory() as home:
        cmd = ["gpg", "--homedir", home, "--batch", "--armor", "--trust-model"]
        cmd += ["always", "--recipient-file", str(recipient), "--encrypt"]
        done = subprocess.run(cmd, input=text, capture_output=True, text=True)
    if done.returncode != 0:
        raise RuntimeError(f"gpg failed: {done.stderr.strip()}")
    return done.stdout


def upload_images(
    r: Runner, *, repo: str, prefix: str, paths: list[Path]
) -> dict[str, str]:
    """Commits images to the orphan assets branch and returns raw URLs by stored name.

    Uses a private index, so the working tree and the current branch stay untouched.
    """
    if not paths:
        return {}
    for _ in range(3):
        index = Path(tempfile.mkdtemp()) / "index"
        env = {**r.env, "GIT_INDEX_FILE": str(index)}
        g = Runner(dry_run=r.dry_run, env=env)
        ref = f"refs/heads/{ASSETS_BRANCH}"
        parent: list[str] = []
        if g.ok(["git", "ls-remote", "--exit-code", "origin", ref]):
            g.run(["git", "fetch", "--quiet", "origin", ref])
            g.run(["git", "read-tree", "FETCH_HEAD"])
            parent = ["-p", "FETCH_HEAD"]
        names = stored_names(paths)
        for path, name in zip(paths, names, strict=True):
            blob = g.run(["git", "hash-object", "-w", str(path)])
            info = f"100644,{blob},{prefix}/{name}"
            g.run(["git", "update-index", "--add", "--cacheinfo", info])
        tree = g.run(["git", "write-tree"])
        msg = f"Add images for {prefix}"
        commit = g.run(["git", "commit-tree", tree, *parent, "-m", msg])
        if r.dry_run or g.ok(["git", "push", "origin", f"{commit}:{ref}"]):
            base = f"https://raw.githubusercontent.com/{repo}/{commit}/{prefix}"
            return {n: f"{base}/{urllib.parse.quote(n)}" for n in names}
    raise RuntimeError(f"failed to push images to {ASSETS_BRANCH} three times")


def find_images(opts: Options, *, out: Path) -> list[Path]:
    found = sorted((out / "images").glob("*")) if (out / "images").is_dir() else []
    images = [*opts.images, *found]
    return [p for p in images if p.suffix.lower() in IMAGE_SUFFIXES]


def write_output(env: dict[str, str], *, name: str, value: str) -> None:
    """Appends a step output, in the delimited form that allows several lines."""
    if path := env.get("GITHUB_OUTPUT"):
        delimiter = f"EOF_{secrets.token_hex(8)}"
        with open(path, "a") as f:
            f.write(f"{name}<<{delimiter}\n{value}\n{delimiter}\n")


def run_url(env: dict[str, str]) -> str:
    if "GITHUB_RUN_ID" not in env:
        return ""
    server = env.get("GITHUB_SERVER_URL", "https://github.com")
    return f"{server}/{env['GITHUB_REPOSITORY']}/actions/runs/{env['GITHUB_RUN_ID']}"


def git_identity(env: dict[str, str]) -> dict[str, str]:
    name, email = env.get("AGENT_GIT_NAME"), env.get("AGENT_GIT_EMAIL")
    if not name or not email:
        return env
    return {
        **env,
        "GIT_AUTHOR_NAME": name,
        "GIT_AUTHOR_EMAIL": email,
        "GIT_COMMITTER_NAME": name,
        "GIT_COMMITTER_EMAIL": email,
    }


def git_auth(env: dict[str, str]) -> dict[str, str]:
    """Authenticates Git to GitHub with GH_TOKEN, through the environment only."""
    token = env.get("GH_TOKEN")
    if not token:
        return env
    basic = base64.b64encode(f"x-access-token:{token}".encode()).decode()
    return {
        **env,
        "GIT_CONFIG_COUNT": "1",
        "GIT_CONFIG_KEY_0": "http.https://github.com/.extraheader",
        "GIT_CONFIG_VALUE_0": f"AUTHORIZATION: basic {basic}",
    }


def main(argv: Sequence[str]) -> int:
    env = git_identity(dict(os.environ))
    opts = parse(argv, env=env)
    pushes = opts.output != "report"
    if pushes and not opts.dry_run:
        if not env.get("AGENT_GIT_NAME") or not env.get("AGENT_GIT_EMAIL"):
            sys.exit("AGENT_GIT_NAME and AGENT_GIT_EMAIL must name the bot identity")
        if not env.get("GH_TOKEN"):
            sys.exit("GH_TOKEN must hold the GitHub App installation token")
    root = Path(
        subprocess.check_output(
            ["git", "rev-parse", "--show-toplevel"], text=True
        ).strip()
    )
    os.chdir(root)
    if pushes and subprocess.check_output(["git", "status", "--porcelain"], text=True):
        sys.exit("The checkout has uncommitted changes, which would ship with the PR")
    # The runner deletes RUNNER_TEMP after each job, even a cancelled one.
    out = Path(tempfile.mkdtemp(prefix="agent-", dir=env.get("RUNNER_TEMP")))
    (out / "images").mkdir()
    r = Runner(dry_run=opts.dry_run, env=git_auth(env))
    repo = env.get("GITHUB_REPOSITORY", "synnaxlabs/synnax")

    head = ""
    if opts.pr is not None:
        view = r.run(
            [
                "gh",
                "pr",
                "view",
                str(opts.pr),
                "--json",
                "headRefName,isCrossRepository",
                "--jq",
                '"\\(.headRefName) \\(.isCrossRepository)"',
            ],
        )
        head, cross = view.split()
        if cross == "true":
            sys.exit(f"PR #{opts.pr} is from a fork: the agent cannot run on it")
        r.run(["gh", "pr", "checkout", str(opts.pr)])
    elif opts.output == "pr":
        topic = opts.title or opts.prompt.splitlines()[0]
        head = branch_name(agent=opts.agent, title=topic, suffix=secrets.token_hex(3))
        r.run(["git", "switch", "--quiet", "-c", head])
    start = r.run(["git", "rev-parse", "HEAD"])

    prompt = (
        opts.prompt if opts.agent == "claude" else skill_prompt(opts.prompt, root=root)
    )
    prompt = f"{prompt}\n\nHow to deliver:\n{instructions(opts, out=out)}"
    cmd = agent_command(opts, prompt=prompt, out=out)
    if opts.recipient is not None:
        try:
            env_private = {**private_env(env, out=out), "AGENT_OUT_DIR": str(out)}
            stdout = run_agent(
                cmd, env=env_private, seconds=opts.timeout_minutes * 60, quiet=True
            )
            message = read(out / "last-message.md") or stdout.strip()
            write_output(
                env, name="report", value=encrypt(message, recipient=opts.recipient)
            )
        finally:
            shutil.rmtree(out)
        return 0
    # A report agent delivers its own work, so it gets Git push access as the App.
    report = opts.output == "report" and not opts.dry_run
    agent_env = git_auth(env) if report else env
    agent_env = {**agent_env, "AGENT_OUT_DIR": str(out)}
    stdout = run_agent(cmd, env=agent_env, seconds=opts.timeout_minutes * 60)
    message = read(out / "last-message.md") or stdout.strip()

    images = find_images(opts, out=out)
    if opts.output == "report":
        if path := env.get("GITHUB_STEP_SUMMARY"):
            Path(path).write_text(message + "\n")
        write_output(env, name="report", value=message)
        print(message)
        return 0

    prefix = head.replace("/", "-") if head else f"run-{secrets.token_hex(4)}"
    urls = upload_images(r, repo=repo, prefix=prefix, paths=images)
    pictures = image_markdown(urls)
    footer = f"Agent: {opts.agent}. [Run]({run_url(env)})." if run_url(env) else ""

    if opts.output != "report":
        if r.run(["git", "status", "--porcelain"]):
            r.run(["git", "add", "--all"])
            msg = read(out / "title.txt") or opts.title or default_title(opts.prompt)
            r.run(["git", "commit", "--quiet", "-m", msg])
        changed = r.run(["git", "rev-parse", "HEAD"]) != start
        if not changed and opts.pr is None:
            print(f"The agent made no changes.\n\n{message}")
            return 0
        if changed:
            r.run(["git", "push", "origin", f"HEAD:refs/heads/{head}"], writes=True)

    if opts.pr is not None:
        body = "\n\n".join(p for p in (message, pictures, footer) if p)
        body_file = out / "comment.md"
        body_file.write_text(body + "\n")
        r.run(
            ["gh", "pr", "comment", str(opts.pr), "--body-file", str(body_file)],
            writes=True,
        )
        url = f"https://github.com/{repo}/pull/{opts.pr}"
    else:
        title = opts.title or read(out / "title.txt") or default_title(opts.prompt)
        description = read(out / "description.md") or message
        description = "\n\n".join(p for p in (description, pictures, footer) if p)
        labels = [*opts.labels, *split_labels(read(out / "labels.txt"))]
        issue = read(out / "issue.txt")
        body = render_body(
            TEMPLATE.read_text(), issue=issue, description=description, labels=labels
        )
        body_file = out / "body.md"
        body_file.write_text(body)
        cmd = [
            "gh",
            "pr",
            "create",
            "--base",
            BASE,
            "--head",
            head,
            "--title",
            title,
            "--body-file",
            str(body_file),
        ]
        cmd += [arg for label in dict.fromkeys(labels) for arg in ("--label", label)]
        if opts.draft:
            cmd.append("--draft")
        url = r.run(cmd, writes=True)
        if opts.dry_run:
            print(f"{title}\n\n{body}", file=sys.stderr)
    write_output(env, name="pr-url", value=url)
    print(url)
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
