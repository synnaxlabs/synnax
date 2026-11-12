// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/vcs/Plan.css";

import { vcs } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Synnax } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";
import { Fragment, type ReactElement, useEffect, useState } from "react";

import { useApply } from "@/feature/vcs/actions";
import {
  changeCount,
  type Choices,
  chooseAll,
  group,
  isReady,
  runningNote,
} from "@/feature/vcs/choices";
import {
  Empty,
  Footer,
  ItemList,
  Loading,
  ResourceItem,
  SectionHeader,
  ViewHeader,
} from "@/feature/vcs/common";
import { Diff } from "@/feature/vcs/diff";
import { CSS } from "@/platform/css";
import { Triggers } from "@/platform/triggers";
import { Vcs } from "@/platform/vcs";

const KINDS: Record<vcs.PlanItemKind, vcs.ChangeKind> = {
  create: "added",
  change: "modified",
  delete: "deleted",
};

interface ChoiceButtonsProps {
  value: vcs.ChoiceValue | undefined;
  onChange: (value: vcs.ChoiceValue) => void;
}

const ChoiceButtons = ({ value, onChange }: ChoiceButtonsProps): ReactElement => (
  <Flex.Box x pack className={CSS.BE("vcs-plan", "choice")}>
    <Button.Button
      size="small"
      variant={value === "mine" ? "filled" : "outlined"}
      onClick={() => onChange("mine")}
    >
      Keep mine
    </Button.Button>
    <Button.Button
      size="small"
      variant={value === "incoming" ? "filled" : "outlined"}
      onClick={() => onChange("incoming")}
    >
      Take incoming
    </Button.Button>
  </Flex.Box>
);

const OTHER_SIDE: Record<vcs.TargetKind, string> = {
  pull: "the incoming commits",
  restore: "the restored version",
  load: "the loaded configuration",
};

interface ItemProps {
  item: vcs.PlanItem;
  selected: boolean;
  choice: vcs.ChoiceValue | undefined;
  onChoose: (value: vcs.ChoiceValue) => void;
  onOpen: () => void;
}

const Item = ({
  item,
  selected,
  choice,
  onChoose,
  onOpen,
}: ItemProps): ReactElement => {
  const note = runningNote(item);
  const below =
    item.conflict || note != null ? (
      <Flex.Box y gap="small" align="start" className={CSS.BE("vcs-plan", "below")}>
        {note != null && (
          <Text.Text
            level="small"
            gap="tiny"
            status={item.restarts ? "warning" : undefined}
            color={item.restarts ? undefined : 8}
          >
            {item.restarts && <Icon.Refresh />}
            {note}
          </Text.Text>
        )}
        {item.conflict && <ChoiceButtons value={choice} onChange={onChoose} />}
      </Flex.Box>
    ) : undefined;
  return (
    <ResourceItem
      itemKey={item.resource}
      type={item.type}
      name={item.name}
      selected={selected}
      onClick={onOpen}
      below={below}
      end={
        <>
          {item.conflict && (
            <Text.Text
              level="small"
              status={choice == null ? "error" : undefined}
              color={choice == null ? undefined : 8}
              weight={500}
            >
              Conflict
            </Text.Text>
          )}
          <Vcs.KindMarker kind={KINDS[item.kind]} />
        </>
      }
    />
  );
};

export interface ReviewProps {
  view: Vcs.PlanView;
}

/**
 * Shows what a pull, restore, or load would change on the Core, settles conflicts, and
 * applies it.
 */
export const Review = ({ view }: ReviewProps): ReactElement => {
  const client = Synnax.use();
  const update = Vcs.useUpdateUI();
  const apply = useApply();
  const openDiff = Diff.useOpen();
  const shows = Diff.useShows();
  const handleError = Status.useErrorHandler();
  const [plan, setPlan] = useState<vcs.Plan | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [choices, setChoices] = useState<Choices>({});
  const [applying, setApplying] = useState(false);
  const openItem = ({ resource, type, name }: vcs.PlanItem) =>
    openDiff({
      resource,
      type,
      name,
      before: vcs.WORKING,
      beforeLabel: "Now",
      after: view.after,
      afterLabel: view.afterLabel,
    });
  useEffect(() => {
    if (client == null) return;
    let cancelled = false;
    setPlan(null);
    setError(null);
    setChoices({});
    client.vcs
      .plan(view.target)
      .then((p) => {
        if (cancelled) return;
        setPlan(p);
        const first = group(p).at(0)?.items.at(0);
        if (first != null) openItem(first);
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(errors.fromUnknown(e).message);
      });
    return () => {
      cancelled = true;
    };
  }, [client, view]);
  const back = () => update((prev) => ({ ...prev, view: { variant: "main" } }));
  const conflicts = plan?.items.filter(({ conflict }) => conflict) ?? [];
  const unsettled = conflicts.filter(
    ({ resource }) => choices[resource] == null,
  ).length;
  const ready = plan != null && isReady(plan, choices);
  const n = plan != null ? changeCount(plan, choices) : 0;
  const handleApply = async () => {
    if (plan == null) return;
    setApplying(true);
    try {
      await apply(plan, choices, view);
    } catch (e) {
      handleError(e, { message: "Failed to apply changes", name: view.title });
    }
    setApplying(false);
  };
  let body: ReactElement;
  if (error != null)
    body = (
      <Status.Summary
        variant="error"
        level="small"
        center={false}
        message="Failed to plan changes"
        description={error}
        className={CSS.BE("vcs", "status")}
      />
    );
  else if (plan == null) body = <Loading />;
  else if (plan.items.length === 0)
    body = (
      <Empty
        message="Nothing changes on the Core"
        description={
          view.target.kind === "restore"
            ? "The live configuration already matches this version."
            : "The live configuration already matches."
        }
      />
    );
  else
    body = (
      <>
        {conflicts.length > 0 && (
          <Flex.Box
            y
            gap="medium"
            shrink={false}
            className={CSS.BE("vcs-plan", "conflicts")}
          >
            <Status.Summary
              variant={unsettled > 0 ? "error" : "success"}
              level="small"
              center={false}
              message={
                unsettled > 0
                  ? `${Vcs.count(unsettled, "conflict")} to settle`
                  : `${Vcs.count(conflicts.length, "conflict")} settled`
              }
              description={`${conflicts.length === 1 ? conflicts[0].name : "These"} changed on this Core and in ${OTHER_SIDE[view.target.kind]}. Choose which version to keep.`}
            />
            {conflicts.length > 1 && (
              <Flex.Box x gap="small">
                <Button.Button
                  size="small"
                  variant="outlined"
                  onClick={() => setChoices(chooseAll(plan, "mine"))}
                >
                  Keep all mine
                </Button.Button>
                <Button.Button
                  size="small"
                  variant="outlined"
                  onClick={() => setChoices(chooseAll(plan, "incoming"))}
                >
                  Take all incoming
                </Button.Button>
              </Flex.Box>
            )}
          </Flex.Box>
        )}
        <ItemList>
          {group(plan).map(({ kind, title, items }) => (
            <Fragment key={kind}>
              <SectionHeader title={title} detail={items.length} />
              {items.map((item) => (
                <Item
                  key={item.resource}
                  item={item}
                  selected={shows(item.resource, vcs.WORKING, view.after)}
                  choice={choices[item.resource]}
                  onChoose={(value) =>
                    setChoices((prev) => ({ ...prev, [item.resource]: value }))
                  }
                  onOpen={() => openItem(item)}
                />
              ))}
            </Fragment>
          ))}
        </ItemList>
      </>
    );
  const empty = plan != null && plan.items.length === 0;
  const nothingToDo = empty && view.target.kind === "restore";
  let applyLabel = `Apply ${Vcs.count(n, "change")}`;
  if (nothingToDo) applyLabel = "Done";
  else if (plan != null && !ready) applyLabel = "Apply changes";
  else if (n === 0 && view.target.kind === "pull") applyLabel = "Pull";
  else if (n === 0 && view.target.kind === "load")
    applyLabel = `Load ${view.target.configuration ?? ""}`;
  const disabled = plan == null || !ready || applying;
  let note = "Applies in one step. You can undo it.";
  if (plan != null && !ready) note = "Settle each conflict to apply.";
  return (
    <Flex.Box y full empty className={CSS.B("vcs-plan")}>
      <ViewHeader title={view.title} onBack={back} />
      {body}
      <Footer>
        {!nothingToDo && plan != null && (
          <Text.Text level="small" color={8}>
            {note}
          </Text.Text>
        )}
        <Button.Button
          variant="filled"
          justify="center"
          full="x"
          disabled={disabled}
          status={applying ? "loading" : undefined}
          onClick={nothingToDo ? back : () => void handleApply()}
          trigger={disabled ? undefined : Triggers.SAVE}
          triggerIndicator={nothingToDo ? undefined : Triggers.SAVE}
        >
          {applyLabel}
        </Button.Button>
      </Footer>
    </Flex.Box>
  );
};
