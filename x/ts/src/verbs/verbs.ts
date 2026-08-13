// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** Conjugations of a verb used to phrase messages about an operation. */
export interface Verbs {
  present: string;
  past: string;
  participle: string;
}

export const RENAME: Verbs = {
  present: "rename",
  past: "renamed",
  participle: "renaming",
};

export const DELETE: Verbs = {
  present: "delete",
  past: "deleted",
  participle: "deleting",
};

export const UPDATE: Verbs = {
  present: "update",
  past: "updated",
  participle: "updating",
};

export const CREATE: Verbs = {
  present: "create",
  past: "created",
  participle: "creating",
};

export const SNAPSHOT: Verbs = {
  present: "snapshot",
  past: "snapshotted",
  participle: "snapshotting",
};

export const COPY: Verbs = {
  present: "copy",
  past: "copied",
  participle: "copying",
};

export const SET: Verbs = {
  present: "set",
  past: "set",
  participle: "setting",
};

export const SAVE: Verbs = {
  present: "save",
  past: "saved",
  participle: "saving",
};
