// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type label, type ranger } from "@synnaxlabs/client";
import { TimeRange, TimeSpan } from "@synnaxlabs/x";

/** How a labels condition reads the labels of a range. `empty` matches no labels. */
export type LabelMatch = "any" | "all" | "none" | "empty";

/** One chip of the explorer filter. The chips of a filter combine with and. */
export type Condition =
  | { property: "labels"; match: LabelMatch; keys: label.Key[] }
  | { property: "stage"; stages: ranger.Stage[] }
  /** A span matches the ranges that overlap the span that ends now. */
  | { property: "time"; window: TimeSpan | TimeRange }
  | { property: "parent"; keys: ranger.Key[] }
  /** A filter this explorer did not write. It shows as text. */
  | { property: "other"; where: ranger.Where };

export type Property = Condition["property"];

export type ConditionOf<P extends Property> = Extract<Condition, { property: P }>;

const encodeLabels = ({ match, keys }: ConditionOf<"labels">): ranger.Where => {
  switch (match) {
    case "any":
      return { labels: keys };
    case "all":
      return { and: keys.map((key) => ({ labels: [key] })) };
    case "none":
      return { not: { labels: keys } };
    case "empty":
      return { labeled: false };
  }
};

const encodeOne = (condition: Condition): ranger.Where => {
  switch (condition.property) {
    case "labels":
      return encodeLabels(condition);
    case "stage":
      return { stages: condition.stages };
    case "time":
      return condition.window instanceof TimeSpan
        ? { recent: condition.window }
        : { overlaps: condition.window };
    case "parent":
      return { parents: condition.keys };
    case "other":
      return condition.where;
  }
};

/** @returns the filter that holds the conditions, or undefined for none. */
export const encode = (conditions: Condition[]): ranger.Where | undefined =>
  conditions.length === 0 ? undefined : { and: conditions.map(encodeOne) };

const definedFields = (where: ranger.Where): (keyof ranger.Where)[] =>
  (Object.keys(where) as (keyof ranger.Where)[]).filter(
    (field) => where[field] !== undefined,
  );

const soleField = (where: ranger.Where): keyof ranger.Where | null => {
  const fields = definedFields(where);
  return fields.length === 1 ? fields[0] : null;
};

const soleLabel = (where: ranger.Where): label.Key | undefined =>
  soleField(where) != null && where.labels?.length === 1 ? where.labels[0] : undefined;

const decodeOne = (where: ranger.Where): Condition => {
  const other: Condition = { property: "other", where };
  if (soleField(where) == null) return other;
  const { labels, labeled, not, and, stages, recent, overlaps, parents } = where;
  if (labels != null) return { property: "labels", match: "any", keys: labels };
  if (labeled === false) return { property: "labels", match: "empty", keys: [] };
  if (not?.labels != null && soleField(not) != null)
    return { property: "labels", match: "none", keys: not.labels };
  if (and != null && and.length > 0) {
    const keys = and.map(soleLabel);
    if (keys.every((key): key is label.Key => key != null))
      return { property: "labels", match: "all", keys };
  }
  if (stages != null) return { property: "stage", stages };
  if (recent != null) return { property: "time", window: new TimeSpan(recent) };
  if (overlaps != null) return { property: "time", window: new TimeRange(overlaps) };
  if (parents != null) return { property: "parent", keys: parents };
  return other;
};

/** @returns the conditions of a filter, one for each chip. */
export const decode = (where?: ranger.Where): Condition[] => {
  if (where == null || definedFields(where).length === 0) return [];
  if (soleField(where) === "and" && where.and != null) return where.and.map(decodeOne);
  return [decodeOne(where)];
};

/** @returns the condition on the property, if the filter has one. */
export const find = <P extends Property>(
  conditions: Condition[],
  property: P,
): ConditionOf<P> | undefined =>
  conditions.find((c): c is ConditionOf<P> => c.property === property);

/** @returns the conditions with the condition in place of the one on its property. */
export const put = (conditions: Condition[], condition: Condition): Condition[] =>
  conditions.some((c) => c.property === condition.property)
    ? conditions.map((c) => (c.property === condition.property ? condition : c))
    : [...conditions, condition];

/** @returns the conditions without the one on the property. */
export const drop = (conditions: Condition[], property: Property): Condition[] =>
  conditions.filter((c) => c.property !== property);

export interface TimePreset {
  key: string;
  name: string;
  span: TimeSpan;
}

export const TIME_PRESETS: TimePreset[] = [
  { key: "day", name: "Last 24 hours", span: TimeSpan.days(1) },
  { key: "week", name: "Last 7 days", span: TimeSpan.days(7) },
  { key: "month", name: "Last 30 days", span: TimeSpan.days(30) },
  { key: "quarter", name: "Last 90 days", span: TimeSpan.days(90) },
];

/** @returns the preset that matches the span, if any. */
export const findPreset = (span: TimeSpan): TimePreset | undefined =>
  TIME_PRESETS.find((p) => p.span.equals(span));
