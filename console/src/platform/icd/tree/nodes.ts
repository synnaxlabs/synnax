// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { icd } from "@synnaxlabs/client";
import type { record } from "@synnaxlabs/x";

import type { TreeField } from "@/platform/icd/read";

/** A source of the task, such as a topic or a record, at the root of the tree. */
export interface SourceNode {
  kind: "source";
  source: string;
}

/** A message of a source that reads several, in the tree under its source. */
export interface MessageNode {
  kind: "message";
  /** The key of the source. */
  source: string;
  /** The index of the message in the source's reads. */
  read: number;
}

/** A field of a message, in the tree under its message, or under its source. */
export interface FieldNode {
  kind: "field";
  /** The key of the source. */
  source: string;
  /** The index of the message in the source's reads. */
  read: number;
  field: TreeField;
}

export type Node = SourceNode | MessageNode | FieldNode;

export const messageKey = (source: string, message: icd.MessageKey): string =>
  `${source}/${message}`;

export const fieldKey = (
  source: string,
  message: icd.MessageKey,
  field: string,
): string => `${source}/${message}/${field}`;

/** A source as the tree opens it. A source that reads no messages has none. */
export interface InitialSource extends record.Keyed<string> {
  messages?: icd.ReadMessage[];
}

/**
 * @returns the expanded state of a task's tree as it opens: every source, and every
 * message of a source that reads several.
 */
export const initialExpanded = (sources: InitialSource[]): string[] =>
  sources.flatMap(({ key, messages = [] }) =>
    messages.length > 1
      ? [key, ...messages.map(({ message }) => messageKey(key, message))]
      : [key],
  );
