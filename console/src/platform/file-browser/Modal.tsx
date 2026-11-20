// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/file-browser/Modal.css";

import { type rack, TimeSpan, type TimeStamp } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { useNow } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
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
import { useDispatch } from "react-redux";

import { CSS } from "@/platform/css";
import {
  COLUMNS,
  Entries,
  Head,
  isShortcut,
  type KeyedEntry,
  pickable,
  type Sort,
  sortEntries,
  type TypeFilter,
} from "@/platform/file-browser/Entries";
import type { Entry, Listing } from "@/platform/file-browser/list";
import {
  basenameOf,
  dirnameOf,
  expandHome,
  hidden,
  normalize,
  withSeparator,
} from "@/platform/file-browser/path";
import { PathBar } from "@/platform/file-browser/PathBar";
import { Places, placesOf } from "@/platform/file-browser/Places";
import { useSession } from "@/platform/file-browser/Provider";
import type { Session, Snapshot } from "@/platform/file-browser/session";
import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";
import { FileBrowser } from "@/session/file-browser";

const nextMinute = (now: TimeStamp): TimeStamp => now.add(TimeSpan.MINUTE);

const NO_SNAPSHOT: Snapshot = {
  listings: new Map(),
  failures: new Map(),
  loading: new Set(),
};

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
   * A file whose folder the modal opens first, with the file selected. Without it, or
   * when it is not absolute, the folder of the last file picked on the rack opens, or
   * else the home folder.
   */
  path?: string;
}

const HIDDEN_FILES_TRIGGER: PTriggers.Trigger = ["Control", "Shift", "Period"];

const Content = ({
  close,
  rack,
  path: start = "",
}: Modals.ContentProps<ModalParams, Entry>): ReactElement => {
  const session = useSession(rack);
  const snapshot = useSnapshot(session);
  const now = useNow(nextMinute);
  const rackResult = Rack.useResult({ key: rack });
  const dispatch = useDispatch();
  const recent = FileBrowser.useSelectRecent(rack);
  const startFile = normalize(start.trim());
  const [history, setHistory] = useState<History>(() => ({
    stack: [startFile === "" ? (recent[0] ?? "") : dirnameOf(startFile)],
    index: 0,
  }));
  const [selected, setSelected] = useState<string | null>(() =>
    startFile === "" ? null : startFile,
  );
  const [hiddenShown, setHiddenShown] = useState(() => hidden(basenameOf(startFile)));
  // Each navigation clears the filter, so it never hides the entry that opens selected.
  const [query, setQuery] = useState("");
  const [type, setType] = useState<TypeFilter>("certificates");
  const [sort, setSort] = useState<Sort>({ key: "name", ascending: true });
  const [shownPath, setShownPath] = useState<string | null>(null);
  // The text the path field opened with, or null while the bar shows the path.
  const [editing, setEditing] = useState<string | null>(null);
  const holderRef = useRef<HTMLDivElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const filterRef = useRef<HTMLInputElement>(null);
  // Each navigation counts, so a typed path that resolves late does not undo a later
  // one.
  const navigations = useRef(0);
  const [resolving, setResolving] = useState(false);
  const headRef = useRef<HTMLDivElement>(null);
  // The headings follow the entries when a narrow dialog scrolls them sideways.
  const alignHead = (left: number): void => {
    if (headRef.current != null) headRef.current.scrollLeft = left;
  };

  const path = history.stack[history.index];
  const places = placesOf(snapshot.listings.get(""));
  const home = places.find((p) => p.kind === "home")?.path;
  if (path === "" && home != null)
    setHistory((h) => ({
      stack: h.stack.map((p, i) => (i === h.index ? home : p)),
      index: h.index,
    }));

  // Each open lists the places again, so one failed list does not leave them empty. The
  // snapshot keeps the failure.
  useEffect(() => {
    session?.list("").catch(() => {});
  }, [session]);

  // Each visit lists the folder again, and the places while they have no listing. The
  // snapshot shows the result or the failure. A start folder from Recent that fails
  // gives way to the home folder.
  const relist = (): void => {
    if (!snapshot.listings.has("")) session?.list("").catch(() => {});
    session?.list(path).catch(() => {
      if (startFile !== "" || path === "" || path === home) return;
      setHistory((h) =>
        h.stack.length === 1 && h.stack[0] === path ? { stack: [""], index: 0 } : h,
      );
    });
  };
  useEffect(relist, [session, path]);

  const listing = snapshot.listings.get(path);
  // The home folder replaces the top level, so the top level never shows as stale.
  if (listing != null && shownPath !== path && (path !== "" || home == null))
    setShownPath(path);
  // A folder that is still loading shows the folder before it, dimmed.
  const shown =
    listing ?? (shownPath == null ? undefined : snapshot.listings.get(shownPath));
  const stale = listing == null;
  // The Driver sends no details in the secrets folder, so each file there passes.
  const typeFilter: TypeFilter = shown?.secret === true ? "all" : type;
  const loading = snapshot.loading.has(path);
  const failure = snapshot.failures.get(path);
  // While a folder loads or fails, the dimmed last folder offers no file to pick.
  const current = !stale && failure == null;
  const visible = shown?.entries.filter((e) => hiddenShown || !hidden(e.name)) ?? [];
  const folded = query.toLowerCase();
  const matched = visible.filter((e) => e.name.toLowerCase().includes(folded));
  const entries: KeyedEntry[] = sortEntries(matched, sort).map((e) => ({
    ...e,
    key: e.path,
  }));
  const { data, getItem } = List.useStaticData<string, KeyedEntry>({ data: entries });

  // A hidden entry to select shows the hidden entries, so it is in the list.
  const select = (key: string | null): void => {
    setSelected(key);
    if (key != null && hidden(basenameOf(key))) setHiddenShown(true);
  };
  // A folder opens with the entry selected, such as the folder the user came up from.
  const countNavigation = (): void => {
    navigations.current++;
    setResolving(false);
    setQuery("");
  };
  const go = (to: string, entry: string | null = null): void => {
    countNavigation();
    if (to === path) {
      if (entry != null) select(entry);
      return;
    }
    select(entry);
    setHistory((h) => ({
      stack: [...h.stack.slice(0, h.index + 1), to],
      index: h.index + 1,
    }));
  };
  const back = (): void => {
    if (history.index === 0) return;
    countNavigation();
    // The folder left is selected only in its parent.
    select(dirnameOf(path) === history.stack[history.index - 1] ? path : null);
    setHistory((h) => ({ ...h, index: h.index - 1 }));
  };
  const forward = (): void => {
    if (history.index === history.stack.length - 1) return;
    countNavigation();
    setSelected(null);
    setHistory((h) => ({ ...h, index: h.index + 1 }));
  };
  const parent = dirnameOf(path);
  const canGoUp = parent !== "" && parent !== path;
  const up = (): void => {
    if (canGoUp) go(parent, path);
  };
  const pick = (entry: Entry): void => {
    dispatch(FileBrowser.addRecent({ rack, folder: dirnameOf(entry.path) }));
    close(entry);
  };
  const open = (entry: Entry): void => {
    if (entry.kind === "directory") go(entry.path);
    else if (current && pickable(entry, typeFilter)) pick(entry);
  };
  const listAhead = (folder: string): void => session?.prefetch(folder);
  // A folder the user points at lists in the background, so it often opens at once.
  const prefetch = (entry: Entry): void => {
    if (entry.kind === "directory") listAhead(entry.path);
  };
  const handleSelect = (key: string | null): void => {
    setSelected(key);
    const entry = key == null ? undefined : getItem(key);
    if (entry != null) prefetch(entry);
  };
  const selectedEntry = selected == null ? undefined : getItem(selected);
  const picked =
    current && selectedEntry != null && pickable(selectedEntry, typeFilter)
      ? selectedEntry
      : undefined;

  const focusList = (): void => (listRef.current ?? holderRef.current)?.focus();
  const cancelEditing = (): void => {
    setEditing(null);
    focusList();
  };
  // A failed list here only leaves the typed path as it is, and the folder it opens
  // then shows the failure.
  const listingOf = async (folder: string): Promise<Listing | undefined> =>
    snapshot.listings.get(folder) ??
    (await session?.list(folder).catch(() => undefined));
  // A typed file opens its folder with the file selected. The top listing names the
  // home folder for a ~, and the folder's listing tells a file from a folder.
  const goTyped = async (text: string): Promise<void> => {
    cancelEditing();
    let typed = text.trim();
    if (typed === "") return;
    const navigation = ++navigations.current;
    setResolving(true);
    if (typed.startsWith("~"))
      typed = expandHome(
        typed,
        placesOf(await listingOf("")).find((p) => p.kind === "home")?.path,
      );
    // The Driver says why it cannot list a path that is not absolute.
    const target = normalize(typed) || typed;
    const folder = dirnameOf(target);
    // Windows and macOS disks often ignore case, so one match in another case counts.
    const find = (listing?: Listing): Entry | undefined => {
      const entries = listing?.entries ?? [];
      const exact = entries.find((e) => e.path === target);
      if (exact != null) return exact;
      const lower = target.toLowerCase();
      const loose = entries.filter((e) => e.path.toLowerCase() === lower);
      return loose.length === 1 ? loose[0] : undefined;
    };
    // A cached listing can be old, so a target it does not hold lists the folder again.
    const entry =
      find(snapshot.listings.get(folder)) ??
      find(await session?.list(folder).catch(() => undefined));
    if (navigations.current !== navigation) return;
    if (entry?.kind === "file") go(folder, entry.path);
    else go(entry?.path ?? target);
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLElement>): void => {
    // The path field keeps the keys it edits with.
    if (editing != null) return;
    const mod = e.metaKey || e.ctrlKey;
    const key = e.key.toLowerCase();
    // The filter keeps the arrow keys that move its caret.
    const inInput = e.target instanceof HTMLInputElement;
    if (e.key === "Escape" && query !== "") {
      // The filter clears first, so the dialog stays open.
      e.preventDefault();
      e.stopPropagation();
      setQuery("");
    } else if ((e.ctrlKey && key === "l") || (mod && e.shiftKey && key === "g")) {
      e.preventDefault();
      setEditing(withSeparator(path));
    } else if (
      (!inInput && e.altKey && e.key === "ArrowLeft") ||
      (e.metaKey && e.key === "[")
    ) {
      e.preventDefault();
      back();
    } else if (
      (!inInput && e.altKey && e.key === "ArrowRight") ||
      (e.metaKey && e.key === "]")
    ) {
      e.preventDefault();
      forward();
    } else if (!inInput && (e.altKey || e.metaKey) && e.key === "ArrowUp") {
      e.preventDefault();
      up();
    } else if (mod && key === "f") {
      e.preventDefault();
      filterRef.current?.focus();
    } else if (mod && e.shiftKey && e.code === "Period") {
      e.preventDefault();
      setHiddenShown((shown) => !shown);
    }
  };
  const handleFilterKeyDown = (e: KeyboardEvent<HTMLInputElement>): void => {
    // The filter keeps the focus while Escape clears it.
    if (e.key === "Escape" && query !== "") e.preventDefault();
    else if (e.key === "ArrowDown") {
      e.preventDefault();
      focusList();
    }
  };

  // A / or ~ typed in the list, or in a state in its place, opens the path field.
  const handlePathKey = (e: KeyboardEvent<HTMLElement>): void => {
    if ((e.key !== "/" && e.key !== "~") || isShortcut(e)) return;
    e.preventDefault();
    e.stopPropagation();
    setEditing(e.key);
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
        <Flex.Box x gap="small">
          <Button.Button variant="text" size="small" textColor={9} onClick={relist}>
            Try again
          </Button.Button>
          {history.index > 0 && (
            <Button.Button variant="text" size="small" textColor={9} onClick={back}>
              Go back
            </Button.Button>
          )}
        </Flex.Box>
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
  else if (matched.length === 0 && query !== "")
    body = (
      <State>
        <Text.Text level="small" color={8}>
          {`No files match "${query}"`}
        </Text.Text>
        <Button.Button
          variant="text"
          size="small"
          textColor={9}
          onClick={() => {
            setQuery("");
            filterRef.current?.focus();
          }}
        >
          Clear filter
        </Button.Button>
      </State>
    );
  else if (!stale && visible.length === 0)
    body = (
      <State>
        <Text.Text level="small" color={8}>
          This folder has only hidden files
        </Text.Text>
        <Button.Button
          variant="text"
          size="small"
          textColor={9}
          onClick={() => setHiddenShown(true)}
        >
          Show hidden files
        </Button.Button>
      </State>
    );
  else
    body = (
      <Entries
        folder={shownPath}
        selected={selected}
        ref={listRef}
        now={now}
        type={typeFilter}
        onSelect={handleSelect}
        onOpen={open}
        onPoint={prefetch}
        onUp={up}
        onScrollX={alignHead}
      />
    );

  let summary = "";
  if (shown != null && !stale && failure == null) {
    // A cut folder names the count that the Driver listed, hidden entries included.
    const listed = plural(shown.entries.length, "item", "items");
    const count = plural(visible.length, "item", "items");
    const matches = plural(matched.length, "match", "matches");
    if (query === "") summary = shown.truncated ? `Shows the first ${listed}.` : count;
    else if (shown.truncated)
      summary = `${matches}. The filter searched only the first ${listed}.`;
    else summary = `${matched.length.toLocaleString()} of ${count}`;
  }
  const rackName = rackResult.variant === "success" ? rackResult.data.name : null;
  return (
    <Modal.Frame className={CSS.BE("file-browser", "frame")} onKeyDown={handleKeyDown}>
      <Modal.Header icon={<Icon.Folder />}>
        {rackName == null ? "Browse files" : ["Browse files", rackName]}
      </Modal.Header>
      <Flex.Box x grow empty className={CSS.BE("file-browser", "body")}>
        <Places places={places} recent={recent} path={path} onGo={go} />
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
                  aria-label="Back"
                  tooltip={
                    <PTriggers.Text trigger={["Alt", "ArrowLeft"]} level="small">
                      Back
                    </PTriggers.Text>
                  }
                  disabled={history.index === 0}
                  onClick={back}
                >
                  <Icon.Arrow.Left />
                </Button.Button>
                <Button.Button
                  variant="text"
                  size="small"
                  textColor={9}
                  aria-label="Forward"
                  tooltip={
                    <PTriggers.Text trigger={["Alt", "ArrowRight"]} level="small">
                      Forward
                    </PTriggers.Text>
                  }
                  disabled={history.index >= history.stack.length - 1}
                  onClick={forward}
                >
                  <Icon.Arrow.Right />
                </Button.Button>
                <Button.Button
                  variant="text"
                  size="small"
                  textColor={9}
                  aria-label="Up"
                  tooltip={
                    <PTriggers.Text trigger={["Alt", "ArrowUp"]} level="small">
                      Up
                    </PTriggers.Text>
                  }
                  disabled={!canGoUp}
                  onClick={up}
                >
                  <Icon.Arrow.Up />
                </Button.Button>
                <PathBar
                  path={path}
                  home={home}
                  listings={snapshot.listings}
                  editing={editing}
                  onEdit={setEditing}
                  onCancel={cancelEditing}
                  onPrefetch={listAhead}
                  onGo={go}
                  onSubmit={(text) => void goTyped(text)}
                />
                <Input.Text
                  ref={filterRef}
                  value={query}
                  onChange={setQuery}
                  size="small"
                  aria-label="Filter"
                  placeholder="Filter"
                  className={CSS.BE("file-browser", "filter")}
                  startContent={<Icon.Search />}
                  spellCheck={false}
                  autoComplete="off"
                  onKeyDown={handleFilterKeyDown}
                />
                <Button.Button
                  variant="text"
                  size="small"
                  textColor={hiddenShown ? 11 : 9}
                  tooltip={
                    <PTriggers.Text trigger={HIDDEN_FILES_TRIGGER} level="small">
                      {hiddenShown ? "Hide hidden files" : "Show hidden files"}
                    </PTriggers.Text>
                  }
                  aria-label="Hidden files"
                  aria-pressed={hiddenShown}
                  onClick={() => setHiddenShown((shown) => !shown)}
                >
                  {hiddenShown ? <Icon.Visible /> : <Icon.Hidden />}
                </Button.Button>
                {((loading && stale) || resolving) && (
                  <Progress.Progress
                    thin
                    className={CSS.BE("file-browser", "progress")}
                  />
                )}
              </Flex.Box>
              <Head ref={headRef} sort={sort} onSort={setSort} />
              <div
                className={CSS.cls(
                  CSS.BE("file-browser", "list"),
                  stale && loading && CSS.M("stale"),
                )}
                ref={holderRef}
                tabIndex={-1}
                onKeyDownCapture={handlePathKey}
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
              if (picked != null) pick(picked);
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
