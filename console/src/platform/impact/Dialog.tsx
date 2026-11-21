// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/impact/Dialog.css";

import { impact, PlanChangedError, resource } from "@synnaxlabs/client";
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
import { Text } from "@synnaxlabs/lyra/text";
import { Triggers as PTriggers } from "@synnaxlabs/lyra/triggers";
import { Resource, Synnax } from "@synnaxlabs/pluto";
import { type ReactElement, Suspense, useState } from "react";

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

const GROUPS: { effect: impact.Effect; title: string }[] = [
  { effect: "restrict", title: "Blocks the delete" },
  { effect: "cascade", title: "Deleted with it" },
  { effect: "detach", title: "Loses the channel" },
];

type Picks = Record<string, impact.Choice>;

const recordKey = (e: impact.Entry): string => resource.idToString(e.record);

const messageOf = (names: string[]): string =>
  names.length === 1 ? `Delete channel ${names[0]}` : `Delete ${names.length} channels`;

const descriptionOf = (names: string[]): string =>
  names.length === 1
    ? "Deletes the channel and its data."
    : "Deletes the channels and their data.";

interface Row {
  key: string;
  entry: impact.Entry;
  resource?: resource.Resource;
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
  row: { key, entry, resource: res },
  index,
  translate,
  picked,
  unpicked,
  autoFocus,
  onPick,
}: EntryProps): ReactElement => {
  const Item = Tree.useItems()[entry.record.type] ?? Tree.DefaultItem;
  const name = res?.name ?? entry.record.key;
  return (
    <List.Item<string>
      itemKey={key}
      index={index}
      translate={translate}
      className={CSS.BE("impact", "entry")}
      align="center"
      gap="medium"
    >
      {res != null && Item.icon(res)}
      <Text.Text overflow="fade" grow>
        {name}
      </Text.Text>
      {entry.effect === "restrict" && (
        <Flex.Box x align="center" gap="small" className={CSS.BE("impact", "pick")}>
          {unpicked && (
            <Text.Text level="small" status="error">
              Uses the channel
            </Text.Text>
          )}
          <Input.Checkbox
            aria-label={`Delete ${name} too`}
            value={picked}
            onChange={(v) => onPick(key, v)}
            autoFocus={autoFocus}
          />
          <Text.Text level="small">Delete too</Text.Text>
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
  onPick: (key: string, picked: boolean) => void;
}

const Group = ({
  title,
  rows,
  picks,
  unpicked,
  focusFirst,
  onPick,
}: GroupProps): ReactElement => {
  const [open, setOpen] = useState(true);
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
          <List.Scroll displayItems={DISPLAY_ITEMS}>
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

interface EntriesProps extends Omit<GroupProps, "title" | "rows" | "focusFirst"> {
  plan: impact.Plan;
}

const Entries = ({ plan, ...rest }: EntriesProps): ReactElement => {
  const resources = Resource.useMultiple({ ids: plan.entries.map((e) => e.record) });
  const byID = new Map(resources.map((r) => [resource.idToString(r.id), r]));
  return (
    <Flex.Box y gap="small">
      {GROUPS.map(({ effect, title }) => {
        const rows = plan.entries
          .filter((e) => e.effect === effect)
          .map((entry) => ({
            key: recordKey(entry),
            entry,
            resource: byID.get(recordKey(entry)),
          }));
        if (rows.length === 0) return null;
        return (
          <Group
            key={effect}
            title={title}
            rows={rows}
            focusFirst={effect === "restrict"}
            {...rest}
          />
        );
      })}
    </Flex.Box>
  );
};

const Dialog = ({
  change,
  names,
  close,
}: Modals.ContentProps<DialogParams, impact.Plan>): ReactElement => {
  const client = Synnax.use();
  const [picks, setPicks] = useState<Picks>({});
  const [hidden, setHidden] = useState(false);
  const [plan, setPlan] = useState<impact.Plan | null>(null);
  const [planning, setPlanning] = useState(true);
  const [planError, setPlanError] = useState<string | null>(null);
  const [changed, setChanged] = useState(false);
  const [unpicked, setUnpicked] = useState<Set<string>>(() => new Set());
  const [applying, setApplying] = useState(false);
  const [applyError, setApplyError] = useState<string | null>(null);
  Modals.useBusy(applying);
  const hiddenPick = hidden ? "delete" : undefined;

  useAsyncEffect(
    async (signal) => {
      if (client == null) return;
      setPlanning(true);
      try {
        const next = await client.impact.plan(change, { picks, hidden: hiddenPick });
        if (signal.aborted) return;
        setPlan(next);
        setPlanError(null);
        setPlanning(false);
      } catch (e) {
        if (signal.aborted) return;
        setPlanError((e as Error).message);
        setPlanning(false);
      }
    },
    [client, change, picks, hiddenPick],
  );

  const handlePick = (key: string, picked: boolean) => {
    setUnpicked((prev) => {
      const next = new Set(prev);
      next.delete(key);
      return next;
    });
    setPicks((prev) => {
      const next = { ...prev };
      if (picked) next[key] = "delete";
      else delete next[key];
      return next;
    });
  };

  const handleDelete = async () => {
    if (client == null || plan == null) return;
    const blockers = plan.entries
      .filter((e) => e.effect === "restrict" && picks[recordKey(e)] !== "delete")
      .map(recordKey);
    if (blockers.length > 0) {
      setUnpicked(new Set(blockers));
      return;
    }
    setApplying(true);
    setApplyError(null);
    try {
      const applied = await client.impact.apply(change, {
        token: plan.token,
        picks,
        hidden: hiddenPick,
      });
      setApplying(false);
      close(applied);
    } catch (e) {
      setApplying(false);
      if (!(e instanceof PlanChangedError)) {
        setApplyError((e as Error).message);
        return;
      }
      const next = impact.planOf(e);
      const kept = new Set(next.entries.map(recordKey));
      setPlan(next);
      setChanged(true);
      setPicks((prev) =>
        Object.fromEntries(Object.entries(prev).filter(([key]) => kept.has(key))),
      );
    }
  };

  const empty =
    plan != null && plan.entries.length === 0 && plan.kept === 0 && plan.hidden === 0;
  const hasBlockers = plan?.entries.some((e) => e.effect === "restrict") ?? false;

  return (
    <Modal.Frame className={CSS.B("impact")}>
      <Modal.Header icon={<Icon.Channel />}>Channel.Delete</Modal.Header>
      <Modal.Body gap="medium" className={CSS.BE("impact", "body")}>
        {changed && (
          <Note.Note variant="warning">
            <Text.Text level="small">
              These records changed. Check them again.
            </Text.Text>
          </Note.Note>
        )}
        <Text.Text level="h3" weight={450}>
          {messageOf(names)}
        </Text.Text>
        <Text.Text weight={450}>
          {descriptionOf(names)}
          {empty && " Nothing else uses it."}
        </Text.Text>
        {planError != null && (
          <Note.Note variant="error">
            <Text.Text level="small">{planError}</Text.Text>
          </Note.Note>
        )}
        {plan != null && plan.entries.length > 0 && (
          <Suspense fallback={null}>
            <Entries
              plan={plan}
              picks={picks}
              unpicked={unpicked}
              onPick={handlePick}
            />
          </Suspense>
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
            <Text.Text level="small">
              {plan.hidden === 1
                ? "1 record you cannot see."
                : `${plan.hidden} records you cannot see.`}
            </Text.Text>
            <Input.Checkbox
              aria-label="Delete the records you cannot see too"
              value={hidden}
              onChange={setHidden}
            />
            <Text.Text level="small">Delete them too</Text.Text>
          </Flex.Box>
        )}
      </Modal.Body>
      <Modal.Footer>
        <Flex.Box y grow gap="small">
          <Nav.Bar.End x align="center">
            <Button.Button
              variant="outlined"
              triggerIndicator={PTriggers.ESCAPE}
              onClick={() => close()}
            >
              Cancel
            </Button.Button>
            <Button.Button
              variant="filled"
              status={applying || planning ? "loading" : "error"}
              disabled={plan == null}
              trigger={Triggers.SAVE}
              triggerIndicator
              autoFocus={plan != null && !hasBlockers}
              onClick={() => void handleDelete()}
            >
              Delete
            </Button.Button>
          </Nav.Bar.End>
          <Note.Note
            variant="error"
            className={CSS.cls(
              CSS.BE("impact", "error"),
              applyError == null && CSS.M("ghost"),
            )}
          >
            <Text.Text level="small">{applyError ?? " "}</Text.Text>
          </Note.Note>
        </Flex.Box>
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
