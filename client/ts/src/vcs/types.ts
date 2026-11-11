// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeStamp, zod } from "@synnaxlabs/x";
import { z } from "zod";

import { listZ } from "@/vcs/list";

/**
 * A revision of the configuration: "working" (the live state of the Core), "head" (the
 * last commit), a full commit hash, or "configuration/<name>".
 */
export type Rev = string;

/** The live state of the Core. */
export const WORKING: Rev = "working";

/** The commit the live state of the Core is based on. */
export const HEAD: Rev = "head";

/** Returns the rev of the tip of the configuration with the given name. */
export const configurationRev = (name: string): Rev => `configuration/${name}`;

/** One file of a resource in the repository. */
export const fileZ = z.object({ path: z.string(), content: z.string() });
export interface File extends z.infer<typeof fileZ> {}

/**
 * The previous state of what an apply or discard changed on the Core. Write it back to
 * undo the change.
 */
export const inverseZ = z.object({
  files: listZ(fileZ),
  removed: listZ(z.string()),
  /** configuration is the configuration to track again, or empty to keep it. */
  configuration: z.string(),
});
export interface Inverse extends z.infer<typeof inverseZ> {}
export interface CrudeInverse extends z.input<typeof inverseZ> {}

export const changeKindZ = z.enum(["added", "modified", "deleted"]);
export type ChangeKind = z.infer<typeof changeKindZ>;

/** An open pull request that holds a change. */
export const reviewZ = z.object({ number: z.number(), url: z.string() });
export interface Review extends z.infer<typeof reviewZ> {}

/**
 * One changed resource. `resource` is the compact resource ID, such as
 * "schematic:6f1c...".
 */
export const changeZ = z.object({
  resource: z.string(),
  type: z.string(),
  name: z.string(),
  kind: changeKindZ,
  path: z.string(),
  /** digest changes whenever the newer files of the resource change. */
  digest: z.string(),
  review: zod.nullToUndefined(reviewZ),
});
export interface Change extends z.infer<typeof changeZ> {}

export const commitZ = z.object({
  hash: z.string(),
  shortHash: z.string(),
  message: z.string(),
  author: z.string(),
  time: TimeStamp.z,
  changes: zod.nullToUndefined(listZ(changeZ)),
});
export interface Commit extends z.infer<typeof commitZ> {}

export const statusZ = z.object({
  enabled: z.boolean(),
  configuration: z.string(),
  remoteUrl: z.string(),
  tokenSet: z.boolean(),
  head: zod.nullToUndefined(commitZ),
  changes: listZ(changeZ),
  incoming: listZ(commitZ),
  incomingChanges: listZ(changeZ),
  waiting: z.number(),
  fetchedAt: TimeStamp.z,
  error: z.string(),
});
export interface Status extends z.infer<typeof statusZ> {}

export const targetKindZ = z.enum(["pull", "restore", "load"]);
export type TargetKind = z.infer<typeof targetKindZ>;

/** What a plan or apply brings onto the Core. */
export const targetZ = z.object({
  kind: targetKindZ,
  commit: z.string().optional(),
  configuration: z.string().optional(),
  resources: z.string().array().optional(),
});
export interface Target extends z.infer<typeof targetZ> {}

export const planItemKindZ = z.enum(["create", "change", "delete"]);
export type PlanItemKind = z.infer<typeof planItemKindZ>;

export const planItemZ = z.object({
  resource: z.string(),
  type: z.string(),
  name: z.string(),
  kind: planItemKindZ,
  conflict: z.boolean(),
  running: z.boolean(),
  restarts: z.boolean(),
});
export interface PlanItem extends z.infer<typeof planItemZ> {}

export const planZ = z.object({
  target: targetZ.extend({ resources: zod.nullToUndefined(z.string().array()) }),
  items: listZ(planItemZ),
});
export interface Plan extends z.infer<typeof planZ> {}

export const choiceValueZ = z.enum(["mine", "incoming"]);
export type ChoiceValue = z.infer<typeof choiceValueZ>;

/** How to settle a conflict on one resource. */
export const choiceZ = z.object({ resource: z.string(), choice: choiceValueZ });
export interface Choice extends z.infer<typeof choiceZ> {}

/** A test configuration: a branch in the repository. */
export const configurationZ = z.object({
  name: z.string(),
  head: commitZ,
  current: z.boolean(),
});
export interface Configuration extends z.infer<typeof configurationZ> {}

/** A commit of the live configuration, tagged to a range. */
export const snapshotZ = z.object({
  range: z.string(),
  tag: z.string(),
  commit: commitZ,
  configuration: z.string(),
  uncommitted: z.number(),
});
export interface Snapshot extends z.infer<typeof snapshotZ> {}

export const connectRequestZ = z.object({
  configuration: z.string(),
  remoteUrl: z.string(),
  token: z.string(),
});
export interface ConnectRequest extends z.infer<typeof connectRequestZ> {}

export const commitRequestZ = z.object({
  message: z.string(),
  resources: z.string().array(),
  pullRequest: z.boolean(),
});
export interface CommitRequest extends z.infer<typeof commitRequestZ> {}

export const commitResponseZ = z.object({
  commit: commitZ,
  pullRequestUrl: z.string().default(""),
});
export interface CommitResponse extends z.infer<typeof commitResponseZ> {}

export const logRequestZ = z.object({
  resource: z.string().optional(),
  configuration: z.string().optional(),
  limit: z.number(),
});
export interface LogRequest extends z.input<typeof logRequestZ> {}
