// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/vcs/Changes.css";

import { vcs } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Component } from "@synnaxlabs/lyra/component";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Status } from "@synnaxlabs/lyra/status";
import { Telem } from "@synnaxlabs/lyra/telem";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Synnax } from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
import { type KeyboardEvent, type ReactElement, useState } from "react";

import { useCommit, useDiscard } from "@/feature/vcs/actions";
import {
  Empty,
  Footer,
  ItemList,
  ResourceItem,
  SectionHeader,
} from "@/feature/vcs/common";
import { Diff } from "@/feature/vcs/diff";
import { ContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { Runtime } from "@/platform/runtime";
import { Triggers } from "@/platform/triggers";
import { Vcs } from "@/platform/vcs";

const INCOMING_PREFIX = "incoming:";

const byName = (a: vcs.Change, b: vcs.Change): number => a.name.localeCompare(b.name);

/** Returns the newest of the incoming commits, which the remote branch points at. */
export const newestCommit = (commits: vcs.Commit[]): vcs.Commit | undefined =>
  commits.reduce<vcs.Commit | undefined>(
    (newest, c) => (newest == null || c.time.after(newest.time) ? c : newest),
    undefined,
  );

const useOpenChangeDiff = (): ((change: vcs.Change) => void) => {
  const open = Diff.useOpen();
  return ({ resource, type, name }) =>
    open({
      resource,
      type,
      name,
      before: vcs.HEAD,
      beforeLabel: "Last commit",
      after: vcs.WORKING,
      afterLabel: "Now",
    });
};

interface ReviewTagProps {
  review: vcs.Review;
}

const ReviewTag = ({ review }: ReviewTagProps): ReactElement => {
  const handleError = Status.useErrorHandler();
  return (
    <Button.Button
      variant="text"
      size="tiny"
      level="tiny"
      textColor={9}
      tooltip="Open the pull request"
      onClick={(e) => {
        e.stopPropagation();
        handleError(
          async () => await Runtime.openExternal(review.url),
          "Failed to open pull request",
        );
      }}
    >
      <Icon.PullRequest />#{review.number}
    </Button.Button>
  );
};

const stopPropagation = (e: { stopPropagation: () => void }) => e.stopPropagation();

interface ChangeItemProps {
  change: vcs.Change;
  selected: boolean;
  checked: boolean;
  canUpdate: boolean;
  onCheck: (checked: boolean) => void;
  onDiscard: () => void;
  onOpen: () => void;
}

const ChangeItem = ({
  change,
  selected,
  checked,
  canUpdate,
  onCheck,
  onDiscard,
  onOpen,
}: ChangeItemProps): ReactElement => (
  <ResourceItem
    itemKey={change.resource}
    selected={selected}
    type={change.type}
    name={change.name}
    onClick={onOpen}
    start={
      canUpdate ? (
        <span className={CSS.BE("vcs", "check")} onClick={stopPropagation}>
          <Input.Checkbox
            value={checked}
            onChange={onCheck}
            aria-label={`Commit ${change.name}`}
          />
        </span>
      ) : undefined
    }
    end={
      <>
        {change.review != null && <ReviewTag review={change.review} />}
        {canUpdate && (
          <Button.Button
            reveal
            variant="text"
            size="tiny"
            textColor={9}
            tooltip="Discard changes"
            aria-label={`Discard changes to ${change.name}`}
            onClick={(e) => {
              e.stopPropagation();
              onDiscard();
            }}
          >
            <Icon.Discard />
          </Button.Button>
        )}
        <Vcs.KindMarker kind={change.kind} />
      </>
    }
  />
);

interface ChangeMenuProps extends Menu.ContextMenuMenuProps {
  changes: vcs.Change[];
  canUpdate: boolean;
}

const ChangeMenu = ({ keys, changes, canUpdate }: ChangeMenuProps): ReactElement => {
  const discard = useDiscard();
  const openDiff = useOpenChangeDiff();
  const showHistory = Vcs.useShowHistory();
  const targets = changes.filter(({ resource }) => keys.includes(resource));
  const single = targets.length === 1 ? targets[0] : null;
  return (
    <ContextMenu.Menu>
      {single != null && (
        <Menu.Item itemKey="diff" onClick={() => openDiff(single)}>
          <Icon.Diff />
          Open changes
        </Menu.Item>
      )}
      {single != null && single.kind !== "added" && (
        <Menu.Item
          itemKey="history"
          onClick={() => showHistory({ resource: single.resource, name: single.name })}
        >
          <Icon.History />
          Show history
        </Menu.Item>
      )}
      {canUpdate && targets.length > 0 && (
        <>
          <Menu.Divider />
          <Menu.Item itemKey="discard" onClick={() => discard(targets)}>
            <Icon.Discard />
            {single != null ? "Discard changes" : `Discard ${targets.length} changes`}
          </Menu.Item>
        </>
      )}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

interface CommitBoxProps {
  status: vcs.Status;
  checked: string[];
}

const CommitBox = ({ status, checked }: CommitBoxProps): ReactElement => {
  const { message } = Vcs.useUI();
  const update = Vcs.useUpdateUI();
  const commit = useCommit();
  const handleError = Status.useErrorHandler();
  const [loading, setLoading] = useState<"commit" | "pullRequest" | null>(null);
  const trimmed = message.trim();
  const disabled = trimmed.length === 0 || checked.length === 0 || loading != null;
  const github =
    URL.canParse(status.remoteUrl) &&
    new URL(status.remoteUrl).hostname === "github.com";
  const run = (pullRequest: boolean) => {
    if (disabled) return;
    handleError(
      async () => {
        setLoading(pullRequest ? "pullRequest" : "commit");
        await commit({ message: trimmed, resources: checked, pullRequest }).finally(
          () => setLoading(null),
        );
      },
      {
        message: pullRequest ? "Failed to open pull request" : "Failed to commit",
        name: status.configuration,
      },
    );
  };
  const handleKeyDown = (e: KeyboardEvent) => {
    if (e.key !== "Enter" || !(e.metaKey || e.ctrlKey)) return;
    e.preventDefault();
    run(false);
  };
  let tooltip = "Commits to the repository on this Core";
  if (status.remoteUrl.length > 0)
    tooltip = `Commits and pushes to ${status.remoteUrl}`;
  return (
    <Footer>
      <Input.Text
        area
        placeholder="Describe the changes"
        aria-label="Commit message"
        value={message}
        onChange={(v) => update((prev) => ({ ...prev, message: v }))}
        onKeyDown={handleKeyDown}
        className={CSS.BE("vcs-changes", "message")}
      />
      <Flex.Box x pack className={CSS.BE("vcs-changes", "commit-actions")}>
        <Button.Button
          variant="filled"
          grow
          justify="center"
          disabled={disabled}
          status={loading === "commit" ? "loading" : undefined}
          onClick={() => run(false)}
          tooltip={tooltip}
          triggerIndicator={Triggers.SAVE}
        >
          <Icon.Commit />
          Commit to {status.configuration}
        </Button.Button>
        {github && (
          <Dialog.Frame location={location.TOP_RIGHT}>
            <Dialog.Trigger
              variant="filled"
              hideCaret
              full={false}
              disabled={disabled}
              status={loading === "pullRequest" ? "loading" : undefined}
              tooltip="More ways to commit"
              aria-label="More ways to commit"
              className={CSS.BE("vcs-changes", "more")}
            >
              <Icon.Caret.Up />
            </Dialog.Trigger>
            <Dialog.Dialog background={1} pack={false} bordered>
              <CommitMenu onPullRequest={() => run(true)} />
            </Dialog.Dialog>
          </Dialog.Frame>
        )}
      </Flex.Box>
    </Footer>
  );
};

interface CommitMenuProps {
  onPullRequest: () => void;
}

const CommitMenu = ({ onPullRequest }: CommitMenuProps): ReactElement => {
  const { close } = Dialog.useContext();
  return (
    <Menu.Menu
      level="small"
      gap="small"
      onChange={() => {
        close();
        onPullRequest();
      }}
    >
      <Menu.Item itemKey="pullRequest">
        <Icon.PullRequest />
        Commit to a new pull request
      </Menu.Item>
    </Menu.Menu>
  );
};

interface NoticesProps {
  status: vcs.Status;
}

const Notices = ({ status }: NoticesProps): ReactElement | null => {
  const client = Synnax.use();
  const setStatus = Vcs.useSetStatus();
  const handleError = Status.useErrorHandler();
  const [loading, setLoading] = useState(false);
  const { waiting, error, remoteUrl } = status;
  const pending = waiting > 0 && remoteUrl.length > 0;
  if (!pending && error.length === 0) return null;
  const retry = () =>
    handleError(async () => {
      if (client == null) return;
      setLoading(true);
      setStatus(await client.vcs.fetch().finally(() => setLoading(false)));
    }, "Failed to sync");
  return (
    <Flex.Box
      x
      align="start"
      justify="between"
      gap="small"
      shrink={false}
      className={CSS.BE("vcs-changes", "notice")}
    >
      <Status.Summary
        variant={error.length > 0 ? "error" : "warning"}
        level="small"
        center={false}
        message={
          error.length > 0
            ? "Failed to sync with the remote"
            : `${Vcs.count(waiting, "commit")} to push`
        }
        description={
          error.length > 0
            ? error
            : "The Core pushes again on each fetch. Commits work offline."
        }
      />
      <Button.Button
        variant="text"
        size="small"
        textColor={9}
        status={loading ? "loading" : undefined}
        onClick={retry}
      >
        Retry
      </Button.Button>
    </Flex.Box>
  );
};

interface IncomingProps {
  status: vcs.Status;
  canUpdate: boolean;
}

const Incoming = ({ status, canUpdate }: IncomingProps): ReactElement | null => {
  const openPlan = Vcs.useOpenPlan();
  const openDiff = Diff.useOpen();
  const shows = Diff.useShows();
  const { incoming, incomingChanges } = status;
  const newest = newestCommit(incoming);
  if (newest == null) return null;
  const pull = () =>
    openPlan({
      target: { kind: "pull" },
      title: `Pull ${Vcs.count(incoming.length, "commit")}`,
      after: newest.hash,
      afterLabel: "Incoming",
    });
  return (
    <Flex.Box y empty shrink={false} className={CSS.BE("vcs-changes", "incoming")}>
      <SectionHeader title="Incoming" detail={incoming.length}>
        {canUpdate && (
          <Button.Button variant="filled" size="small" onClick={pull}>
            <Icon.Pull />
            Pull
          </Button.Button>
        )}
      </SectionHeader>
      <Flex.Box y empty className={CSS.BE("vcs-changes", "incoming-list")}>
        {incoming.map((c) => (
          <Flex.Box
            key={c.hash}
            y
            gap="tiny"
            className={CSS.BE("vcs-changes", "incoming-commit")}
          >
            <Text.Text overflow="fade">{c.message}</Text.Text>
            <Text.Text level="tiny" color={8} gap="tiny" overflow="fade">
              {c.author}
              <span aria-hidden>·</span>
              <Telem.Text.TimeSpanSince
                level="tiny"
                format="semantic"
                color={8}
                el="span"
              >
                {c.time}
              </Telem.Text.TimeSpanSince>
            </Text.Text>
          </Flex.Box>
        ))}
        {[...incomingChanges].sort(byName).map((change) => (
          <ResourceItem
            key={change.resource}
            itemKey={`${INCOMING_PREFIX}${change.resource}`}
            type={change.type}
            name={change.name}
            end={<Vcs.KindMarker kind={change.kind} />}
            selected={shows(change.resource, vcs.HEAD, newest.hash)}
            onClick={() =>
              openDiff({
                resource: change.resource,
                type: change.type,
                name: change.name,
                before: vcs.HEAD,
                beforeLabel: "Last commit",
                after: newest.hash,
                afterLabel: "Incoming",
              })
            }
          />
        ))}
      </Flex.Box>
    </Flex.Box>
  );
};

export interface ChangesProps {
  status: vcs.Status;
}

/** The uncommitted changes, the incoming commits, and the commit box. */
export const Changes = ({ status }: ChangesProps): ReactElement => {
  const { selection } = Vcs.useUI();
  const update = Vcs.useUpdateUI();
  const discard = useDiscard();
  const openDiff = useOpenChangeDiff();
  const shows = Diff.useShows();
  const canUpdate = Access.useUpdateGranted(Vcs.RESOURCE_ID);
  const menuProps = Menu.useContextMenu();
  const changes = [...status.changes].sort(byName);
  const checked = Vcs.checkedResources(selection, changes);
  const allChecked = changes.length > 0 && checked.length === changes.length;
  const contextMenu = Component.renderProp((props: Menu.ContextMenuMenuProps) => (
    <ChangeMenu {...props} changes={changes} canUpdate={canUpdate} />
  ));
  const setChecked = (resource: string, value: boolean) =>
    update((prev) => ({
      ...prev,
      selection: Vcs.setChecked(prev.selection, resource, value),
    }));
  const setAll = (value: boolean) =>
    update((prev) => ({ ...prev, selection: Vcs.setAllChecked(changes, value) }));
  const hasChanges = changes.length > 0;
  return (
    <Flex.Box y full empty className={CSS.B("vcs-changes")}>
      <Notices status={status} />
      <Incoming status={status} canUpdate={canUpdate} />
      {hasChanges ? (
        <>
          <SectionHeader
            title={Vcs.count(changes.length, "change")}
            start={
              canUpdate && (
                <span className={CSS.BE("vcs", "check")}>
                  <Input.Checkbox
                    value={allChecked}
                    indeterminate={checked.length > 0 && !allChecked}
                    onChange={setAll}
                    aria-label="Commit every change"
                  />
                </span>
              )
            }
          >
            {canUpdate && (
              <Button.Button
                variant="text"
                size="tiny"
                level="small"
                textColor={9}
                tooltip="Discard every change"
                onClick={() => discard(changes)}
              >
                Discard all
              </Button.Button>
            )}
          </SectionHeader>
          <Menu.ContextMenu menu={contextMenu} {...menuProps} />
          <ItemList onContextMenu={menuProps.open}>
            {changes.map((change) => (
              <ChangeItem
                key={change.resource}
                change={change}
                selected={shows(change.resource, vcs.HEAD, vcs.WORKING)}
                checked={Vcs.isChecked(selection, change)}
                canUpdate={canUpdate}
                onCheck={(v) => setChecked(change.resource, v)}
                onDiscard={() => discard([change])}
                onOpen={() => openDiff(change)}
              />
            ))}
          </ItemList>
        </>
      ) : (
        <Empty
          message="No changes"
          description="Edits to tasks, Arc programs, schematics, plots, and channels show here."
        />
      )}
      {canUpdate && hasChanges && <CommitBox status={status} checked={checked} />}
    </Flex.Box>
  );
};
