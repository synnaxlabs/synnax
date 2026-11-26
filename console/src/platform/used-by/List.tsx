// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/used-by/List.css";

import { resource } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Header } from "@synnaxlabs/lyra/header";
import { Skeleton } from "@synnaxlabs/lyra/skeleton";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Errors, Resource } from "@synnaxlabs/pluto";
import { compare, errors } from "@synnaxlabs/x";
import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  use,
  useEffect,
  useRef,
  useState,
} from "react";

import { CSS } from "@/platform/css";
import { Tree } from "@/platform/tree";

/** Whether a use reads or writes the item. */
export type Direction = "read" | "write";

/** A resource that uses an item, with the direction of the use when it has one. */
export interface Use {
  id: resource.ID;
  direction?: Direction;
}

/** Selects a use of a resource type by its key, keyed by the type. */
export type Selects = Partial<Record<resource.Type, (key: string) => void>>;

/** The headings of the uses with a direction, in the order they show. */
const HEADINGS: { direction: Direction; title: string }[] = [
  { direction: "write", title: "Written by" },
  { direction: "read", title: "Read by" },
];

interface Group<T> {
  title?: string;
  items: T[];
}

/**
 * Groups the items by the direction of their use. The items with no direction come
 * first, with no heading. A group with no items is left out.
 */
const group = <T,>(
  items: T[],
  directionOf: (item: T) => Direction | undefined,
): Group<T>[] =>
  [
    { items: items.filter((item) => directionOf(item) == null) },
    ...HEADINGS.map(({ direction, title }) => ({
      title,
      items: items.filter((item) => directionOf(item) === direction),
    })),
  ].filter(({ items }) => items.length > 0);

interface ResolvedUse {
  resource: resource.Resource;
  direction?: Direction;
}

/** Orders uses by name, then by key, so equal names keep one order. */
const USE_ORDER = compare.chain<ResolvedUse>(
  compare.by(({ resource: r }) => r.name),
  compare.by(({ resource: r }) => r.id.key),
);

interface EntryProps {
  resource: resource.Resource;
  select?: (key: string) => void;
  onSelect?: () => void;
}

/** A use of the item. A click opens it as a double click on its tree item does. */
const Entry = ({ resource: r, select, onSelect }: EntryProps): ReactElement => {
  const Item = Tree.useItems()[r.id.type] ?? Tree.DefaultItem;
  const content = (
    <>
      {Item.icon(r)}
      <Text.Text level="small" overflow="fade" color={10}>
        {r.name}
      </Text.Text>
      {Item.typeName != null && (
        <Text.Text level="small" color={9} className={CSS.BE("used-by", "type")}>
          {Item.typeName}
        </Text.Text>
      )}
    </>
  );
  const renderButton = (onClick: () => void) => (
    <Button.Button
      variant="text"
      size="small"
      justify="start"
      full="x"
      className={CSS.BE("used-by", "resource")}
      tooltip={`Open ${r.name}`}
      tooltipLocation="left"
      onClick={onClick}
    >
      {content}
    </Button.Button>
  );
  if (select != null)
    return renderButton(() => {
      onSelect?.();
      select(r.id.key);
    });
  // A use that nothing opens is plain text, so it promises no action.
  if (!Item.selectable)
    return (
      <Text.Text
        el="div"
        level="small"
        x
        align="center"
        gap="small"
        className={CSS.BE("used-by", "row")}
      >
        {content}
      </Text.Text>
    );
  return <Item.OnSelect>{(open) => renderButton(() => open(r))}</Item.OnSelect>;
};

interface SectionProps extends PropsWithChildren {
  title?: string;
  count: number;
}

/** The uses in one direction, under a heading that opens and closes them. */
const Section = ({ title, count, children }: SectionProps): ReactElement => {
  const [open, setOpen] = useState(true);
  return (
    <Flex.Box
      y
      empty
      role="group"
      aria-label={title}
      className={CSS.BE("used-by", "section")}
    >
      {title != null && (
        <Header.Group size="small" level="small" open={open} onOpenChange={setOpen}>
          {title}
          <Text.Text level="tiny" color={9} className={CSS.BE("used-by", "count")}>
            {count}
          </Text.Text>
        </Header.Group>
      )}
      {(title == null || open) && children}
    </Flex.Box>
  );
};

interface UnusedProps {
  kind: string;
}

const Unused = ({ kind }: UnusedProps): ReactElement => (
  <Text.Text level="small" color={9} className={CSS.BE("used-by", "unused")}>
    Nothing uses this {kind}.
  </Text.Text>
);

/** Holds the place of each entry while the resources load, so nothing below moves. */
const Loading = ({ uses }: Pick<ListProps, "uses">): ReactElement => (
  <>
    {group(uses, ({ direction }) => direction).map(({ title, items }) => (
      <Section key={title ?? ""} title={title} count={items.length}>
        {items.map((_, i) => (
          <Flex.Box key={i} x align="center" className={CSS.BE("used-by", "row")}>
            <Skeleton.Skeleton className={CSS.BE("used-by", "skeleton")} />
          </Flex.Box>
        ))}
      </Section>
    ))}
  </>
);

interface Retry {
  /** Reads the uses again, then resets the boundary with reset. */
  retry: (reset: () => void) => void;
  /** Whether a retry is reading, so a failure after it focuses Retry. */
  retrying: boolean;
  /** Ends a retry once its read fails or the list shows. */
  settle: () => void;
}

const RetryContext = createContext<Retry | null>(null);
RetryContext.displayName = "UsedBy.Retry";

const Failed = ({ error, resetErrorBoundary }: Errors.FallbackProps): ReactElement => {
  const ctx = use(RetryContext);
  const button = useRef<HTMLButtonElement>(null);
  useEffect(() => {
    if (ctx?.retrying !== true) return;
    button.current?.focus();
    ctx.settle();
  }, [ctx]);
  if (ctx == null) throw new Error("Failed must be inside a used-by list");
  // A read wraps the error of the Core, which says why the read failed.
  const cause = Array.from(errors.causes(error)).at(-1);
  return (
    <Flex.Box y empty align="start">
      <div className={CSS.BE("used-by", "error")}>
        <Status.Summary
          variant="error"
          level="small"
          center={false}
          message="Failed to load the uses"
          description={errors.fromUnknown(cause ?? error).message}
        />
      </div>
      <Button.Button
        ref={button}
        variant="text"
        size="small"
        className={CSS.BE("used-by", "retry")}
        onClick={() => ctx.retry(resetErrorBoundary)}
      >
        Retry
      </Button.Button>
    </Flex.Box>
  );
};

const Resolved = ({ kind, uses, selects, onSelect }: ListProps): ReactElement => {
  // A resource in the uses can be deleted between the read of the uses and this read.
  const resources = Resource.useMultiple({ ids: uses.map(({ id }) => id) });
  const ctx = use(RetryContext);
  useEffect(() => {
    if (ctx?.retrying === true) ctx.settle();
  }, [ctx]);
  const byID = new Map(resources.map((r) => [resource.idToString(r.id), r]));
  const resolved = uses.flatMap(({ id, direction }): ResolvedUse[] => {
    const r = byID.get(resource.idToString(id));
    return r == null ? [] : [{ resource: r, direction }];
  });
  if (resolved.length === 0) return <Unused kind={kind} />;
  const groups = group(resolved.toSorted(USE_ORDER), ({ direction }) => direction);
  return (
    <>
      {groups.map(({ title, items }) => (
        <Section key={title ?? ""} title={title} count={items.length}>
          {items.map(({ resource: r }) => (
            <Entry
              key={resource.idToString(r.id)}
              resource={r}
              select={selects?.[r.id.type]}
              onSelect={onSelect}
            />
          ))}
        </Section>
      ))}
    </>
  );
};

export interface ListProps {
  /** The kind of the item, such as "secret". */
  kind: string;
  /** The resources that use the item. */
  uses: Use[];
  /** Selects a use of a resource type in place of its tree item. Keyed by the type. */
  selects?: Selects;
  /** Runs before a select from selects. */
  onSelect?: () => void;
}

const Uses = (props: ListProps): ReactElement => {
  const root = useRef<HTMLDivElement>(null);
  const [retrying, setRetrying] = useState(false);
  const invalidate = Resource.useInvalidateMultiple();
  const retry = (reset: () => void) => {
    // Focus stays in the list while the read runs again.
    root.current?.focus();
    invalidate({ ids: props.uses.map(({ id }) => id) });
    setRetrying(true);
    reset();
  };
  return (
    <Flex.Box y empty ref={root} tabIndex={-1} className={CSS.BE("used-by", "list")}>
      <RetryContext value={{ retry, retrying, settle: () => setRetrying(false) }}>
        <Errors.SuspenseBoundary
          FallbackComponent={Failed}
          loading={<Loading uses={props.uses} />}
        >
          <Resolved {...props} />
        </Errors.SuspenseBoundary>
      </RetryContext>
    </Flex.Box>
  );
};

/**
 * Lists the resources that use an item, by name. The uses with a direction show under
 * "Written by" and "Read by", each with its count. A click on a use opens it.
 */
export const List = (props: ListProps): ReactElement =>
  props.uses.length === 0 ? <Unused kind={props.kind} /> : <Uses {...props} />;
