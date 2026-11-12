// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/vcs/History.css";

import { type vcs } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Status } from "@synnaxlabs/lyra/status";
import { Telem } from "@synnaxlabs/lyra/telem";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Synnax } from "@synnaxlabs/pluto";
import { type ReactElement, useEffect, useState } from "react";

import {
  Empty,
  ItemList,
  Loading,
  ResourceItem,
  SectionHeader,
} from "@/feature/vcs/common";
import { Diff } from "@/feature/vcs/diff";
import { CSS } from "@/platform/css";
import { Vcs } from "@/platform/vcs";

const LOG_LIMIT = 200;

/**
 * The rev before a commit. History is linear, so the next older commit in the log is
 * the parent, or the last commit before it that touched the filtered resource.
 */
const parentRev = (commit: vcs.Commit, older: vcs.Commit | undefined): vcs.Rev =>
  older?.hash ?? `${commit.hash}^`;

interface CommitItemProps {
  commit: vcs.Commit;
  parent: vcs.Rev;
  parentLabel: string;
  expanded: boolean;
  onToggle: () => void;
  filter: Vcs.HistoryFilter | null;
  canUpdate: boolean;
}

const CommitItem = ({
  commit,
  parent,
  parentLabel,
  expanded,
  onToggle,
  filter,
  canUpdate,
}: CommitItemProps): ReactElement => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const openDiff = Diff.useOpen();
  const shows = Diff.useShows();
  const openPlan = Vcs.useOpenPlan();
  const showHistory = Vcs.useShowHistory();
  const [changes, setChanges] = useState<vcs.Change[] | null>(commit.changes ?? null);
  const { hash, shortHash: short, message, author, time } = commit;
  useEffect(() => {
    if (!expanded || changes != null || client == null) return;
    handleError(async () => {
      const shown = await client.vcs.show(hash);
      setChanges(shown.changes ?? []);
    }, "Failed to load commit");
  }, [expanded, client, hash]);
  const restore = (resource?: { resource: string; name: string }) =>
    openPlan({
      target: {
        kind: "restore",
        commit: hash,
        resources: resource == null ? undefined : [resource.resource],
      },
      title:
        resource == null ? `Restore ${short}` : `Restore ${resource.name} to ${short}`,
      after: hash,
      afterLabel: short,
      message: `Restore ${short}: ${message}`,
    });
  const shown =
    filter == null ? changes : changes?.filter((c) => c.resource === filter.resource);
  const Caret = expanded ? Icon.Caret.Down : Icon.Caret.Right;
  return (
    <Flex.Box y empty className={CSS.B("vcs-commit")}>
      <List.Item
        itemKey={hash}
        y
        gap="tiny"
        onClick={onToggle}
        className={CSS.cls(CSS.BE("vcs", "item"), CSS.BE("vcs-commit", "item"))}
      >
        <Flex.Box x align="center" gap="small" className={CSS.BE("vcs", "item-line")}>
          <Caret className={CSS.BE("vcs-commit", "caret")} />
          <Text.Text weight={450} overflow="fade" className={CSS.BE("vcs", "name")}>
            {message}
          </Text.Text>
          <Flex.Box x align="center" gap="tiny" className={CSS.BE("vcs", "item-end")}>
            {canUpdate && (
              <Button.Button
                reveal
                variant="text"
                size="tiny"
                textColor={9}
                tooltip={
                  filter == null
                    ? "Restore this version"
                    : `Restore this version of ${filter.name}`
                }
                aria-label={`Restore ${short}`}
                onClick={(e) => {
                  e.stopPropagation();
                  restore(filter ?? undefined);
                }}
              >
                <Icon.Reset />
              </Button.Button>
            )}
          </Flex.Box>
        </Flex.Box>
        <Flex.Box
          x
          align="center"
          justify="between"
          gap="small"
          className={CSS.BE("vcs", "item-below")}
        >
          <Text.Text level="small" color={8} gap="tiny" overflow="fade">
            {author}
            <span aria-hidden>·</span>
            <Telem.Text.TimeSpanSince
              level="small"
              format="semantic"
              color={8}
              el="span"
            >
              {time}
            </Telem.Text.TimeSpanSince>
          </Text.Text>
          <Text.Text level="tiny" variant="code" color={8}>
            {short}
          </Text.Text>
        </Flex.Box>
      </List.Item>
      {expanded &&
        (shown == null ? (
          <Loading className={CSS.BE("vcs-commit", "loading")} />
        ) : (
          shown.map((change) => (
            <ResourceItem
              key={change.resource}
              itemKey={`${hash}:${change.resource}`}
              type={change.type}
              name={change.name}
              className={CSS.BE("vcs-commit", "resource")}
              selected={shows(change.resource, parent, hash)}
              onClick={() =>
                openDiff({
                  resource: change.resource,
                  type: change.type,
                  name: change.name,
                  before: parent,
                  beforeLabel: parentLabel,
                  after: hash,
                  afterLabel: short,
                })
              }
              end={
                <>
                  {filter == null && (
                    <Button.Button
                      reveal
                      variant="text"
                      size="tiny"
                      textColor={9}
                      tooltip="Show history"
                      aria-label={`Show history of ${change.name}`}
                      onClick={(e) => {
                        e.stopPropagation();
                        showHistory({ resource: change.resource, name: change.name });
                      }}
                    >
                      <Icon.History />
                    </Button.Button>
                  )}
                  {canUpdate && (
                    <Button.Button
                      reveal
                      variant="text"
                      size="tiny"
                      textColor={9}
                      tooltip="Restore this version"
                      aria-label={`Restore ${change.name} to ${short}`}
                      onClick={(e) => {
                        e.stopPropagation();
                        restore(change);
                      }}
                    >
                      <Icon.Reset />
                    </Button.Button>
                  )}
                  <Vcs.KindMarker kind={change.kind} />
                </>
              }
            />
          ))
        ))}
    </Flex.Box>
  );
};

export interface HistoryProps {
  status: vcs.Status;
}

/** The commits of the current configuration, newest first. */
export const History = ({ status }: HistoryProps): ReactElement => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const { history: filter } = Vcs.useUI();
  const update = Vcs.useUpdateUI();
  const canUpdate = Access.useUpdateGranted(Vcs.RESOURCE_ID);
  const [commits, setCommits] = useState<vcs.Commit[] | null>(null);
  const [expanded, setExpanded] = useState<string | null>(null);
  const head = status.head?.hash;
  const resource = filter?.resource;
  useEffect(() => {
    if (client == null) return;
    let cancelled = false;
    handleError(async () => {
      const next = await client.vcs.log({ resource, limit: LOG_LIMIT });
      if (!cancelled) setCommits(next);
    }, "Failed to load history");
    return () => {
      cancelled = true;
    };
  }, [client, resource, head, status.configuration]);
  const clear = () => update((prev) => ({ ...prev, history: null }));
  let content: ReactElement;
  if (commits == null) content = <Loading />;
  else if (commits.length === 0)
    content = (
      <Empty
        message="No commits"
        description={
          filter == null ? undefined : `No commit has changed ${filter.name} yet.`
        }
      />
    );
  else
    content = (
      <ItemList>
        {commits.map((commit, i) => {
          const older = commits[i + 1];
          return (
            <CommitItem
              key={commit.hash}
              commit={commit}
              parent={parentRev(commit, older)}
              parentLabel={older?.shortHash ?? "Before"}
              expanded={expanded === commit.hash}
              onToggle={() =>
                setExpanded((prev) => (prev === commit.hash ? null : commit.hash))
              }
              filter={filter}
              canUpdate={canUpdate}
            />
          );
        })}
      </ItemList>
    );
  return (
    <Flex.Box y full empty className={CSS.B("vcs-history")}>
      {filter != null && (
        <SectionHeader title={`History of ${filter.name}`}>
          <Button.Button
            variant="text"
            size="tiny"
            level="small"
            textColor={9}
            onClick={clear}
          >
            Show all
          </Button.Button>
        </SectionHeader>
      )}
      {content}
    </Flex.Box>
  );
};
