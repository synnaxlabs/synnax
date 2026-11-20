// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/file-browser/Modal.css";

import { type rack, status, TimeSpan, type TimeStamp } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { useNow } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Progress } from "@synnaxlabs/lyra/progress";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Triggers as PTriggers } from "@synnaxlabs/lyra/triggers";
import { Rack } from "@synnaxlabs/pluto";
import {
  type KeyboardEvent,
  type ReactElement,
  type ReactNode,
  useEffect,
  useRef,
  useState,
  useSyncExternalStore,
} from "react";

import { CSS } from "@/platform/css";
import {
  COLUMNS,
  Entries,
  Head,
  type KeyedEntry,
  pickable,
  type Sort,
  sortEntries,
  type TypeFilter,
} from "@/platform/file-browser/Entries";
import { type Entry, type Listing } from "@/platform/file-browser/list";
import { dirnameOf } from "@/platform/file-browser/path";
import { PathBar } from "@/platform/file-browser/PathBar";
import { Places, placesOf } from "@/platform/file-browser/Places";
import { useSession } from "@/platform/file-browser/Provider";
import { type Session, type Snapshot } from "@/platform/file-browser/session";
import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";

const nextMinute = (now: TimeStamp): TimeStamp => now.add(TimeSpan.MINUTE);

const NO_SNAPSHOT: Snapshot = { listings: new Map(), loading: new Set() };

const subscribeToNothing = (): (() => void) => () => {};

const getNoSnapshot = (): Snapshot => NO_SNAPSHOT;

const useSnapshot = (session: Session | null): Snapshot =>
  useSyncExternalStore(
    session?.subscribe ?? subscribeToNothing,
    session?.snapshot ?? getNoSnapshot,
  );

interface History {
  stack: string[];
  index: number;
}

const ITEM_HEIGHT = 33;

const plural = (count: number, one: string, many: string): string =>
  `${count.toLocaleString()} ${count === 1 ? one : many}`;

const State = ({ children }: { children: ReactNode }): ReactElement => (
  <Flex.Box y center grow gap="small" className={CSS.BE("file-browser", "state")}>
    {children}
  </Flex.Box>
);

export interface ModalParams {
  /** The rack whose files the modal lists. */
  rack: rack.Key;
  /**
   * A file whose folder the modal opens first, with the file selected. Without it, the
   * home folder opens.
   */
  path?: string;
}

const Content = ({
  close,
  rack,
  path: start = "",
}: Modals.ContentProps<ModalParams, Entry>): ReactElement => {
  const session = useSession(rack);
  const snapshot = useSnapshot(session);
  const now = useNow(nextMinute);
  const rackResult = Rack.useResult({ key: rack });
  const [history, setHistory] = useState<History>(() => ({
    stack: [dirnameOf(start.trim())],
    index: 0,
  }));
  const [failures, setFailures] = useState<ReadonlyMap<string, status.Status>>(
    () => new Map(),
  );
  const [selected, setSelected] = useState<string | null>(null);
  const [type, setType] = useState<TypeFilter>("certificates");
  const [sort, setSort] = useState<Sort>({ key: "name", ascending: true });
  const [lastShown, setLastShown] = useState<Listing | null>(null);
  // The entry to select once the next folder opens: the start file, or the folder the
  // user came up from.
  const selectAfter = useRef<string | null>(start.trim() === "" ? null : start.trim());

  const path = history.stack[history.index];
  const places = placesOf(snapshot.listings.get(""));
  const home = places.find((p) => p.kind === "home")?.path;
  if (path === "" && home != null)
    setHistory((h) => ({
      stack: h.stack.map((p, i) => (i === h.index ? home : p)),
      index: h.index,
    }));

  useEffect(() => session?.prefetch(""), [session]);

  useEffect(() => {
    if (session == null) return;
    session.list(path).then(
      () =>
        setFailures((f) => {
          if (!f.has(path)) return f;
          const next = new Map(f);
          next.delete(path);
          return next;
        }),
      (e: unknown) =>
        setFailures((f) =>
          new Map(f).set(path, status.fromException(e, "Failed to list files")),
        ),
    );
    setSelected(selectAfter.current);
    selectAfter.current = null;
  }, [session, path]);

  const listing = snapshot.listings.get(path);
  if (listing != null && listing !== lastShown) setLastShown(listing);
  // A folder that is still loading shows the folder before it, dimmed.
  const shown = listing ?? lastShown;
  const stale = listing == null;
  const loading = snapshot.loading.has(path);
  const failure = failures.get(path);
  const entries: KeyedEntry[] =
    shown == null
      ? []
      : sortEntries(shown.entries, sort).map((e) => ({ ...e, key: e.path }));
  const { data, getItem } = List.useStaticData<string, KeyedEntry>({ data: entries });

  const go = (to: string, from: string | null = null): void => {
    if (to === path) return;
    selectAfter.current = from;
    setHistory((h) => ({
      stack: [...h.stack.slice(0, h.index + 1), to],
      index: h.index + 1,
    }));
  };
  const back = (): void => {
    if (history.index === 0) return;
    selectAfter.current = path;
    setHistory((h) => ({ ...h, index: h.index - 1 }));
  };
  const forward = (): void => {
    if (history.index < history.stack.length - 1)
      setHistory((h) => ({ ...h, index: h.index + 1 }));
  };
  const parent = dirnameOf(path);
  const canGoUp = parent !== "" && parent !== path;
  const up = (): void => {
    if (canGoUp) go(parent, path);
  };
  const open = (entry: Entry): void => {
    if (entry.kind === "directory") go(entry.path);
    else if (pickable(entry, type)) close(entry);
  };
  // A folder the user points at lists in the background, so it often opens at once.
  const prefetch = (entry: Entry): void => {
    if (entry.kind === "directory") session?.prefetch(entry.path);
  };
  const handleSelect = (key: string | null): void => {
    setSelected(key);
    const entry = key == null ? undefined : getItem(key);
    if (entry != null) prefetch(entry);
  };
  const selectedEntry = selected == null ? undefined : getItem(selected);
  const picked =
    selectedEntry != null && pickable(selectedEntry, type) ? selectedEntry : undefined;

  const handleKeyDown = (e: KeyboardEvent<HTMLElement>): void => {
    if ((e.altKey && e.key === "ArrowLeft") || (e.metaKey && e.key === "[")) {
      e.preventDefault();
      back();
    } else if ((e.altKey && e.key === "ArrowRight") || (e.metaKey && e.key === "]")) {
      e.preventDefault();
      forward();
    } else if ((e.altKey || e.metaKey) && e.key === "ArrowUp") {
      e.preventDefault();
      up();
    }
  };

  let body: ReactElement;
  if (failure != null)
    body = (
      <State>
        <Text.Text level="small" status="error">
          {failure.message}
        </Text.Text>
        <Text.Text level="small" color={10}>
          {failure.description}
        </Text.Text>
        {history.index > 0 && (
          <Button.Button variant="text" size="small" textColor={9} onClick={back}>
            Go back
          </Button.Button>
        )}
      </State>
    );
  else if (shown == null)
    body = (
      <State>
        <Text.Text level="small" color={8}>
          Listing files
        </Text.Text>
      </State>
    );
  else if (!stale && shown.entries.length === 0)
    body = (
      <State>
        <Text.Text level="small" color={8}>
          This folder is empty
        </Text.Text>
      </State>
    );
  else
    body = <Entries now={now} type={type} onOpen={open} onPoint={prefetch} onUp={up} />;

  let summary = "";
  if (shown != null && !stale && failure == null) {
    const count = plural(shown.entries.length, "item", "items");
    summary = shown.truncated ? `Shows the first ${count}.` : count;
  }
  const rackName = rackResult.variant === "success" ? rackResult.data.name : null;
  return (
    <Modal.Frame className={CSS.BE("file-browser", "frame")}>
      <Modal.Header icon={<Icon.Folder />}>
        {rackName == null ? "Browse files" : ["Browse files", rackName]}
      </Modal.Header>
      <Flex.Box
        x
        grow
        empty
        className={CSS.BE("file-browser", "body")}
        onKeyDown={handleKeyDown}
      >
        <Places places={places} path={path} onGo={go} />
        <Select.Frame<string, KeyedEntry>
          data={data}
          getItem={getItem}
          value={selected ?? undefined}
          onChange={handleSelect}
          cursor="selection"
          virtual
          itemHeight={ITEM_HEIGHT}
        >
          <List.Columns columns={COLUMNS}>
            <Flex.Box y grow empty className={CSS.BE("file-browser", "main")}>
              <Flex.Box
                x
                align="center"
                gap="small"
                className={CSS.BE("file-browser", "toolbar")}
              >
                <Button.Button
                  variant="text"
                  size="small"
                  textColor={9}
                  tooltip="Back"
                  disabled={history.index === 0}
                  onClick={back}
                >
                  <Icon.Arrow.Left />
                </Button.Button>
                <Button.Button
                  variant="text"
                  size="small"
                  textColor={9}
                  tooltip="Forward"
                  disabled={history.index >= history.stack.length - 1}
                  onClick={forward}
                >
                  <Icon.Arrow.Right />
                </Button.Button>
                <Button.Button
                  variant="text"
                  size="small"
                  textColor={9}
                  tooltip="Up"
                  disabled={!canGoUp}
                  onClick={up}
                >
                  <Icon.Arrow.Up />
                </Button.Button>
                <PathBar path={path} onGo={go} />
                {loading && stale && (
                  <Progress.Progress
                    thin
                    className={CSS.BE("file-browser", "progress")}
                  />
                )}
              </Flex.Box>
              <Head sort={sort} onSort={setSort} />
              <div
                className={CSS.cls(
                  CSS.BE("file-browser", "list"),
                  stale && loading && CSS.M("stale"),
                )}
                tabIndex={-1}
              >
                {body}
              </div>
              <Flex.Box x align="center" className={CSS.BE("file-browser", "summary")}>
                <Text.Text level="small" color={8}>
                  {summary}
                </Text.Text>
              </Flex.Box>
            </Flex.Box>
          </List.Columns>
        </Select.Frame>
      </Flex.Box>
      <Modal.Footer className={CSS.BE("file-browser", "footer")}>
        <Nav.Bar.Start x align="center" gap="medium">
          <Select.Buttons<TypeFilter> value={type} onChange={setType}>
            <Select.Item itemKey="certificates" size="small">
              <Icon.Certificate />
              Certificates
            </Select.Item>
            <Select.Item itemKey="all" size="small">
              <Icon.File />
              All files
            </Select.Item>
          </Select.Buttons>
        </Nav.Bar.Start>
        <Nav.Bar.End x align="center">
          <Button.Button
            variant="outlined"
            onClick={() => close()}
            triggerIndicator={PTriggers.ESCAPE}
          >
            Cancel
          </Button.Button>
          <Button.Button
            variant="filled"
            disabled={picked == null}
            onClick={() => {
              if (picked != null) close(picked);
            }}
            trigger={Triggers.SAVE}
            triggerIndicator
          >
            Select
          </Button.Button>
        </Nav.Bar.End>
      </Modal.Footer>
    </Modal.Frame>
  );
};

/**
 * Opens a modal that browses the files of a rack. Resolves with the picked file, or
 * null on cancel.
 */
export const useModal = Modals.createPrompt<Entry, ModalParams>(Content);
