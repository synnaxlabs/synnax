#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""What every runner is working on and what is waiting to be picked up. Job starts and
finishes arrive as workflow_job deliveries; step transitions have no webhook, so the
runs that are running are polled for them."""

import re
from collections import Counter
from concurrent.futures import Executor, Future
from datetime import datetime

import synnax as sy

from github import GitHub
from log import Log
from scaler import Pool, pool_of

# Distinct runs polled for step detail per cycle. At a 5 s cycle this is 3600 calls an
# hour, which leaves room under GitHub's 5000 for the runner status poll.
STEP_RUNS_PER_CYCLE = 5
# How often the queued set is rebuilt from the API, so a delivery GitHub never landed
# cannot leave the scaler acting on a job that no longer exists.
RESYNC_INTERVAL = 5 * sy.TimeSpan.MINUTE


def view(job: dict) -> dict:
    """The fields of a GitHub job that <runner>_events reports on."""
    return {
        "id": job["id"],
        "run_id": job["run_id"],
        "name": job["name"],
        "url": job["html_url"],
        "branch": job["head_branch"],
        "workflow": job["workflow_name"],
        "steps": job["steps"],
    }


def duration(start: str | None, end: str | None) -> str:
    if not start or not end:
        return "?"
    span = sy.TimeStamp(datetime.fromisoformat(end)).span(
        sy.TimeStamp(datetime.fromisoformat(start))
    )
    return str(span.trunc(sy.TimeSpan.SECOND)) if span >= sy.TimeSpan.SECOND else "<1s"


def line(view: dict, *, tail: str) -> str:
    """<workflow> <step> <ticket or branch> | <url>"""
    # A run name can trail the workflow name, as in "Test - Integration 2886/merge".
    workflow = re.split(r"\s+(?=\S*[\d/])", view["workflow"])[0]
    workflow = re.sub(r"\s+", "", workflow)
    branch = view["branch"]
    if key := re.match(r"[A-Za-z]+-\d+", branch):
        branch = key.group()
    return f"{workflow} {tail} {branch} | {view['url']}"


def step_events(view: dict, *, before: list[dict], after: list[dict]) -> list[str]:
    seen = {s["number"]: (s["status"], s["conclusion"]) for s in before}
    lines = []
    for s in after:
        if (s["status"], s["conclusion"]) == seen.get(s["number"]):
            continue
        if s["status"] == "in_progress":
            lines.append(line(view, tail=f"{s['name']} started"))
        elif s["status"] == "completed" and s["conclusion"] != "skipped":
            taken = duration(s["started_at"], s["completed_at"])
            lines.append(line(view, tail=f"{s['name']} {s['conclusion']} {taken}"))
    return lines


def started_line(view: dict) -> str:
    return line(view, tail="started")


def completed_line(job: dict) -> str:
    taken = duration(job.get("started_at"), job.get("completed_at"))
    return line(view(job), tail=f"done {job['conclusion']} {taken}")


class Jobs:
    """Holds the job each runner is running and the labels of every queued job. Both
    are driven by deliveries; the step poll doubles as the reconciler, finishing a job
    whose completed delivery never arrived."""

    def __init__(self, *, github: GitHub, log: Log, pool: Executor) -> None:
        self.github = github
        self.log = log
        self.pool = pool
        self.syncing: Future[tuple[dict[int, set[str]], list[dict]]] | None = None
        self.active: dict[str, dict] = {}
        self.queued: dict[int, set[str]] = {}
        # Rotates the step poll over the active runs when there are more than a cycle
        # can afford.
        self.cursor = 0
        self.synced: sy.TimeStamp | None = None
        # Set once a queue read has landed, so the queue holds every waiting job.
        self.ready = False
        # Deliveries applied while a resync reads, replayed over its older snapshot.
        self.pending: list[dict] = []

    def queue(self, job: dict, *, action: object) -> None:
        if action == "queued":
            self.queued[job["id"]] = set(job.get("labels", []))
        else:
            self.queued.pop(job["id"], None)

    def apply(self, deliveries: list[dict]) -> dict[str, list[str]]:
        """Folds in a batch of workflow_job deliveries, returning the event lines they
        produced per runner."""
        events: dict[str, list[str]] = {}
        for payload in deliveries:
            job = payload.get("workflow_job")
            if not isinstance(job, dict):
                continue
            action = payload.get("action")
            self.queue(job, action=action)
            if self.syncing is not None:
                self.pending.append(payload)
            if action == "queued":
                continue
            runner = job.get("runner_name")
            if runner is None:
                continue
            current = view(job)
            if action == "in_progress":
                self.active[runner] = current
                lines = events.setdefault(runner, [])
                lines.append(started_line(current))
                lines.extend(step_events(current, before=[], after=job["steps"]))
            elif action == "completed":
                last = self.active.get(runner)
                lines = events.setdefault(runner, [])
                # A late completion of an earlier job leaves the current one active.
                if last is not None and last["id"] == job["id"]:
                    del self.active[runner]
                    lines.extend(
                        step_events(current, before=last["steps"], after=job["steps"])
                    )
                lines.append(completed_line(job))
        return {r: lines for r, lines in events.items() if lines}

    def poll(self) -> dict[str, list[str]]:
        """Refreshes step detail for the runs that are running, a slice at a time."""
        runs = sorted({j["run_id"] for j in self.active.values()})
        if not runs:
            self.cursor = 0
            return {}
        take = min(STEP_RUNS_PER_CYCLE, len(runs))
        self.cursor %= len(runs)
        chosen = (runs + runs)[self.cursor : self.cursor + take]
        self.cursor += take
        events: dict[str, list[str]] = {}
        for run in chosen:
            try:
                jobs = self.github.run_jobs(run)
            except Exception as e:
                self.log.error(f"run {run} jobs", e)
                continue
            for job in jobs:
                self.absorb(job, events=events)
        return {r: lines for r, lines in events.items() if lines}

    def absorb(self, job: dict, *, events: dict[str, list[str]]) -> None:
        runner = job.get("runner_name")
        last = self.active.get(runner) if runner is not None else None
        if runner is None or last is None or last["id"] != job["id"]:
            return
        lines = events.setdefault(runner, [])
        lines.extend(step_events(view(job), before=last["steps"], after=job["steps"]))
        if job["status"] == "completed":
            # The completed delivery never arrived, so finish the job from the poll.
            del self.active[runner]
            lines.append(completed_line(job))
        else:
            self.active[runner] = view(job)

    def counts(self) -> dict[Pool, int]:
        """Queued jobs per pool, what the scaler starts instances against."""
        pools: Counter[Pool] = Counter()
        for labels in self.queued.values():
            pool = pool_of(labels)
            if pool is not None:
                pools[pool] += 1
        return dict(pools)

    def resync(self, now: sy.TimeStamp) -> None:
        """Rebuilds the queued jobs from the API and adds the running jobs that no
        delivery reported, at most once per RESYNC_INTERVAL. The API reads run on pool,
        since hundreds of runs take minutes, and land on the first call after they
        finish."""
        if self.syncing is None:
            if self.synced is not None and now.span(self.synced) < RESYNC_INTERVAL:
                return
            # Stamped before the calls, so a failure waits out the interval instead of
            # retrying every cycle.
            self.synced = now
            self.syncing = self.pool.submit(self.fetch)
        if not self.syncing.done():
            return
        syncing, self.syncing = self.syncing, None
        pending, self.pending = self.pending, []
        try:
            queued, running = syncing.result()
        except Exception as e:
            self.log.error("queued resync", e)
            return
        self.queued = queued
        self.ready = True
        for payload in pending:
            self.queue(payload["workflow_job"], action=payload.get("action"))
        done = {
            p["workflow_job"]["id"] for p in pending if p.get("action") == "completed"
        }
        # Restores the jobs that started before a restart.
        for job in running:
            if job["id"] not in done:
                self.active.setdefault(job["runner_name"], view(job))

    def fetch(self) -> tuple[dict[int, set[str]], list[dict]]:
        queued: dict[int, set[str]] = {}
        running: list[dict] = []
        for run in self.github.unfinished_runs():
            for job in self.github.run_jobs(run):
                if job["status"] == "queued":
                    queued[job["id"]] = set(job.get("labels", []))
                elif job["status"] == "in_progress" and job.get("runner_name"):
                    running.append(job)
        return queued, running
