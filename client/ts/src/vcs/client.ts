// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { z } from "zod";

import { listZ } from "@/vcs/list";
import {
  type Change,
  changeZ,
  type Choice,
  choiceZ,
  type Commit,
  type CommitRequest,
  commitRequestZ,
  type CommitResponse,
  commitResponseZ,
  commitZ,
  type Configuration,
  configurationZ,
  type ConnectRequest,
  connectRequestZ,
  type CrudeInverse,
  type File,
  fileZ,
  type Inverse,
  inverseZ,
  type LogRequest,
  logRequestZ,
  type Plan,
  planZ,
  type Rev,
  type Snapshot,
  snapshotZ,
  type Status,
  statusZ,
  type Target,
  targetZ,
} from "@/vcs/types";

const emptyZ = z.object({}).optional();
const resourcesReqZ = z.object({ resources: z.string().array() });
const planReqZ = z.object({ target: targetZ });
const applyReqZ = z.object({ target: targetZ, choices: choiceZ.array() });
const filesReqZ = z.object({ rev: z.string(), resource: z.string() });
const filesResZ = z.object({ files: listZ(fileZ) });
const logResZ = z.object({ commits: listZ(commitZ) });
const showReqZ = z.object({ commit: z.string() });
const showResZ = z.object({ commit: commitZ });
const configurationsResZ = z.object({ configurations: listZ(configurationZ) });
const cloneReqZ = z.object({ name: z.string() });
const compareReqZ = z.object({ from: z.string(), to: z.string() });
const compareResZ = z.object({ changes: listZ(changeZ) });
const rangeReqZ = z.object({ range: z.string() });
const snapshotResZ = z.object({ snapshot: snapshotZ });
const snapshotsResZ = z.object({ snapshots: listZ(snapshotZ) });
const writeResZ = z.unknown();

export interface ClientParams {
  unary: UnaryClient;
}

/**
 * Keeps the configuration of a Core in a Git repository the Core holds. The live state
 * of the Core is the working copy: edits show as uncommitted changes against the last
 * commit.
 */
export class Client {
  private readonly unary: UnaryClient;

  constructor({ unary }: ClientParams) {
    this.unary = unary;
  }

  /** Returns the version control status the Core last computed. */
  async status(): Promise<Status> {
    return await this.unary.send("/vcs/status", {}, emptyZ, statusZ);
  }

  /**
   * Starts version control, or changes the connection when it is already on.
   * @throws {AccessDeniedError} if the caller is not an owner.
   */
  async connect(req: ConnectRequest): Promise<Status> {
    return await this.unary.send("/vcs/connect", req, connectRequestZ, statusZ);
  }

  /** Fetches the remote, retries waiting pushes, and returns the new status. */
  async fetch(): Promise<Status> {
    return await this.unary.send("/vcs/fetch", {}, emptyZ, statusZ);
  }

  /**
   * Commits the live state of the given resources and pushes it. With `pullRequest`,
   * commits onto a new branch and opens a pull request instead.
   */
  async commit(req: CommitRequest): Promise<CommitResponse> {
    return await this.unary.send("/vcs/commit", req, commitRequestZ, commitResponseZ);
  }

  /**
   * Resets the given resources to the last commit.
   * @returns the previous state, to write back for undo.
   */
  async discard(resources: string[]): Promise<Inverse> {
    return await this.unary.send(
      "/vcs/discard",
      { resources },
      resourcesReqZ,
      inverseZ,
    );
  }

  /** Writes files back onto the Core, such as the inverse of an apply or discard. */
  async write(inverse: CrudeInverse): Promise<void> {
    await this.unary.send("/vcs/write", inverse, inverseZ, writeResZ);
  }

  /** Returns what applying the target would change on the Core. */
  async plan(target: Target): Promise<Plan> {
    return await this.unary.send("/vcs/plan", { target }, planReqZ, planZ);
  }

  /**
   * Applies the target in one transaction.
   * @param choices - one choice for every conflict in the plan.
   * @returns the previous state, to write back for undo.
   * @throws {ValidationError} if a conflict has no choice.
   */
  async apply(target: Target, choices: Choice[] = []): Promise<Inverse> {
    return await this.unary.send(
      "/vcs/apply",
      { target, choices },
      applyReqZ,
      inverseZ,
    );
  }

  /** Returns the files of a resource at a rev, or none if it does not exist there. */
  async files(rev: Rev, resource: string): Promise<File[]> {
    const res = await this.unary.send(
      "/vcs/files",
      { rev, resource },
      filesReqZ,
      filesResZ,
    );
    return res.files;
  }

  /** Returns the commits of a configuration, newest first. */
  async log(req: LogRequest): Promise<Commit[]> {
    const res = await this.unary.send("/vcs/log", req, logRequestZ, logResZ);
    return res.commits;
  }

  /** Returns a commit with its changes against its parent. */
  async show(commit: string): Promise<Commit> {
    const res = await this.unary.send("/vcs/show", { commit }, showReqZ, showResZ);
    return res.commit;
  }

  async configurations(): Promise<Configuration[]> {
    const res = await this.unary.send(
      "/vcs/configurations",
      {},
      emptyZ,
      configurationsResZ,
    );
    return res.configurations;
  }

  /**
   * Creates a configuration from the current one and tracks it. Nothing on the Core
   * changes.
   */
  async clone(name: string): Promise<Status> {
    return await this.unary.send("/vcs/clone", { name }, cloneReqZ, statusZ);
  }

  /** Returns the resources that differ between two revs. */
  async compare(from: Rev, to: Rev): Promise<Change[]> {
    const res = await this.unary.send(
      "/vcs/compare",
      { from, to },
      compareReqZ,
      compareResZ,
    );
    return res.changes;
  }

  /** Commits the live configuration, uncommitted changes included, to a range. */
  async snapshot(range: string): Promise<Snapshot> {
    const res = await this.unary.send(
      "/vcs/snapshot",
      { range },
      rangeReqZ,
      snapshotResZ,
    );
    return res.snapshot;
  }

  /** Returns the configuration snapshots of a range, newest first. */
  async snapshots(range: string): Promise<Snapshot[]> {
    const res = await this.unary.send(
      "/vcs/snapshots",
      { range },
      rangeReqZ,
      snapshotsResZ,
    );
    return res.snapshots;
  }
}
