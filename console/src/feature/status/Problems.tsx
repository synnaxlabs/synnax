// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { List } from "@synnaxlabs/lyra/list";
import { Status } from "@synnaxlabs/lyra/status";
import { Telem } from "@synnaxlabs/lyra/telem";
import { Text } from "@synnaxlabs/lyra/text";
import { TimeStamp } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { Nav } from "@/platform/nav";
import { Notifications } from "@/platform/notifications";

interface Entry {
  key: status.Key;
  problem: Notifications.Problem;
  /** Whether the entry shows under the problem that covers it. */
  nested: boolean;
}

const startOf = ({ status: { since, time } }: Notifications.Problem): TimeStamp =>
  since ?? time;

const newestFirst = (a: Notifications.Problem, b: Notifications.Problem): number =>
  TimeStamp.sort(startOf(b), startOf(a));

const entry = (problem: Notifications.Problem, nested: boolean): Entry => ({
  key: problem.status.key,
  problem,
  nested,
});

// Only problems that no listed problem covers take children. A problem whose covering
// problem nests shows at the top.
const arrange = (problems: Notifications.Problem[]): Entry[] => {
  const sorted = [...problems].sort(newestFirst);
  const listed = new Set(sorted.map(({ status: { key } }) => key));
  const nests = ({ coveredBy }: Notifications.Problem): boolean =>
    coveredBy != null && listed.has(coveredBy);
  const parents = new Set(sorted.filter((p) => !nests(p)).map((p) => p.status.key));
  const children = Map.groupBy(
    sorted.filter((p) => p.coveredBy != null && parents.has(p.coveredBy)),
    ({ coveredBy }) => coveredBy,
  );
  return sorted
    .filter(({ coveredBy }) => coveredBy == null || !parents.has(coveredBy))
    .flatMap((p) => [
      entry(p, false),
      ...(children.get(p.status.key) ?? []).map((c) => entry(c, true)),
    ]);
};

const Item = ({
  itemKey,
  ...rest
}: List.ItemProps<status.Key>): ReactElement | null => {
  const item = List.useItem<status.Key, Entry>(itemKey);
  if (item == null) return null;
  const { problem, nested } = item;
  const { status: stat, count } = problem;
  const { name, variant } = stat;
  const named = name.length > 0;
  const message = named ? stat.message : "";
  return (
    <List.Item
      itemKey={itemKey}
      className={CSS.cls(CSS.B("status-problem"), nested && CSS.M("nested"))}
      gap="small"
      y
      {...rest}
    >
      <Flex.Box x justify="between" align="center">
        <Flex.Box x align="center" gap="small">
          <Status.Indicator variant={variant} />
          <Text.Text level="p" weight={450} status={variant}>
            {named ? name : stat.message}
          </Text.Text>
        </Flex.Box>
        <Telem.Text.TimeSpanSince
          level="small"
          format="semantic"
          variant="code"
          color={9}
        >
          {startOf(problem)}
        </Telem.Text.TimeSpanSince>
      </Flex.Box>
      {(message.length > 0 || count > 1) && (
        <Flex.Box x justify="between" align="center" gap="small">
          <Text.Text level="small" color={9}>
            {message}
          </Text.Text>
          {count > 1 && <Text.Text level="small" color={9}>{`x${count}`}</Text.Text>}
        </Flex.Box>
      )}
    </List.Item>
  );
};

const item = Component.renderProp(Item);

/**
 * Lists the open problems, newest first, with how long each has been open and its
 * count. A problem shows under the open problem that covers it, such as its rack's.
 */
export const Problems = (): ReactElement => {
  const problems = Notifications.useProblems();
  const { data, getItem } = List.useStaticData<status.Key, Entry>({
    data: arrange(problems),
  });
  return (
    <List.Frame<status.Key, Entry> data={data} getItem={getItem}>
      <List.Scroll full="y">
        <List.Items<status.Key> emptyContent={<Empty.Action message="No problems" />}>
          {item}
        </List.Items>
      </List.Scroll>
    </List.Frame>
  );
};

/** Counts the open problems, in the color of the worst one. */
export const Badge = (): ReactElement | null => {
  const problems = Notifications.useProblems();
  if (problems.length === 0) return null;
  const failed = problems.some(({ status: { variant } }) => variant === "error");
  return <Nav.Badge count={problems.length} variant={failed ? "error" : "warning"} />;
};
