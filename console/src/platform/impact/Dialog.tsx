// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/impact/Dialog.css";

import {
  AccessDeniedError,
  impact,
  isConnectionError,
  PlanChangedError,
  resource,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { Timeout, Unreachable } from "@synnaxlabs/freighter";
import { Badge } from "@synnaxlabs/lyra/badge";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Header } from "@synnaxlabs/lyra/header";
import { useAsyncEffect } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Note } from "@synnaxlabs/lyra/note";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Triggers as PTriggers } from "@synnaxlabs/lyra/triggers";
import { Synnax } from "@synnaxlabs/pluto";
import { compare, type record, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import {
  type ReactElement,
  useEffect,
  useId,
  useLayoutEffect,
  useReducer,
  useRef,
  useState,
} from "react";

import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Tree } from "@/platform/tree";
import { Triggers } from "@/platform/triggers";

/** The inputs of the dialog: the change, and the names of the records it changes. */
export interface DialogParams {
  change: impact.Change;
  names: string[];
}

const ITEM_HEIGHT = 36;
const DISPLAY_ITEMS = 8;
/** Past this many entries, every group but the first starts closed. */
const MANY_ENTRIES = 100;
/** A spinner shows only after this long, so a fast load shows none. */
const SPINNER_DELAY = TimeSpan.milliseconds(150);
/** A spinner that shows stays at least this long, so it never flashes. */
const SPINNER_MIN = TimeSpan.milliseconds(300);

interface GroupSpec {
  effect: impact.Effect;
  title: string;
}

type Picks = Record<string, impact.Choice>;

const recordKey = (e: impact.Entry): string => resource.idToString(e.record);

const PERMANENT = "This action cannot be undone.";

/** The text of the dialog that depends on the kind of change. */
interface ChangeText {
  icon: Icon.ReactElement;
  breadcrumb: string;
  /** noun names the change in a sentence: "the delete". */
  noun: string;
  verb: string;
  message: (names: string[]) => string;
  description: (names: string[], empty: boolean) => string;
  failure: (names: string[]) => string;
  groups: (plural: boolean) => GroupSpec[];
  /** outcome says where the user sees what an unanswered apply did. */
  outcome: string;
}

const CHANNEL_ICON = <Icon.Channel />;
const GROUP_ICON = <Icon.Group />;

const TEXTS: Record<impact.ChangeType, ChangeText> = {
  delete_channels: {
    icon: CHANNEL_ICON,
    breadcrumb: "Channel.Delete",
    noun: "delete",
    verb: "Delete",
    message: (names) =>
      names.length === 1
        ? `Delete channel ${names[0]}`
        : `Delete ${names.length} channels`,
    description: (names, empty) => {
      const one = names.length === 1;
      const text = one
        ? "Deletes the channel and its data."
        : "Deletes the channels and their data.";
      if (!empty) return `${text} ${PERMANENT}`;
      return `${text} Nothing else uses ${one ? "it" : "them"}. ${PERMANENT}`;
    },
    failure: (names) =>
      names.length === 1 ? "Failed to delete channel" : "Failed to delete channels",
    groups: (plural) => [
      { effect: "restrict", title: "Blocks the delete" },
      { effect: "cascade", title: plural ? "Deleted with them" : "Deleted with it" },
      { effect: "detach", title: plural ? "Loses a channel" : "Loses the channel" },
    ],
    outcome: "The Channels list shows the outcome.",
  },
  delete_groups: {
    icon: GROUP_ICON,
    breadcrumb: "Group.Delete",
    noun: "delete",
    verb: "Delete",
    message: (names) =>
      names.length === 1 ? `Delete group ${names[0]}` : `Delete ${names.length} groups`,
    description: (names, empty) => {
      const one = names.length === 1;
      if (empty)
        return `${one ? "Deletes the group." : "Deletes the groups."} ${PERMANENT}`;
      return one
        ? `Deletes the group, with the paths of deleted channels in it. ${PERMANENT}`
        : `Deletes the groups, with the paths of deleted channels in them. ${PERMANENT}`;
    },
    failure: (names) =>
      names.length === 1 ? "Failed to delete group" : "Failed to delete groups",
    groups: () => [{ effect: "cascade", title: "Paths of deleted channels" }],
    outcome: "The tree shows the outcome.",
  },
  ungroup_groups: {
    icon: GROUP_ICON,
    breadcrumb: "Group.Ungroup",
    noun: "ungroup",
    verb: "Ungroup",
    message: (names) =>
      names.length === 1 ? `Ungroup ${names[0]}` : `Ungroup ${names.length} groups`,
    description: (names) =>
      names.length === 1
        ? "Moves what the group holds to its parent, then deletes the group."
        : "Moves what the groups hold to their parent, then deletes the groups.",
    failure: (names) =>
      names.length === 1 ? "Failed to ungroup group" : "Failed to ungroup groups",
    groups: () => [{ effect: "cascade", title: "Paths that drop" }],
    outcome: "The tree shows the outcome.",
  },
};

const blockedOf = (count: number, { noun }: ChangeText): string =>
  count === 1 ? `1 record blocks the ${noun}` : `${count} records block the ${noun}`;

/** What failed: listing the records, or the change itself. */
type Step = "plan" | "apply";

const deniedOf = (step: Step, { noun }: ChangeText): string =>
  step === "plan"
    ? `Your role cannot see a record that this ${noun} names. Ask an admin for ` +
      "access, then try again."
    : `Your role cannot change a record that this ${noun} changes. Ask an admin for ` +
      "access, or cancel.";

const unreachableOf = (step: Step, { verb }: ChangeText): string =>
  step === "plan"
    ? "Check the connection to the Core, then try again."
    : `Check the connection to the Core, then press ${verb} again.`;

/** The next step when the Core may have the change but no answer came back. */
const unansweredOf = ({ noun, outcome }: ChangeText): string =>
  `The Core can still apply the ${noun}. ${outcome}`;

/** Reports whether err leaves the Core holding a request that it can still apply. */
const unanswered = (err: unknown): boolean =>
  Timeout.matches(err) || (err instanceof Unreachable && err.delivered);

/**
 * Returns why err failed step: our sentence with the next step when one helps, then the
 * error's own text and cause chain, as Status.fromException gives them.
 */
const reasonOf = (err: unknown, step: Step, text: ChangeText): string => {
  const { description } = Status.fromException(err, "");
  if (AccessDeniedError.matches(err)) return `${deniedOf(step, text)} ${description}`;
  if (!isConnectionError(err)) return description;
  if (step === "apply" && unanswered(err))
    return `${unansweredOf(text)} ${description}`;
  return `${unreachableOf(step, text)} ${description}`;
};

/** A plan with the records it names, as the dialog shows it. */
interface View {
  plan: impact.Plan;
  resources: Map<string, resource.Resource>;
  /** fresh holds each record that the plan lists and the plan before it did not. */
  fresh: Set<string>;
  /** freshHidden holds each effect whose hidden count grew since the plan before. */
  freshHidden: Set<impact.Effect>;
}

/**
 * Returns the view of plan, with the records that its entries name. A record that
 * previous does not list is fresh.
 */
const createView = async (
  client: Client,
  plan: impact.Plan,
  previous: impact.Plan | null,
): Promise<View> => {
  // A tombstone entry holds a dead key that no record has, so its path names it.
  const ids = plan.entries.filter((e) => e.path == null).map((e) => e.record);
  const found = ids.length === 0 ? [] : await client.resources.retrieve(ids);
  const fresh = new Set<string>();
  const freshHidden = new Set<impact.Effect>();
  if (previous != null) {
    const listed = new Set(previous.entries.map(recordKey));
    for (const e of plan.entries)
      if (!listed.has(recordKey(e))) fresh.add(recordKey(e));
    for (const effect of impact.EFFECTS)
      if (plan.hidden[effect] > previous.hidden[effect]) freshHidden.add(effect);
  }
  return {
    plan,
    resources: new Map(found.map((r) => [resource.idToString(r.id), r])),
    fresh,
    freshHidden,
  };
};

interface Row {
  key: string;
  entry: impact.Entry;
  resource?: resource.Resource;
  fresh: boolean;
}

/** Icons for the record types that no tree item shows, such as a channel's status. */
const TYPE_ICONS: Partial<Record<resource.Type, Icon.FC>> = { status: Icon.Status };

/** Orders rows by name, then by record, so equal names keep one order. */
const ROW_ORDER = compare.chain<Row>(
  compare.by((r) => r.entry.path ?? r.resource?.name ?? r.entry.record.key),
  compare.by((r) => r.key),
);

interface FooterStatus {
  variant: "error" | "loading";
  message: string;
  description?: string;
}

const UPDATING: FooterStatus = { variant: "loading", message: "Updating the records" };

/**
 * Returns whether to show a spinner for a load: from SPINNER_DELAY after loading starts
 * until it stops, and for at least SPINNER_MIN.
 */
const useSpinner = (loading: boolean): boolean => {
  const [shownAt, setShownAt] = useState<TimeStamp | null>(null);
  useEffect(() => {
    if (loading && shownAt == null) {
      const timer = setTimeout(
        () => setShownAt(TimeStamp.now()),
        SPINNER_DELAY.milliseconds,
      );
      return () => clearTimeout(timer);
    }
    if (loading || shownAt == null) return;
    const left = SPINNER_MIN.milliseconds - TimeStamp.since(shownAt).milliseconds;
    const timer = setTimeout(() => setShownAt(null), Math.max(0, left));
    return () => clearTimeout(timer);
  }, [loading, shownAt]);
  return shownAt != null;
};

/** The label of every check that deletes a blocking record too. */
const PICK_LABEL = "Delete too";

interface CheckProps {
  label: string;
  /** nameID is the id of the text that names the record, read before the label. */
  nameID?: string;
  picked: boolean;
  indeterminate?: boolean;
  /** mark says why the record blocks, after a delete that did not pick it. */
  mark?: string;
  disabled: boolean;
  onChange: (picked: boolean) => void;
}

const Check = ({
  label,
  nameID,
  picked,
  indeterminate,
  mark,
  disabled,
  onChange,
}: CheckProps): ReactElement => {
  const id = useId();
  // The check's name and description hold the label and mark, so a screen reader reads
  // each once, on the check.
  return (
    <Flex.Box x align="center" gap="small" className={CSS.BE("impact", "pick")}>
      {mark != null && (
        <Text.Text
          id={`${id}-mark`}
          level="small"
          aria-hidden
          className={CSS.BE("impact", "error")}
        >
          {mark}
        </Text.Text>
      )}
      <Input.Checkbox
        id={id}
        aria-labelledby={nameID == null ? `${id}-label` : `${nameID} ${id}-label`}
        aria-describedby={mark == null ? undefined : `${id}-mark`}
        value={picked}
        indeterminate={indeterminate}
        disabled={disabled}
        onChange={onChange}
      />
      <Text.Text
        el="label"
        id={`${id}-label`}
        htmlFor={id}
        level="small"
        aria-hidden
        className={CSS.cls(CSS.BE("impact", "label"), disabled && CSS.M("disabled"))}
      >
        {label}
      </Text.Text>
    </Flex.Box>
  );
};

interface EntryProps {
  row: Row;
  index: number;
  translate?: number;
  picked: boolean;
  unpicked: boolean;
  disabled: boolean;
  onPick: (keys: string[], picked: boolean) => void;
}

const Entry = ({
  row: { key, entry, resource: res, fresh },
  index,
  translate,
  picked,
  unpicked,
  disabled,
  onPick,
}: EntryProps): ReactElement => {
  const id = useId();
  const Item = Tree.useItems()[entry.record.type] ?? Tree.DefaultItem;
  const TypeIcon = TYPE_ICONS[entry.record.type];
  const { path } = entry;
  const name = path ?? res?.name ?? entry.record.key;
  const treeIcon = res == null ? undefined : Item.icon(res);
  const icon =
    path != null
      ? CHANNEL_ICON
      : (treeIcon ?? (TypeIcon == null ? null : <TypeIcon />));
  return (
    <List.Item<string>
      itemKey={key}
      index={index}
      translate={translate}
      className={CSS.BE("impact", "entry")}
      align="center"
      gap="medium"
    >
      <span className={CSS.BE("impact", "icon")}>{icon}</span>
      <Text.Text
        id={`${id}-name`}
        overflow="fade"
        grow
        color={path == null ? undefined : 9}
      >
        {name}
      </Text.Text>
      {fresh && <Badge.Badge status="warning">New</Badge.Badge>}
      {entry.effect === "restrict" && (
        <Check
          label={PICK_LABEL}
          nameID={`${id}-name`}
          picked={picked}
          mark={unpicked ? "Uses the channel" : undefined}
          disabled={disabled}
          onChange={(v) => onPick([key], v)}
        />
      )}
    </List.Item>
  );
};

/** The list key of the row for a group's records that the user cannot see. */
const HIDDEN_KEY = "hidden";

interface HiddenEntryProps {
  index: number;
  translate?: number;
  count: number;
  /** fresh is true when the count grew since the plan before. */
  fresh: boolean;
  /** pick is the check for the records, absent when the user picks nothing for them. */
  pick?: Omit<CheckProps, "label" | "nameID" | "indeterminate">;
}

/** The row for the records of a group that the user cannot see. */
const HiddenEntry = ({
  index,
  translate,
  count,
  fresh,
  pick,
}: HiddenEntryProps): ReactElement => {
  const id = useId();
  return (
    <List.Item<string>
      itemKey={HIDDEN_KEY}
      index={index}
      translate={translate}
      className={CSS.BE("impact", "entry")}
      align="center"
      gap="medium"
    >
      <span className={CSS.BE("impact", "icon")}>
        <Icon.Hidden />
      </span>
      <Text.Text id={`${id}-name`} overflow="fade" grow>
        {count === 1 ? "1 record you cannot see" : `${count} records you cannot see`}
      </Text.Text>
      {fresh && <Badge.Badge status="warning">New</Badge.Badge>}
      {pick != null && <Check label={PICK_LABEL} nameID={`${id}-name`} {...pick} />}
    </List.Item>
  );
};

interface GroupProps {
  title: string;
  rows: Row[];
  picks: Picks;
  unpicked: Set<string>;
  initiallyOpen: boolean;
  /** pickable is true for the records that block the delete, which the user picks. */
  pickable: boolean;
  onPick: (keys: string[], picked: boolean) => void;
  /** hidden is the number of the group's records that the user cannot see. */
  hidden: number;
  /** hiddenFresh is true when the hidden count grew since the plan before. */
  hiddenFresh: boolean;
  /** hiddenCheck is the check for the hidden records of a pickable group. */
  hiddenCheck: HiddenCheck;
  /** disabled locks every check of the group. */
  disabled: boolean;
}

/** The check for the hidden records that block the delete. */
interface HiddenCheck {
  picked: boolean;
  /** unpicked is true after a delete that the unpicked hidden records blocked. */
  unpicked: boolean;
  onChange: (picked: boolean) => void;
}

const Group = ({
  title,
  rows,
  picks,
  unpicked,
  initiallyOpen,
  pickable,
  onPick,
  hidden,
  hiddenFresh,
  hiddenCheck,
  disabled,
}: GroupProps): ReactElement => {
  const [open, setOpen] = useState(initiallyOpen);
  const byKey = new Map(rows.map((r) => [r.key, r]));
  const items: record.Keyed<string>[] =
    hidden > 0 ? [...rows, { key: HIDDEN_KEY }] : rows;
  const { data, getItem } = List.useStaticData<string, record.Keyed<string>>({
    data: items,
  });
  // The hidden records share one check, so they count as one pick.
  const hiddenPicks = pickable && hidden > 0 ? 1 : 0;
  const pickCount = rows.length + hiddenPicks;
  const pickedCount =
    rows.filter((r) => picks[r.key] === "delete").length +
    (hiddenPicks > 0 && hiddenCheck.picked ? 1 : 0);
  const scrolls = items.length > DISPLAY_ITEMS;
  const scrollsClass = scrolls && CSS.M("scrolls");
  return (
    <section className={CSS.BE("impact", "group")} aria-label={title}>
      <Header.Group size="medium" open={open} onOpenChange={setOpen}>
        {title}
        <Text.Text level="tiny" color={9}>
          {rows.length + hidden}
        </Text.Text>
      </Header.Group>
      {open && pickable && pickCount > 1 && (
        <Flex.Box
          x
          align="center"
          justify="end"
          gap="small"
          className={CSS.cls(CSS.BE("impact", "all"), scrollsClass)}
        >
          <Check
            label="Delete all too"
            picked={pickedCount === pickCount}
            indeterminate={pickedCount > 0 && pickedCount < pickCount}
            disabled={disabled}
            onChange={(v) => {
              onPick(
                rows.map((r) => r.key),
                v,
              );
              if (hiddenPicks > 0) hiddenCheck.onChange(v);
            }}
          />
        </Flex.Box>
      )}
      {open && (
        <List.Frame<string, record.Keyed<string>>
          data={data}
          getItem={getItem}
          itemHeight={ITEM_HEIGHT}
          virtual
        >
          <List.Scroll
            displayItems={DISPLAY_ITEMS}
            className={CSS.BE("impact", "scroll")}
            {...(scrolls && {
              tabIndex: 0,
              role: "region",
              "aria-label": `${title}, ${rows.length + hidden} records`,
            })}
          >
            <List.Items<string>>
              {({ key, index, translate }) =>
                key === HIDDEN_KEY ? (
                  <HiddenEntry
                    key={key}
                    index={index}
                    translate={translate}
                    count={hidden}
                    fresh={hiddenFresh}
                    pick={
                      pickable
                        ? {
                            picked: hiddenCheck.picked,
                            mark: hiddenCheck.unpicked
                              ? hidden === 1
                                ? "Uses the channel"
                                : "Use the channel"
                              : undefined,
                            disabled,
                            onChange: hiddenCheck.onChange,
                          }
                        : undefined
                    }
                  />
                ) : (
                  <Entry
                    key={key}
                    row={byKey.get(key) as Row}
                    index={index}
                    translate={translate}
                    picked={picks[key] === "delete"}
                    unpicked={unpicked.has(key)}
                    disabled={disabled}
                    onPick={onPick}
                  />
                )
              }
            </List.Items>
          </List.Scroll>
        </List.Frame>
      )}
    </section>
  );
};

interface EntriesProps extends Pick<
  GroupProps,
  "picks" | "unpicked" | "onPick" | "hiddenCheck" | "disabled"
> {
  view: View;
  groups: GroupSpec[];
}

const Entries = ({ view, groups, ...rest }: EntriesProps): ReactElement => {
  const many = view.plan.entries.length > MANY_ENTRIES;
  const first = groups.find(
    ({ effect }) =>
      view.plan.hidden[effect] > 0 ||
      view.plan.entries.some((e) => e.effect === effect),
  );
  return (
    <Flex.Box y gap="small" className={CSS.BE("impact", "entries")}>
      {groups.map(({ effect, title }) => {
        const rows = view.plan.entries
          .filter((e) => e.effect === effect)
          .map((entry) => ({
            key: recordKey(entry),
            entry,
            resource: view.resources.get(recordKey(entry)),
            fresh: view.fresh.has(recordKey(entry)),
          }))
          .toSorted(ROW_ORDER);
        const hidden = view.plan.hidden[effect];
        if (rows.length === 0 && hidden === 0) return null;
        const initiallyOpen = !many || effect === first?.effect;
        return (
          <Group
            key={effect}
            title={title}
            rows={rows}
            pickable={effect === "restrict"}
            hidden={hidden}
            hiddenFresh={view.freshHidden.has(effect)}
            initiallyOpen={initiallyOpen}
            {...rest}
          />
        );
      })}
    </Flex.Box>
  );
};

/** Request is the input of a plan. */
interface Request {
  picks: Picks;
  /** hiddenPick is the pick for the hidden records that block the delete. */
  hiddenPick?: impact.Choice;
}

/** Selection is what the user picked, and the plan request that the picks make. */
interface Selection {
  picks: Picks;
  /** hiddenPicked is true when the user checked the hidden records that block. */
  hiddenPicked: boolean;
  /**
   * request is the input of the plan to show. Only a pick, a retry, or a refused token
   * sets it, so a plan that drops picks plans nothing more.
   */
  request: Request;
}

type SelectionAction =
  | { type: "pick"; keys: string[]; picked: boolean }
  | { type: "pickHidden"; picked: boolean }
  /** keep drops each pick whose record the plan does not list. */
  | { type: "keep"; plan: impact.Plan }
  /** request plans again with the current picks. */
  | { type: "request" };

const INITIAL_SELECTION: Selection = {
  picks: {},
  hiddenPicked: false,
  request: { picks: {} },
};

const requestOf = (picks: Picks, hiddenPicked: boolean): Request => ({
  picks,
  hiddenPick: hiddenPicked ? "delete" : undefined,
});

const updateSelection = (s: Selection, action: SelectionAction): Selection => {
  switch (action.type) {
    case "pick": {
      const picks = { ...s.picks };
      for (const key of action.keys)
        if (action.picked) picks[key] = "delete";
        else delete picks[key];
      return { ...s, picks, request: requestOf(picks, s.hiddenPicked) };
    }
    case "pickHidden":
      return {
        ...s,
        hiddenPicked: action.picked,
        request: requestOf(s.picks, action.picked),
      };
    case "keep": {
      const listed = new Set(action.plan.entries.map(recordKey));
      const kept = Object.entries(s.picks).filter(([key]) => listed.has(key));
      // A plan with no hidden blocker drops the hidden pick, so the hidden pick goes
      // out only while a hidden record blocks.
      const hiddenPicked = s.hiddenPicked && action.plan.hidden.restrict > 0;
      if (
        kept.length === Object.keys(s.picks).length &&
        hiddenPicked === s.hiddenPicked
      )
        return s;
      return { ...s, picks: Object.fromEntries(kept), hiddenPicked };
    }
    case "request":
      return { ...s, request: requestOf(s.picks, s.hiddenPicked) };
  }
};

const Dialog = ({
  change,
  names,
  close,
}: Modals.ContentProps<DialogParams, impact.Plan>): ReactElement => {
  const client = Synnax.use();
  const [selection, dispatch] = useReducer(updateSelection, INITIAL_SELECTION);
  const { picks, hiddenPicked, request } = selection;
  // shownPlan is the plan that the user saw when the Core refused a stale token, which
  // the next plan marks its new entries against.
  const shownPlan = useRef<impact.Plan | null>(null);
  const [view, setView] = useState<View | null>(null);
  const [planning, setPlanning] = useState(true);
  /** planError is why the last plan failed. */
  const [planError, setPlanError] = useState<string | null>(null);
  const [changed, setChanged] = useState(false);
  const [unpicked, setUnpicked] = useState<Set<string>>(() => new Set());
  const [applying, setApplying] = useState(false);
  const [footer, setFooter] = useState<FooterStatus | null>(null);
  const [hiddenUnpicked, setHiddenUnpicked] = useState(false);
  const frameRef = useRef<HTMLDivElement>(null);
  // The checks lock while an apply runs, which takes the focus from a focused check.
  // It goes back once the apply ends.
  const refocus = useRef<HTMLElement | null>(null);
  useLayoutEffect(() => {
    if (applying) return;
    refocus.current?.focus();
    refocus.current = null;
  }, [applying]);
  Modals.useBusy(applying);
  const listing = useSpinner(view == null && planError == null);
  // The dialog stays unseen until its first plan arrives or the spinner shows, so a
  // plan inside the spinner delay opens the dialog at its final size. Lyra places a
  // modal by its size, so it shows two frames later, once that place settles. Once
  // shown, it stays shown, also while a retry plans again.
  const ready = view != null || planError != null || listing;
  const [painted, setPainted] = useState(false);
  useEffect(() => {
    if (!ready || painted) return;
    let second = 0;
    const first = requestAnimationFrame(() => {
      second = requestAnimationFrame(() => setPainted(true));
    });
    return () => {
      cancelAnimationFrame(first);
      cancelAnimationFrame(second);
    };
  }, [ready, painted]);
  const updating = useSpinner(planning && view != null);
  const plural = names.length > 1;
  const text = TEXTS[change.type];

  useAsyncEffect(
    async (signal) => {
      if (client == null) return;
      setPlanning(true);
      let next: View | null = null;
      let failure: unknown = null;
      const shown = shownPlan.current;
      try {
        const planned = await client.impact.plan(change, {
          picks: request.picks,
          hidden: request.hiddenPick,
        });
        next = await createView(client, planned, shown);
      } catch (e) {
        failure = e;
      }
      if (signal.aborted) return;
      setPlanning(false);
      if (next == null) {
        setPlanError(reasonOf(failure, "plan", text));
        return;
      }
      shownPlan.current = null;
      const planned = next;
      setPlanError(null);
      dispatch({ type: "keep", plan: planned.plan });
      setView((prev) => {
        if (prev == null || shown != null) return planned;
        return { ...planned, fresh: prev.fresh, freshHidden: prev.freshHidden };
      });
    },
    [client, change, request],
  );

  // The retry button unmounts, so focus moves to the modal's native dialog instead of
  // the page.
  const handleRetry = () => {
    frameRef.current?.closest("dialog")?.focus();
    setPlanning(true);
    setPlanError(null);
    dispatch({ type: "request" });
  };

  // A pick plans again, so the apply waits for that plan.
  const handlePick = (keys: string[], picked: boolean) => {
    setPlanning(true);
    setUnpicked((prev) => {
      const next = new Set(prev);
      for (const key of keys) next.delete(key);
      return next;
    });
    setFooter(null);
    dispatch({ type: "pick", keys, picked });
  };

  const handleHidden = (picked: boolean) => {
    setPlanning(true);
    setHiddenUnpicked(false);
    setFooter(null);
    dispatch({ type: "pickHidden", picked });
  };

  const handleApply = async () => {
    if (client == null || view == null) return;
    const blockers = view.plan.entries
      .filter((e) => e.effect === "restrict" && picks[recordKey(e)] !== "delete")
      .map(recordKey);
    const hiddenBlockers = hiddenPicked ? 0 : view.plan.hidden.restrict;
    const blocking = blockers.length + hiddenBlockers;
    if (blocking > 0) {
      setUnpicked(new Set(blockers));
      setHiddenUnpicked(hiddenBlockers > 0);
      setFooter({
        variant: "error",
        message: blockedOf(blocking, text),
        description:
          blocking === 1
            ? `Check "${PICK_LABEL}", or cancel.`
            : `Check "${PICK_LABEL}" on each one, or cancel.`,
      });
      return;
    }
    refocus.current =
      document.activeElement instanceof HTMLElement ? document.activeElement : null;
    setApplying(true);
    setFooter(null);
    let applied: impact.Plan | null = null;
    let failure: unknown = null;
    try {
      applied = await client.impact.apply(change, {
        token: view.plan.token,
        picks,
        hidden: requestOf(picks, hiddenPicked).hiddenPick,
      });
    } catch (e) {
      failure = e;
    }
    setApplying(false);
    if (applied != null) {
      close(applied);
      return;
    }
    if (!(failure instanceof PlanChangedError)) {
      setFooter({
        variant: "error",
        message: unanswered(failure) ? "No answer from the Core" : text.failure(names),
        description: reasonOf(failure, "apply", text),
      });
      return;
    }
    // The Core refused a stale token. The plan effect loads the current plan once, with
    // its own failure text and Try again, and marks what is new against this plan.
    shownPlan.current = view.plan;
    setChanged(true);
    setPlanning(true);
    dispatch({ type: "keep", plan: impact.planOf(failure) });
    dispatch({ type: "request" });
  };

  // A held spinner hides a failure until it ends, so the two never show together. A
  // failure shows where its spinner showed: the first plan's in the body, in place of
  // the list, and a later plan's in the footer, under the list it would update.
  const failure = listing || updating ? null : planError;
  const listFailed = failure != null && view == null;
  const shown = listing || listFailed ? null : view;
  const plan = shown?.plan;
  const status = updating ? UPDATING : footer;
  const empty =
    plan != null &&
    plan.entries.length === 0 &&
    plan.kept === 0 &&
    plan.hidden.restrict + plan.hidden.cascade + plan.hidden.detach === 0;

  return (
    <Modal.Frame
      ref={frameRef}
      className={CSS.cls(CSS.B("impact"), !painted && CSS.BM("impact", "unpainted"))}
    >
      <Modal.Header icon={text.icon} closeDisabled={applying}>
        {text.breadcrumb}
      </Modal.Header>
      <Modal.Body gap="medium" className={CSS.BE("impact", "body")}>
        {changed && (
          <Note.Note variant="warning" role="alert">
            <Text.Text level="small">
              These records changed. Check them again.
            </Text.Text>
          </Note.Note>
        )}
        <Text.Text level="h3" weight={450}>
          {text.message(names)}
        </Text.Text>
        <Text.Text weight={450}>{text.description(names, empty)}</Text.Text>
        {listing && (
          <Status.Summary
            variant="loading"
            message={`Listing the records that this ${text.noun} touches`}
          />
        )}
        {listFailed && (
          <Flex.Box
            y
            align="start"
            gap="medium"
            role="alert"
            className={CSS.BE("impact", "failure")}
          >
            <Status.Summary
              variant="error"
              message={`Failed to list the records that this ${text.noun} touches`}
              description={failure}
              center={false}
            />
            <Button.Button variant="outlined" onClick={handleRetry}>
              <Icon.Refresh />
              Try again
            </Button.Button>
          </Flex.Box>
        )}
        {shown != null && !empty && (
          <Entries
            view={shown}
            groups={text.groups(plural)}
            picks={picks}
            unpicked={unpicked}
            onPick={handlePick}
            hiddenCheck={{
              picked: hiddenPicked,
              unpicked: hiddenUnpicked,
              onChange: handleHidden,
            }}
            disabled={applying}
          />
        )}
        {plan != null && plan.kept > 0 && (
          <Text.Text level="small" color={9}>
            {plan.kept === 1
              ? "1 more record keeps a link and does not change."
              : `${plan.kept} more records keep a link and do not change.`}
          </Text.Text>
        )}
      </Modal.Body>
      <Modal.Footer>
        {/* A live region stays mounted, so a screen reader reads each status. */}
        <Nav.Bar.Start role="status" className={CSS.BE("impact", "status")}>
          {failure != null && !listFailed ? (
            <Status.Summary variant="error" description={failure} center={false}>
              Failed to update the records
              <Button.Button
                variant="text"
                size="tiny"
                className={CSS.BE("impact", "retry")}
                onClick={handleRetry}
              >
                Try again
              </Button.Button>
            </Status.Summary>
          ) : (
            status != null && (
              <Status.Summary
                variant={status.variant}
                message={status.message}
                description={status.description}
                center={false}
              />
            )
          )}
        </Nav.Bar.Start>
        <Nav.Bar.End x align="center">
          <Button.Button
            variant="outlined"
            triggerIndicator={PTriggers.ESCAPE}
            disabled={applying}
            onClick={() => close()}
          >
            Cancel
          </Button.Button>
          <Button.Button
            variant="filled"
            status={applying ? "loading" : "error"}
            disabled={shown == null || failure != null || planning || updating}
            trigger={Triggers.SAVE}
            triggerIndicator
            onClick={() => void handleApply()}
          >
            {text.verb}
          </Button.Button>
        </Nav.Bar.End>
      </Modal.Footer>
    </Modal.Frame>
  );
};

/**
 * Opens a dialog that plans change, shows every record that it touches, takes the
 * user's picks, and applies it. Resolves with the applied plan, or null when the user
 * cancels.
 */
export const useDialog = Modals.createPrompt<impact.Plan, DialogParams>(Dialog);
