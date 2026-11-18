// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/range/toolbar/Content.css";

import { ranger } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Color } from "@synnaxlabs/lyra/color";
import { context } from "@synnaxlabs/lyra/context";
import { Haul } from "@synnaxlabs/lyra/haul";
import { Header } from "@synnaxlabs/lyra/header";
import { useNow } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Text } from "@synnaxlabs/lyra/text";
import { Theming } from "@synnaxlabs/lyra/theming";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { Access, Ranger, Synnax } from "@synnaxlabs/pluto";
import { color, location, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import {
  type CSSProperties,
  Fragment,
  type KeyboardEvent,
  type MouseEvent,
  type ReactElement,
  type ReactNode,
  useEffect,
  useRef,
  useState,
} from "react";

import { Explorer } from "@/feature/range/explorer";
import { Card } from "@/feature/range/toolbar/Card";
import {
  type Forest,
  forest,
  type Groups,
  useGroups,
} from "@/feature/range/toolbar/data";
import { suggest, type Suggestion } from "@/feature/range/toolbar/suggest";
import {
  formatAgo,
  formatSpan,
  formatStart,
  nextMinute,
  nextSecond,
} from "@/feature/range/toolbar/time";
import { Tree } from "@/feature/range/tree";
import { Analytics } from "@/platform/analytics";
import { CSS } from "@/platform/css";
import { Range } from "@/platform/range";
import { Toolbar } from "@/platform/toolbar";
import { Session } from "@/session";

const B = "range-toolbar";

const Elapsed = ({ start }: { start: TimeStamp }): ReactElement => {
  const now = useNow(nextSecond);
  return <>{formatSpan(start, now)}</>;
};

const Ago = ({ end }: { end: TimeStamp }): ReactElement => {
  const now = useNow(nextMinute);
  return <>{formatAgo(end, now)}</>;
};

const startOf = (r: ranger.Range): TimeStamp => r.timeRange.makeValid().start;
const endOf = (r: ranger.Range): TimeStamp => r.timeRange.makeValid().end;

const stopPropagation = (e: MouseEvent) => e.stopPropagation();

const HAUL_SOURCE = "range_toolbar";

const CARD_LOCATION = {
  targetCorner: location.TOP_RIGHT,
  dialogCorner: location.TOP_LEFT,
};

const Parent = ({ parent }: { parent: ranger.Key }): ReactElement | null => {
  const range = Ranger.useResult({ key: parent }).data;
  if (range == null) return null;
  return (
    <Text.Text level="small" color={9} overflow="fade" className={CSS.BE(B, "parent")}>
      {range.name}
    </Text.Text>
  );
};

interface CaretState {
  expanded: boolean;
  onToggle: () => void;
}

interface LineProps {
  range: ranger.Range;
  depth?: number;
  path?: boolean;
  live?: boolean;
  ending?: boolean;
  caret?: CaretState | null;
  meta?: ReactNode;
  actions?: ReactNode;
}

const rangeStyle = (r: ranger.Range, depth: number): CSSProperties =>
  ({
    "--console-range-depth": depth,
    ...(r.color == null ? {} : { "--console-range-color": color.hex(r.color) }),
  }) as CSSProperties;

const Line = ({
  range,
  depth = 0,
  path = false,
  live = false,
  ending = false,
  caret,
  meta,
  actions,
}: LineProps): ReactElement => {
  const open = Range.useOpen();
  const { update: rename } = Ranger.useRename();
  const canEdit = Access.useUpdateGranted(ranger.resourceID(range.key));
  const { startDrag, onDragEnd } = Haul.useDrag({ type: HAUL_SOURCE });
  const [dragging, setDragging] = useState(false);
  return (
    <Tooltip.Dialog location={CARD_LOCATION} hide={dragging}>
      <Card range={range} />
      <div
        id={range.key}
        draggable
        onDragStart={(event) => {
          setDragging(true);
          startDrag({
            event,
            items: [Ranger.createHaulItem(range.payload)],
            preview: <Ranger.HaulPreview range={range.payload} />,
          });
        }}
        onDragEnd={(e) => {
          setDragging(false);
          onDragEnd(e.nativeEvent);
        }}
        role="listitem"
        aria-label={range.name}
        className={CSS.cls(
          CSS.BE(B, "line"),
          Menu.CONTEXT_TARGET,
          "lyra--reveals",
          live && CSS.BEM(B, "line", "live"),
          ending && CSS.BEM(B, "line", "ending"),
        )}
        style={rangeStyle(range, depth)}
        onClick={() => open(range.key)}
      >
        <span className={CSS.BE(B, "indent")} />
        <span className={CSS.BE(B, "gutter")}>
          {caret != null && (
            <Button.Button
              variant="text"
              size="tiny"
              textColor={8}
              className={CSS.cls(
                CSS.BE(B, "caret"),
                caret.expanded && CSS.BEM(B, "caret", "expanded"),
              )}
              aria-label={caret.expanded ? "Collapse" : "Expand"}
              onClick={(e) => {
                e.stopPropagation();
                caret.onToggle();
              }}
            >
              <Icon.Caret.Right />
            </Button.Button>
          )}
        </span>
        <span className={CSS.BE(B, "mark")} />
        <span className={CSS.BE(B, "text")}>
          <Text.MaybeEditable
            id={List.itemNameID(range.key)}
            value={range.name}
            level="p"
            overflow="fade"
            allowDoubleClick={false}
            className={CSS.BE(B, "name")}
            onChange={canEdit ? (name) => rename({ key: range.key, name }) : undefined}
          />
          {path && range.parent != null && <Parent parent={range.parent} />}
        </span>
        {meta != null && <span className={CSS.BE(B, "meta")}>{meta}</span>}
        {actions != null && (
          <span className={CSS.BE(B, "actions")} onClick={stopPropagation}>
            {actions}
          </span>
        )}
      </div>
    </Tooltip.Dialog>
  );
};

const StopButton = ({ range }: { range: ranger.Range }): ReactElement => {
  const stop = Range.useStop();
  return (
    <Button.Button
      variant="text"
      size="tiny"
      textColor={9}
      tooltip="Stop"
      tooltipLocation="left"
      className={CSS.BE(B, "stop")}
      onClick={() => stop(range)}
    >
      <Icon.Stop />
    </Button.Button>
  );
};

interface DraftContextValue {
  draft: DraftState | null;
  onSave: (draft: DraftState, name: string, color: DraftState["color"]) => void;
  onCancel: () => void;
  onNext: (range: ranger.Range, below: ranger.Range[]) => void;
  onStartChild: (range: ranger.Range) => void;
}

const [DraftContext, useDraftContext] = context.create<DraftContextValue>({
  displayName: "Range.DraftContext",
  providerName: "Range.DraftContext",
});

const RunningActions = ({
  range,
  onNext,
}: {
  range: ranger.Range;
  onNext?: () => void;
}): ReactElement => {
  const { onStartChild } = useDraftContext("Range.RunningActions");
  const canCreate = Access.useCreateGranted(ranger.TYPE_RESOURCE_ID);
  const canUpdate = Access.useUpdateGranted(ranger.resourceID(range.key));
  const next = canCreate && canUpdate ? onNext : undefined;
  return (
    <>
      <span className={CSS.BE(B, "more")}>
        {canCreate && (
          <Button.Button
            variant="text"
            size="tiny"
            textColor={9}
            reveal
            tooltip="Start child range"
            tooltipLocation="left"
            onClick={() => onStartChild(range)}
          >
            <Ranger.CreateIcon />
          </Button.Button>
        )}
        {next != null && (
          <Button.Button
            variant="text"
            size="tiny"
            textColor={9}
            tooltip="Stop and start next"
            tooltipLocation="left"
            className={CSS.BE(B, "next")}
            onClick={next}
          >
            <Range.NextIcon />
          </Button.Button>
        )}
      </span>
      {next == null && <span className={CSS.BE(B, "slot")} />}
      {canUpdate ? (
        <StopButton range={range} />
      ) : (
        <span className={CSS.BE(B, "slot")} />
      )}
    </>
  );
};

const Star = ({ range }: { range: ranger.Range }): ReactElement => (
  <Range.FavoriteButton
    range={range}
    reveal
    size="tiny"
    className={CSS.BE(B, "star")}
  />
);

interface DraftState extends Suggestion {
  startedAt: TimeStamp;
  parent?: ranger.Key;
  /** The running range that the new one follows. */
  after?: ranger.Range;
  /** The ranges that stop as the new one starts: `after` and all that run below it. */
  stops?: ranger.Range[];
}

const Draft = ({
  draft,
  depth = 0,
}: {
  draft: DraftState;
  depth?: number;
}): ReactElement => {
  const { onSave, onCancel } = useDraftContext("Range.Draft");
  const [name, setName] = useState(draft.name);
  const [chosen, setChosen] = useState(draft.color);
  const { colors } = Theming.use();
  const inputRef = useRef<HTMLInputElement>(null);
  // The suggested name starts selected, so typing replaces it.
  useEffect(() => inputRef.current?.select(), []);
  const handleKeyDown = (e: KeyboardEvent) => {
    const trimmed = name.trim();
    if (e.key === "Enter" && trimmed !== "") onSave(draft, trimmed, chosen);
    else if (e.key === "Escape") {
      e.stopPropagation();
      onCancel();
    }
  };
  return (
    <div
      className={CSS.cls(
        CSS.BE(B, "line"),
        CSS.BEM(B, "line", "live"),
        CSS.BEM(B, "line", "draft"),
      )}
      style={{ "--console-range-depth": depth } as CSSProperties}
    >
      <span className={CSS.BE(B, "indent")} />
      <span className={CSS.BE(B, "gutter")} />
      <Color.Swatch
        variant="text"
        size="tiny"
        value={chosen}
        fallback={colors.gray.l7}
        onChange={setChosen}
        // The picker hands focus back to the swatch as it closes, so the name takes it
        // back on the next frame, ready for Enter or Esc.
        onVisibleChange={(visible) => {
          if (!visible) requestAnimationFrame(() => inputRef.current?.focus());
        }}
        tooltip="Color"
        aria-label="Color"
        className={CSS.BE(B, "draft-color")}
      />
      <Input.Text
        autoFocus
        ref={inputRef}
        variant="text"
        size="small"
        level="p"
        value={name}
        onChange={setName}
        placeholder="Range name"
        className={CSS.BE(B, "draft-input")}
        onKeyDown={handleKeyDown}
      />
      <span className={CSS.BE(B, "meta")}>
        <Elapsed start={draft.startedAt} />
      </span>
      <span className={CSS.BE(B, "actions")}>
        <Button.Button
          variant="text"
          size="tiny"
          textColor={9}
          tooltip="Cancel"
          tooltipLocation="left"
          onClick={onCancel}
        >
          <Icon.Close />
        </Button.Button>
      </span>
    </div>
  );
};

const Group = ({
  id,
  title,
  count,
  children,
  icon,
}: {
  id: Session.Range.Group;
  title: string;
  count?: number;
  children: ReactNode;
  icon?: ReactNode;
}): ReactElement => {
  const collapsed = Session.Range.useSelectGroupCollapsed(id);
  const dispatch = Session.useDispatch();
  const handleOpenChange = (open: boolean) =>
    dispatch(Session.Range.setGroupCollapsed({ group: id, collapsed: !open }));
  return (
    <section className={CSS.BE(B, "group")} aria-label={title}>
      <Header.Group
        size="medium"
        open={!collapsed}
        onOpenChange={handleOpenChange}
        className={CSS.BE(B, "group-header")}
      >
        <span className={CSS.BE(B, "group-icon")}>{icon}</span>
        {title}
        {count != null && count > 0 && (
          <Text.Text level="tiny" color={9}>
            {count}
          </Text.Text>
        )}
      </Header.Group>
      {!collapsed && <div className={CSS.BE(B, "group-body")}>{children}</div>}
    </section>
  );
};

const runningBelow = (key: ranger.Key, tree: Forest): ranger.Range[] =>
  (tree.children.get(key) ?? []).flatMap((c) => [c, ...runningBelow(c.key, tree)]);

const RunningLines = ({
  nodes,
  tree,
  depth,
}: {
  nodes: ranger.Range[];
  tree: Forest;
  depth: number;
}): ReactElement => {
  const { draft, onNext } = useDraftContext("Range.RunningLines");
  return (
    <>
      {nodes.map((r) => {
        const children = tree.children.get(r.key) ?? [];
        const follows = draft?.after?.key === r.key;
        return (
          <Fragment key={r.key}>
            <Line
              range={r}
              depth={depth}
              path={depth === 0}
              live
              ending={draft?.stops?.some((stop) => stop.key === r.key) ?? false}
              meta={<Elapsed start={startOf(r)} />}
              actions={
                <RunningActions
                  range={r}
                  onNext={
                    r.parent == null
                      ? undefined
                      : () => onNext(r, runningBelow(r.key, tree))
                  }
                />
              }
            />
            <RunningLines nodes={children} tree={tree} depth={depth + 1} />
            {follows && (
              <Draft key={draft.startedAt.toString()} draft={draft} depth={depth} />
            )}
            {draft != null && draft.after == null && draft.parent === r.key && (
              <Draft key={draft.startedAt.toString()} draft={draft} depth={depth + 1} />
            )}
          </Fragment>
        );
      })}
    </>
  );
};

const RECENT_LIMIT = 10;

const RecentLines = ({
  nodes,
  tree,
  depth,
  expanded,
  onToggle,
}: {
  nodes: ranger.Range[];
  tree: Forest;
  depth: number;
  expanded: Set<ranger.Key>;
  onToggle: (key: ranger.Key) => void;
}): ReactElement => (
  <>
    {nodes.map((r) => {
      const kids = tree.children.get(r.key) ?? [];
      const open = expanded.has(r.key);
      return (
        <Fragment key={r.key}>
          <Line
            range={r}
            depth={depth}
            path={depth === 0}
            caret={
              kids.length === 0
                ? null
                : { expanded: open, onToggle: () => onToggle(r.key) }
            }
            meta={<Ago end={endOf(r)} />}
            actions={<Star range={r} />}
          />
          {open && (
            <RecentLines
              nodes={kids}
              tree={tree}
              depth={depth + 1}
              expanded={expanded}
              onToggle={onToggle}
            />
          )}
        </Fragment>
      );
    })}
  </>
);

const useExpanded = (): [Set<ranger.Key>, (key: ranger.Key) => void] => {
  const [expanded, setExpanded] = useState<Set<ranger.Key>>(() => new Set());
  const toggle = (key: ranger.Key) => {
    const next = new Set(expanded);
    if (next.has(key)) next.delete(key);
    else next.add(key);
    setExpanded(next);
  };
  return [expanded, toggle];
};

const AllRanges = (): ReactElement => {
  const openExplorer = Explorer.useOpenTab();
  return (
    <Button.Button
      variant="text"
      size="small"
      textColor={9}
      gap="small"
      className={CSS.BE(B, "all")}
      onClick={() => openExplorer()}
    >
      <Icon.Explore />
      All ranges
    </Button.Button>
  );
};

const FavoriteLine = ({ range }: { range: ranger.Range }): ReactElement => {
  const now = Ranger.useClock(range.timeRange);
  const live = Ranger.getStage(range.timeRange, now) === "in_progress";
  return (
    <Line
      range={range}
      path
      live={live}
      meta={
        live ? (
          <>
            <Icon.InProgress />
            <Elapsed start={startOf(range)} />
          </>
        ) : (
          formatStart(startOf(range), now)
        )
      }
      actions={<Star range={range} />}
    />
  );
};

const FavoriteLines = ({ favorites }: { favorites: ranger.Range[] }): ReactElement => {
  if (favorites.length === 0)
    return (
      <div className={CSS.BE(B, "hint")}>
        <Icon.StarOutlined />
        Star a range to add it here
      </div>
    );
  return (
    <>
      {favorites.map((r) => (
        <FavoriteLine key={r.key} range={r} />
      ))}
    </>
  );
};

const Body = ({ groups }: { groups: Groups }): ReactElement => {
  const { draft, onNext, onStartChild } = useDraftContext("Range.Body");
  const menuProps = Menu.useContextMenu();
  const [expanded, toggle] = useExpanded();
  const running = forest(groups.running);
  const recent = forest(groups.recent);
  const keys = [
    ...groups.running.map((r) => r.key),
    ...groups.favorites.map((r) => r.key),
    ...groups.recent.map((r) => r.key),
  ];
  return (
    <List.Frame data={keys} getItem={groups.getItem} subscribe={groups.subscribe}>
      <Menu.ContextMenu
        menu={(p) => (
          <Tree.ContextMenu
            {...p}
            onStartChild={onStartChild}
            onNext={(r) => onNext(r, runningBelow(r.key, running))}
          />
        )}
        {...menuProps}
      />
      <div className={CSS.BE(B, "frame")} onContextMenu={menuProps.open}>
        <div className={CSS.B(B)} role="list">
          {(running.roots.length > 0 || draft != null) && (
            <Group
              id="running"
              title="Running"
              count={groups.running.length}
              icon={<Icon.InProgress />}
            >
              {draft != null && draft.parent == null && (
                <Draft key={draft.startedAt.toString()} draft={draft} />
              )}
              <RunningLines nodes={running.roots} tree={running} depth={0} />
            </Group>
          )}
          <Group
            id="favorites"
            title="Favorites"
            count={groups.favorites.length}
            icon={<Icon.StarFilled color="goldenrod" />}
          >
            <FavoriteLines favorites={groups.favorites} />
          </Group>
          <Group id="recent" title="Recent" icon={<Icon.Completed />}>
            <RecentLines
              nodes={recent.roots.slice(0, RECENT_LIMIT)}
              tree={recent}
              depth={0}
              expanded={expanded}
              onToggle={toggle}
            />
            <AllRanges />
          </Group>
        </div>
      </div>
    </List.Frame>
  );
};

export const Content = (): ReactElement => {
  const [draft, setDraft] = useState<DraftState | null>(null);
  // Counts draft requests, so a suggestion that answers late opens no draft.
  const latest = useRef(0);
  const client = Synnax.use();
  const dispatch = Session.useDispatch();
  const groups = useGroups();
  const openExplorer = Explorer.useOpenTab();
  const openCreate = Range.useCreateModal();
  const { capture } = Analytics.use();
  const { updateAsync: create } = Ranger.useCreate();
  const { update: setTimeRange } = Ranger.useSetTimeRange();
  const isRunning = (key: ranger.Key) => groups.running.some((r) => r.key === key);
  // A draft shows under the range that it follows or starts in, so it closes when that
  // range stops.
  const anchor = draft?.after?.key ?? draft?.parent;
  if (anchor != null && !isRunning(anchor)) setDraft(null);
  const open = (next: DraftState) => {
    dispatch(Session.Range.setGroupCollapsed({ group: "running", collapsed: false }));
    setDraft(next);
  };
  const openSuggested = (next: Omit<DraftState, "name" | "color">) => {
    const id = ++latest.current;
    if (client == null || next.parent == null) return;
    void suggest(client, next.parent, next.after?.key).then((s) => {
      if (id === latest.current) open({ ...s, ...next });
    });
  };
  const cancel = () => {
    latest.current++;
    setDraft(null);
  };
  const drafts: DraftContextValue = {
    draft,
    onCancel: cancel,
    onSave: (saved, name, c) => {
      cancel();
      // Old ranges stop only once the new one exists, so a failed start leaves no gap.
      void create({
        name,
        color: c,
        parent: saved.parent,
        timeRange: new TimeRange(saved.startedAt, TimeStamp.MAX),
      }).then((created) => {
        if (!created) return;
        capture("range_created", {});
        for (const stop of saved.stops ?? [])
          setTimeRange({
            key: stop.key,
            timeRange: new TimeRange(stop.timeRange.start, saved.startedAt),
          });
      });
    },
    onNext: (range, below) =>
      openSuggested({
        startedAt: TimeStamp.now(),
        parent: range.parent,
        after: range,
        stops: [range, ...below],
      }),
    onStartChild: (range) => {
      // The child starts now, so only a running range takes it in place.
      if (!isRunning(range.key)) return openCreate({ parent: range.key });
      openSuggested({ startedAt: TimeStamp.now(), parent: range.key });
    },
  };
  const startRange = () => {
    latest.current++;
    open({ startedAt: TimeStamp.now(), name: "" });
  };
  const addPast = () => {
    const now = TimeStamp.now();
    openCreate({
      timeRange: { start: now.sub(TimeSpan.HOUR).nanoseconds, end: now.nanoseconds },
    });
  };
  const canCreate = Access.useCreateGranted(ranger.TYPE_RESOURCE_ID);
  const canRetrieve = Access.useRetrieveGranted(ranger.TYPE_RESOURCE_ID);
  return (
    <Toolbar.Content className={CSS.BE(B, "content")}>
      <Toolbar.Header>
        <Toolbar.Title>
          <Icon.Range />
          Ranges
        </Toolbar.Title>
        <Toolbar.Actions>
          {canRetrieve && (
            <Toolbar.Action
              className={CSS.BE(B, "explore")}
              tooltip="Open range explorer"
              onClick={() => openExplorer()}
            >
              <Icon.Explore />
            </Toolbar.Action>
          )}
          {canCreate && (
            <Toolbar.CreateAction tooltip="Create past range" onClick={addPast} />
          )}
          {canCreate && (
            <Toolbar.CreateAction tooltip="Start range" onClick={startRange}>
              <Icon.Play />
            </Toolbar.CreateAction>
          )}
        </Toolbar.Actions>
      </Toolbar.Header>
      <Toolbar.Body>
        <DraftContext value={drafts}>
          <Body groups={groups} />
        </DraftContext>
      </Toolbar.Body>
    </Toolbar.Content>
  );
};
