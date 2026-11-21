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
  PlanChangedError,
  resource,
  type Synnax as Client,
  ValidationError,
} from "@synnaxlabs/client";
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
import { compare } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useId, useState } from "react";

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
/** How long a plan loads before the loading line shows. */
const LOADING_DELAY_MS = 150;

interface GroupSpec {
  effect: impact.Effect;
  title: string;
}

const groupsOf = (plural: boolean): GroupSpec[] => [
  { effect: "restrict", title: "Blocks the delete" },
  { effect: "cascade", title: plural ? "Deleted with them" : "Deleted with it" },
  { effect: "detach", title: plural ? "Loses a channel" : "Loses the channel" },
];

type Picks = Record<string, impact.Choice>;

const recordKey = (e: impact.Entry): string => resource.idToString(e.record);

const messageOf = (names: string[]): string =>
  names.length === 1 ? `Delete channel ${names[0]}` : `Delete ${names.length} channels`;

const descriptionOf = (names: string[], empty: boolean): string => {
  const one = names.length === 1;
  const text = one
    ? "Deletes the channel and its data."
    : "Deletes the channels and their data.";
  if (!empty) return text;
  return `${text} Nothing else uses ${one ? "it" : "them"}.`;
};

const failureOf = (names: string[]): string =>
  names.length === 1 ? "Failed to delete channel" : "Failed to delete channels";

const blockedOf = (count: number): string =>
  count === 1 ? "1 record blocks the delete" : `${count} records block the delete`;

const ACCESS_DENIED =
  "Your role cannot change a record that this delete changes. Ask an admin for " +
  "access, or cancel.";

const reasonOf = (err: unknown): string => {
  const text = err instanceof Error ? err.message : String(err);
  if (!AccessDeniedError.matches(err)) return text;
  return `${ACCESS_DENIED} ${text}`;
};

/** The next step when a record that the user cannot see and did not pick blocks. */
const hiddenStepOf = (hidden: number): string =>
  `Check ${hidden === 1 ? "Delete it too" : "Delete them too"}, or cancel.`;

/** A plan with the records it names, as the dialog shows it. */
interface View {
  plan: impact.Plan;
  resources: Map<string, resource.Resource>;
  /** fresh holds each record that the plan lists and the plan before it did not. */
  fresh: Set<string>;
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
  const ids = plan.entries.map((e) => e.record);
  const found = ids.length === 0 ? [] : await client.resources.retrieve(ids);
  const fresh = new Set<string>();
  if (previous != null) {
    const listed = new Set(previous.entries.map(recordKey));
    for (const e of plan.entries)
      if (!listed.has(recordKey(e))) fresh.add(recordKey(e));
  }
  return {
    plan,
    resources: new Map(found.map((r) => [resource.idToString(r.id), r])),
    fresh,
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
  compare.by((r) => r.resource?.name ?? r.entry.record.key),
  compare.by((r) => r.key),
);

interface FooterStatus {
  message: string;
  description: string;
}

interface EntryProps {
  row: Row;
  index: number;
  translate?: number;
  picked: boolean;
  unpicked: boolean;
  autoFocus: boolean;
  onPick: (key: string, picked: boolean) => void;
}

const Entry = ({
  row: { key, entry, resource: res, fresh },
  index,
  translate,
  picked,
  unpicked,
  autoFocus,
  onPick,
}: EntryProps): ReactElement => {
  const id = useId();
  const Item = Tree.useItems()[entry.record.type] ?? Tree.DefaultItem;
  const TypeIcon = TYPE_ICONS[entry.record.type];
  const name = res?.name ?? entry.record.key;
  const treeIcon = res == null ? undefined : Item.icon(res);
  const icon = treeIcon ?? (TypeIcon == null ? null : <TypeIcon />);
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
      <Text.Text id={`${id}-name`} overflow="fade" grow>
        {name}
      </Text.Text>
      {fresh && <Badge.Badge status="warning">New</Badge.Badge>}
      {entry.effect === "restrict" && (
        <Flex.Box x align="center" gap="small" className={CSS.BE("impact", "pick")}>
          {unpicked && (
            <Text.Text
              id={`${id}-error`}
              level="small"
              className={CSS.BE("impact", "error")}
            >
              Uses the channel
            </Text.Text>
          )}
          <Input.Checkbox
            id={id}
            aria-labelledby={`${id}-name ${id}-label`}
            aria-describedby={unpicked ? `${id}-error` : undefined}
            value={picked}
            onChange={(v) => onPick(key, v)}
            autoFocus={autoFocus}
          />
          <Text.Text el="label" id={`${id}-label`} htmlFor={id} level="small">
            Delete too
          </Text.Text>
        </Flex.Box>
      )}
    </List.Item>
  );
};

interface GroupProps {
  title: string;
  rows: Row[];
  picks: Picks;
  unpicked: Set<string>;
  focusFirst: boolean;
  initiallyOpen: boolean;
  onPick: (key: string, picked: boolean) => void;
}

const Group = ({
  title,
  rows,
  picks,
  unpicked,
  focusFirst,
  initiallyOpen,
  onPick,
}: GroupProps): ReactElement => {
  const [open, setOpen] = useState(initiallyOpen);
  const byKey = new Map(rows.map((r) => [r.key, r]));
  const { data, getItem } = List.useStaticData<string, Row>({ data: rows });
  return (
    <section className={CSS.BE("impact", "group")} aria-label={title}>
      <Header.Group size="medium" open={open} onOpenChange={setOpen}>
        {title}
        <Text.Text level="tiny" color={9}>
          {rows.length}
        </Text.Text>
      </Header.Group>
      {open && (
        <List.Frame<string, Row>
          data={data}
          getItem={getItem}
          itemHeight={ITEM_HEIGHT}
          virtual
        >
          <List.Scroll
            displayItems={DISPLAY_ITEMS}
            className={CSS.BE("impact", "scroll")}
            {...(rows.length > DISPLAY_ITEMS && {
              tabIndex: 0,
              role: "region",
              "aria-label": `${title}, ${rows.length} records`,
            })}
          >
            <List.Items<string>>
              {({ key, index, translate }) => (
                <Entry
                  key={key}
                  row={byKey.get(key) as Row}
                  index={index}
                  translate={translate}
                  picked={picks[key] === "delete"}
                  unpicked={unpicked.has(key)}
                  autoFocus={focusFirst && index === 0}
                  onPick={onPick}
                />
              )}
            </List.Items>
          </List.Scroll>
        </List.Frame>
      )}
    </section>
  );
};

interface EntriesProps extends Pick<GroupProps, "picks" | "unpicked" | "onPick"> {
  view: View;
  plural: boolean;
}

const Entries = ({ view, plural, ...rest }: EntriesProps): ReactElement => {
  const many = view.plan.entries.length > MANY_ENTRIES;
  const groups = groupsOf(plural);
  const first = groups.find(({ effect }) =>
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
        if (rows.length === 0) return null;
        const initiallyOpen = !many || effect === first?.effect;
        return (
          <Group
            key={effect}
            title={title}
            rows={rows}
            focusFirst={effect === "restrict"}
            initiallyOpen={initiallyOpen}
            {...rest}
          />
        );
      })}
    </Flex.Box>
  );
};

interface SummaryProps {
  status: FooterStatus;
}

const Summary = ({ status: { message, description } }: SummaryProps): ReactElement => (
  <Flex.Box y empty role="status" className={CSS.BE("impact", "summary")}>
    <Status.Summary variant="error" message={message} />
    <Text.Text level="small" color={9}>
      {description}
    </Text.Text>
  </Flex.Box>
);

const Dialog = ({
  change,
  names,
  close,
}: Modals.ContentProps<DialogParams, impact.Plan>): ReactElement => {
  const client = Synnax.use();
  const hiddenID = useId();
  const [picks, setPicks] = useState<Picks>({});
  const [hidden, setHidden] = useState(false);
  const [view, setView] = useState<View | null>(null);
  const [planning, setPlanning] = useState(true);
  const [planError, setPlanError] = useState<FooterStatus | null>(null);
  const [changed, setChanged] = useState(false);
  const [unpicked, setUnpicked] = useState<Set<string>>(() => new Set());
  const [applying, setApplying] = useState(false);
  const [footer, setFooter] = useState<FooterStatus | null>(null);
  const [slow, setSlow] = useState(false);
  Modals.useBusy(applying);
  const hiddenPick = hidden ? "delete" : undefined;

  useEffect(() => {
    if (view != null) return;
    const timer = setTimeout(() => setSlow(true), LOADING_DELAY_MS);
    return () => clearTimeout(timer);
  }, [view]);
  const plural = names.length > 1;

  useAsyncEffect(
    async (signal) => {
      if (client == null) return;
      setPlanning(true);
      let next: View | null = null;
      let failure: unknown = null;
      try {
        const planned = await client.impact.plan(change, { picks, hidden: hiddenPick });
        next = await createView(client, planned, null);
      } catch (e) {
        failure = e;
      }
      if (signal.aborted) return;
      setPlanning(false);
      if (next == null) {
        setPlanError({
          message: "Failed to list the records that this delete touches",
          description: reasonOf(failure),
        });
        return;
      }
      const planned = next;
      setPlanError(null);
      setView((prev) => (prev == null ? planned : { ...planned, fresh: prev.fresh }));
    },
    [client, change, picks, hiddenPick],
  );

  // A pick plans again, so Delete waits for that plan.
  const handlePick = (key: string, picked: boolean) => {
    setPlanning(true);
    setUnpicked((prev) => {
      const next = new Set(prev);
      next.delete(key);
      return next;
    });
    setFooter(null);
    setPicks((prev) => {
      const next = { ...prev };
      if (picked) next[key] = "delete";
      else delete next[key];
      return next;
    });
  };

  const handleDelete = async () => {
    if (client == null || view == null) return;
    const blockers = view.plan.entries
      .filter((e) => e.effect === "restrict" && picks[recordKey(e)] !== "delete")
      .map(recordKey);
    if (blockers.length > 0) {
      setUnpicked(new Set(blockers));
      setFooter({
        message: blockedOf(blockers.length),
        description: "Delete each one too, or cancel.",
      });
      return;
    }
    setApplying(true);
    setFooter(null);
    let applied: impact.Plan | null = null;
    let failure: unknown = null;
    try {
      applied = await client.impact.apply(change, {
        token: view.plan.token,
        picks,
        hidden: hiddenPick,
      });
    } catch (e) {
      failure = e;
    }
    if (applied != null) {
      setApplying(false);
      close(applied);
      return;
    }
    if (!(failure instanceof PlanChangedError)) {
      setApplying(false);
      let description = reasonOf(failure);
      if (ValidationError.matches(failure) && view.plan.hidden > 0 && !hidden)
        description = `${description}. ${hiddenStepOf(view.plan.hidden)}`;
      setFooter({ message: failureOf(names), description });
      return;
    }
    const current = impact.planOf(failure);
    let next: View | null = null;
    try {
      next = await createView(client, current, view.plan);
    } catch (e) {
      failure = e;
    }
    setApplying(false);
    if (next == null) {
      setFooter({ message: failureOf(names), description: reasonOf(failure) });
      return;
    }
    const kept = new Set(current.entries.map(recordKey));
    setView(next);
    setChanged(true);
    setPicks((prev) =>
      Object.fromEntries(Object.entries(prev).filter(([key]) => kept.has(key))),
    );
  };

  const plan = view?.plan;
  const empty =
    plan != null && plan.entries.length === 0 && plan.kept === 0 && plan.hidden === 0;

  return (
    <Modal.Frame className={CSS.B("impact")}>
      <Modal.Header icon={<Icon.Channel />}>Channel.Delete</Modal.Header>
      <Modal.Body gap="medium" className={CSS.BE("impact", "body")}>
        {changed && (
          <Note.Note variant="warning" role="status">
            <Text.Text level="small">
              These records changed. Check them again.
            </Text.Text>
          </Note.Note>
        )}
        <Text.Text level="h3" weight={450}>
          {messageOf(names)}
        </Text.Text>
        <Text.Text weight={450}>{descriptionOf(names, empty)}</Text.Text>
        {view == null && planError == null && slow && (
          <Status.Summary
            variant="loading"
            message="Listing the records that this delete touches"
          />
        )}
        {planError != null && <Summary status={planError} />}
        {view != null && view.plan.entries.length > 0 && (
          <Entries
            view={view}
            plural={plural}
            picks={picks}
            unpicked={unpicked}
            onPick={handlePick}
          />
        )}
        {plan != null && plan.kept > 0 && (
          <Text.Text level="small" color={9}>
            {plan.kept === 1
              ? "1 more record keeps a link and does not change."
              : `${plan.kept} more records keep a link and do not change.`}
          </Text.Text>
        )}
        {plan != null && plan.hidden > 0 && (
          <Flex.Box x align="center" gap="small">
            <Text.Text level="small" id={`${hiddenID}-count`}>
              {plan.hidden === 1
                ? "1 record you cannot see."
                : `${plan.hidden} records you cannot see.`}
            </Text.Text>
            <Input.Checkbox
              id={hiddenID}
              aria-labelledby={`${hiddenID}-count ${hiddenID}-label`}
              value={hidden}
              onChange={(v) => {
                setPlanning(true);
                setHidden(v);
              }}
            />
            <Text.Text
              el="label"
              id={`${hiddenID}-label`}
              htmlFor={hiddenID}
              level="small"
            >
              {plan.hidden === 1 ? "Delete it too" : "Delete them too"}
            </Text.Text>
          </Flex.Box>
        )}
      </Modal.Body>
      <Modal.Footer>
        <Nav.Bar.Start className={CSS.BE("impact", "status")}>
          {footer != null && <Summary status={footer} />}
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
            disabled={view == null || planning || planError != null}
            trigger={Triggers.SAVE}
            triggerIndicator
            onClick={() => void handleDelete()}
          >
            Delete
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
