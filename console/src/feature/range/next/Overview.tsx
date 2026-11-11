// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/range/next/Range.css";

import {
  type channel,
  type label,
  lineplot,
  log,
  query,
  ranger,
  type resource,
  schematic,
  table,
  task,
  type view,
} from "@synnaxlabs/client";
import { Badge } from "@synnaxlabs/lyra/badge";
import { Button } from "@synnaxlabs/lyra/button";
import { Color } from "@synnaxlabs/lyra/color";
import { Divider } from "@synnaxlabs/lyra/divider";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Theming } from "@synnaxlabs/lyra/theming";
import {
  Access,
  Channel,
  Flux,
  Label as PLabel,
  Panel as PlutoPanel,
  Ranger,
  Synnax,
} from "@synnaxlabs/pluto";
import { link, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import {
  type KeyboardEvent,
  type ReactElement,
  type ReactNode,
  useEffect,
  useState,
  useSyncExternalStore,
} from "react";

import { usePlot } from "@/feature/range/next/ContextMenu";
import {
  itemHeight,
  LayoutSelect,
  ORDERS,
  TreeLayout,
} from "@/feature/range/next/Explorer";
import { useOpen } from "@/feature/range/next/Item";
import { childrenQuery, useExpanded, useTree } from "@/feature/range/next/tree";
import { Core } from "@/platform/core";
import { CSS } from "@/platform/css";
import { Framer } from "@/platform/framer";
import { Label } from "@/platform/label";
import { Panel } from "@/platform/panel";
import { Range } from "@/platform/range";
import { Section as PSection } from "@/platform/section";
import { Session } from "@/session";

const path = Ranger.formPath;

interface RangeKeyProps {
  rangeKey: ranger.Key;
}

const Crumbs = ({ rangeKey }: RangeKeyProps): ReactElement | null => {
  const range = Ranger.useResult({ key: rangeKey }).data;
  const open = useOpen();
  if (range == null) return null;
  return (
    <>
      {range.parent != null && (
        <>
          <Crumbs rangeKey={range.parent} />
          <Icon.Caret.Right className={CSS.BE("range-overview", "separator")} />
        </>
      )}
      <Button.Button
        variant="text"
        size="small"
        textColor={9}
        gap="small"
        className={CSS.BE("range-overview", "crumb")}
        onClick={() => open(range.key)}
      >
        {range.name}
      </Button.Button>
    </>
  );
};

const Labels = ({ rangeKey, preview }: RangeKeyProps & { preview: boolean }) => {
  const value = Ranger.useResult({ key: rangeKey }).data?.labels ?? [];
  const { update } = PLabel.useEditFor();
  const handleChange = (next: label.Key[]): void =>
    update({
      id: ranger.resourceID(rangeKey),
      added: next.filter((key) => !value.includes(key)),
      removed: value.filter((key) => !next.includes(key)),
    });
  return (
    <Label.SelectMultiple
      zIndex={100}
      variant="floating"
      className={CSS.BE("range-overview", "labels-select")}
      value={value}
      onChange={handleChange}
      preview={preview}
    />
  );
};

interface SwatchProps {
  range: ranger.Range;
  canEdit: boolean;
}

const Swatch = ({ range, canEdit }: SwatchProps): ReactElement => {
  const { update } = Ranger.useSetColor();
  const { colors } = Theming.use();
  return (
    <Color.Swatch
      variant="shadow"
      value={range.color}
      fallback={colors.gray.l8}
      onChange={canEdit ? (color) => update({ key: range.key, color }) : undefined}
      onlyChangeOnBlur
      tooltip="Color"
      className={CSS.BE("range-overview", "color")}
    />
  );
};

const Header = ({ rangeKey }: RangeKeyProps): ReactElement => {
  const range = Ranger.use({ key: rangeKey });
  const pinned = Session.Range.useSelectState(rangeKey) != null;
  const canEdit = Access.useUpdateGranted(ranger.resourceID(rangeKey));
  const parent = Ranger.useResultParent({ key: rangeKey }).data;
  const { form } = Ranger.useForm({
    query: { key: rangeKey },
    mode: canEdit ? "normal" : "preview",
  });
  const name = Form.useFieldValue(path.name, { ctx: form });
  const plot = usePlot();
  const openCSV = Framer.useDownloadCSVModal();
  return (
    <Form.Form<typeof Ranger.formSchema> {...form}>
      <Flex.Box y className={CSS.BE("range-overview", "header")}>
        {range.parent != null && (
          <Flex.Box x align="center" className={CSS.BE("range-overview", "crumbs")}>
            <Crumbs rangeKey={range.parent} />
          </Flex.Box>
        )}
        <Flex.Box x align="center" justify="between" gap="large">
          <Form.TextField
            path={path.name}
            showLabel={false}
            className={CSS.BE("range-overview", "name")}
            inputProps={{
              variant: "text",
              level: "h1",
              placeholder: "Name",
              onlyChangeOnBlur: true,
              resetOnBlurIfEmpty: true,
            }}
            padHelpText={false}
          />
          <Flex.Box
            x
            align="center"
            gap="small"
            className={CSS.BE("range-overview", "actions")}
          >
            <Button.Button variant="outlined" gap="small" onClick={() => plot([range])}>
              <Icon.LinePlot />
              Plot
            </Button.Button>
            <Divider.Divider y />
            <Button.Copy
              text={() => `rng = client.ranges.retrieve(key="${rangeKey}")`}
              tooltip="Copy Python code"
              tooltipLocation="bottom"
              variant="text"
              textColor={9}
              successMessage={{ message: "Copied Python code to clipboard", name }}
            >
              <Icon.Python />
            </Button.Copy>
            <Button.Copy
              text={() => `const rng = await client.ranges.retrieve("${rangeKey}");`}
              tooltip="Copy TypeScript code"
              tooltipLocation="bottom"
              variant="text"
              textColor={9}
              successMessage={{ message: "Copied TypeScript code to clipboard", name }}
            >
              <Icon.TypeScript />
            </Button.Copy>
            <Button.Button
              tooltip="Download as CSV"
              tooltipLocation="bottom"
              variant="text"
              textColor={9}
              onClick={() =>
                openCSV({
                  timeRange: form.get(path.timeRange).value,
                  name,
                  channels: [],
                  icon: <Icon.Range />,
                })
              }
            >
              <Icon.CSV />
            </Button.Button>
            <Core.CopyLinkToolbarButton
              name={name}
              resourceID={ranger.resourceID(rangeKey)}
              tooltip="Copy link"
              tooltipLocation="bottom"
              textColor={9}
            />
            <Range.FavoriteButton
              range={range}
              size="medium"
              tooltip={pinned ? "Unpin" : "Pin"}
            />
          </Flex.Box>
        </Flex.Box>
        <Form.Field path={path.timeRange} padHelpText={false} showLabel={false}>
          {(p) => (
            <Ranger.Timeline
              level="p"
              variant="shadow"
              className={CSS.BE("range-overview", "timeline")}
              parent={parent?.timeRange.numeric}
              {...p}
            />
          )}
        </Form.Field>
        <Flex.Box x align="center" gap="small">
          <Swatch range={range} canEdit={canEdit} />
          <Labels rangeKey={rangeKey} preview={!canEdit} />
        </Flex.Box>
      </Flex.Box>
    </Form.Form>
  );
};

interface SectionProps {
  title: string;
  icon: Icon.ReactElement;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
}

/**
 * An outlined box with its title and actions in a bar at the top. Clicking the title
 * collapses the box to its bar, and any action opens it again.
 */
const Section = ({
  title,
  icon,
  actions,
  children,
  className,
}: SectionProps): ReactElement => {
  const [open, setOpen] = useState(true);
  return (
    <PSection.Section
      y
      empty
      className={CSS.cls(
        !open && CSS.BEM("range-overview", "section", "collapsed"),
        className,
      )}
    >
      <PSection.Header level="small" bordered={open}>
        <Button.Button
          variant="text"
          size="small"
          level="p"
          gap="small"
          textColor={10}
          aria-expanded={open}
          onClick={() => setOpen((prev) => !prev)}
        >
          <Text.Text el="span" color={9}>
            {icon}
          </Text.Text>
          <Text.Text el="span" weight={500} color={10}>
            {title}
          </Text.Text>
          <Text.Text el="span" color={8} className={CSS.BE("range-overview", "caret")}>
            <Icon.Caret.Down />
          </Text.Text>
        </Button.Button>
        {actions != null && (
          <Flex.Box x align="center" gap="small" onClickCapture={() => setOpen(true)}>
            {actions}
          </Flex.Box>
        )}
      </PSection.Header>
      {open && children}
    </PSection.Section>
  );
};

interface AddButtonProps {
  tooltip: string;
  onClick: () => void;
}

const AddButton = ({ tooltip, onClick }: AddButtonProps): ReactElement => (
  <Button.Button
    variant="text"
    size="small"
    textColor={9}
    tooltip={tooltip}
    tooltipLocation="left"
    onClick={onClick}
  >
    <Icon.Add />
  </Button.Button>
);

interface AddLineProps {
  onClick: () => void;
  children: string;
}

/** Stands in for an empty section: one muted line that adds its first item. */
const AddLine = ({ onClick, children }: AddLineProps): ReactElement => (
  <Button.Button
    variant="text"
    size="small"
    textColor={9}
    gap="small"
    className={CSS.BE("range-overview", "add")}
    onClick={onClick}
  >
    <Icon.Add />
    {children}
  </Button.Button>
);

type ChildrenList = Flux.UseListReturn<ranger.Key, ranger.Range>;

/**
 * A range longer than a day, such as a campaign, holds many children, so it lists them
 * newest first. A shorter one shows its children, such as phases, on a timeline.
 */
const defaultLayout = (range: ranger.Range, now: TimeStamp): view.RangeLayout => {
  const { start, end } = range.timeRange.makeValid();
  const span = start.span(end.before(now) ? end : now);
  return span.greaterThan(TimeSpan.DAY) ? "tree" : "timeline";
};

/** A tree shows the newest child first, and a timeline shows its lanes in order. */
const childOrder = (layout: view.RangeLayout): ranger.Order =>
  layout === "tree" ? ORDERS.newest : ORDERS.oldest;

const CHILD_LAYOUTS: view.RangeLayout[] = ["tree", "timeline"];

interface ChildrenProps {
  range: ranger.Range;
  list: ChildrenList;
  layout: view.RangeLayout;
  onLayoutChange: (layout: view.RangeLayout) => void;
  onCreate?: () => void;
}

const Children = ({
  range,
  list,
  layout,
  onLayoutChange,
  onCreate,
}: ChildrenProps): ReactElement => {
  const [expanded, toggle] = useExpanded();
  const [selected, setSelected] = useState<ranger.Key[]>([]);
  const tree = useTree({ roots: list.data, expanded });
  return (
    <Section
      title="Child ranges"
      icon={<Icon.Range />}
      className={CSS.BE("range-overview", "children")}
      actions={
        <>
          <LayoutSelect
            value={layout}
            onChange={onLayoutChange}
            layouts={CHILD_LAYOUTS}
          />
          {onCreate != null && (
            <AddButton tooltip="Create child range" onClick={onCreate} />
          )}
        </>
      }
    >
      <Select.Frame
        multiple
        data={tree.nodes.map((n) => n.key)}
        getItem={list.getItem}
        subscribe={list.subscribe}
        value={selected}
        onChange={setSelected}
        onFetchMore={list.more}
        virtual
        itemHeight={itemHeight(layout)}
      >
        <TreeLayout
          layout={layout}
          nodes={tree.nodes}
          expanded={expanded}
          onExpand={toggle}
          more={tree.more}
          bounds={range.timeRange}
        />
      </Select.Frame>
    </Section>
  );
};

interface FieldProps {
  name: ReactNode;
  children: ReactNode;
  /** Shows at the end of the field at all times. */
  note?: ReactNode;
  /** Shows at the end of the field on hover and focus. */
  actions?: ReactNode;
}

const Field = ({ name, children, note, actions }: FieldProps): ReactElement => (
  <div className={CSS.BE("range-overview", "field")}>
    {name}
    {children}
    <Flex.Box x align="center" gap="small">
      {note}
      {actions != null && (
        <Flex.Box
          x
          align="center"
          className={CSS.BE("range-overview", "field-actions")}
        >
          {actions}
        </Flex.Box>
      )}
    </Flex.Box>
  </div>
);

const FieldName = ({ children }: { children: ReactNode }): ReactElement => (
  <Text.Text
    color={9}
    overflow="fade"
    className={CSS.BE("range-overview", "field-name")}
  >
    {children}
  </Text.Text>
);

interface RemoveButtonProps {
  onClick: () => void;
}

const RemoveButton = ({ onClick }: RemoveButtonProps): ReactElement => (
  <Button.Button
    variant="text"
    size="small"
    textColor={9}
    tooltip="Remove"
    onClick={onClick}
  >
    <Icon.Close />
  </Button.Button>
);

interface PairProps extends RangeKeyProps {
  pair: ranger.kv.Pair;
}

const Pair = ({ rangeKey, pair }: PairProps): ReactElement => {
  const { update } = Ranger.useUpdateKV();
  const { update: remove } = Ranger.useDeleteKV();
  const isLink = link.is(pair.value);
  return (
    <Field
      name={<FieldName>{pair.key}</FieldName>}
      actions={
        <>
          {isLink && (
            <Button.Button
              href={pair.value}
              target="_blank"
              autoFormatHref
              variant="text"
              size="small"
              textColor={9}
              tooltip="Open link"
            >
              <Icon.LinkExternal />
            </Button.Button>
          )}
          <Button.Copy
            text={pair.value}
            variant="text"
            size="small"
            textColor={9}
            tooltip="Copy value"
            successMessage="Copied value to clipboard"
          />
          <RemoveButton onClick={() => remove({ rangeKey, key: pair.key })} />
        </>
      }
    >
      <Input.Text
        variant="shadow"
        value={pair.value}
        placeholder="Value"
        onlyChangeOnBlur
        className={CSS.BE("range-overview", "field-value")}
        textColor={isLink ? "var(--pluto-primary-z)" : undefined}
        onChange={(value) =>
          update({ rangeKey, range: rangeKey, key: pair.key, value })
        }
      />
    </Field>
  );
};

interface DraftProps extends RangeKeyProps {
  onDone: () => void;
}

const PairDraft = ({ rangeKey, onDone }: DraftProps): ReactElement => {
  const [key, setKey] = useState("");
  const [value, setValue] = useState("");
  const { update } = Ranger.useUpdateKV();
  const handleKeyDown = (e: KeyboardEvent) => {
    if (e.key === "Escape") onDone();
    if (e.key !== "Enter" || key.trim() === "") return;
    update({ rangeKey, range: rangeKey, key: key.trim(), value });
    onDone();
  };
  return (
    <Field
      name={
        <Input.Text
          autoFocus
          variant="shadow"
          placeholder="Key"
          value={key}
          onChange={setKey}
          onKeyDown={handleKeyDown}
          className={CSS.BE("range-overview", "field-input")}
        />
      }
      actions={<RemoveButton onClick={onDone} />}
    >
      <Input.Text
        variant="shadow"
        placeholder="Value"
        value={value}
        onChange={setValue}
        onKeyDown={handleKeyDown}
        className={CSS.BE("range-overview", "field-value")}
      />
    </Field>
  );
};

const byKey = (a: ranger.kv.Pair, b: ranger.kv.Pair) => a.key.localeCompare(b.key);

interface MetaDataProps extends RangeKeyProps {
  pairs: ranger.kv.Pair[];
  adding: boolean;
  onAdd: (adding: boolean) => void;
}

const MetaData = ({ rangeKey, pairs, adding, onAdd }: MetaDataProps): ReactElement => (
  <Section
    title="Metadata"
    icon={<Icon.Details />}
    actions={<AddButton tooltip="Add metadata" onClick={() => onAdd(true)} />}
  >
    <Flex.Box y empty className={CSS.BE("range-overview", "fields")}>
      {pairs.toSorted(byKey).map((pair) => (
        <Pair key={pair.key} rangeKey={rangeKey} pair={pair} />
      ))}
      {adding && <PairDraft rangeKey={rangeKey} onDone={() => onAdd(false)} />}
    </Flex.Box>
  </Section>
);

const NO_ALIASES: Record<channel.Key, string> = {};

interface AliasList {
  aliases: Record<channel.Key, string>;
  /** The channels whose alias comes from an ancestor of the range. */
  inherited: Set<channel.Key>;
}

/** @returns the channel aliases of the range, or null until the Core answers. */
const useAliases = (rangeKey: ranger.Key, parent?: ranger.Key): AliasList | null => {
  const client = Synnax.use();
  const [list, setList] = useState<AliasList | null>(null);
  useEffect(() => {
    if (client == null) return;
    let live = true;
    const load = () => {
      Promise.all([
        client.ranges.listAliases(rangeKey),
        parent == null ? NO_ALIASES : client.ranges.listAliases(parent),
      ]).then(([aliases, fromParent]) => {
        if (!live) return;
        const inherited = new Set(
          Object.keys(aliases)
            .map(Number)
            .filter((ch) => fromParent[ch] === aliases[ch]),
        );
        setList({ aliases, inherited });
      }, console.error);
    };
    load();
    const stop = client.ranges.aliases.subscribeBatch((events) => {
      const touched = events.some((e) => {
        if (!query.isChange(e)) return false;
        const { range } = ranger.alias.decodeDeleteChange(e.key);
        return range === rangeKey || range === parent;
      });
      if (touched) load();
    });
    return () => {
      live = false;
      stop();
    };
  }, [client, rangeKey, parent]);
  return list;
};

interface AliasProps extends RangeKeyProps {
  channel: channel.Key;
  alias: string;
  /** An inherited alias cannot be removed here. Changing it sets the range's own. */
  inherited: boolean;
}

const Alias = ({ rangeKey, channel, alias, inherited }: AliasProps): ReactElement => {
  const name = Channel.useResultName({ key: channel }).data;
  const { update } = Channel.useUpdateAlias();
  const { update: remove } = Channel.useDeleteAlias();
  const input = (
    <Input.Text
      variant="shadow"
      value={alias}
      placeholder="Alias"
      onlyChangeOnBlur
      resetOnBlurIfEmpty
      className={CSS.BE("range-overview", "field-value")}
      onChange={(next) => update({ range: rangeKey, channel, alias: next })}
    />
  );
  if (inherited)
    return (
      <Field
        name={<FieldName>{name}</FieldName>}
        note={
          <Text.Text
            level="small"
            color={8}
            className={CSS.BE("range-overview", "field-note")}
          >
            Inherited
          </Text.Text>
        }
      >
        {input}
      </Field>
    );
  return (
    <Field
      name={<FieldName>{name}</FieldName>}
      actions={
        <RemoveButton onClick={() => remove({ range: rangeKey, channels: channel })} />
      }
    >
      {input}
    </Field>
  );
};

const AliasDraft = ({ rangeKey, onDone }: DraftProps): ReactElement => {
  const [channel, setChannel] = useState<channel.Key | undefined>(undefined);
  const [alias, setAlias] = useState("");
  const { update } = Channel.useUpdateAlias();
  const handleKeyDown = (e: KeyboardEvent) => {
    if (e.key === "Escape") onDone();
    if (e.key !== "Enter" || channel == null || alias.trim() === "") return;
    update({ range: rangeKey, channel, alias: alias.trim() });
    onDone();
  };
  return (
    <Field
      name={
        <Channel.SelectSingle
          value={channel}
          onChange={setChannel}
          triggerProps={{ variant: "shadow", placeholder: "Channel" }}
          dialogProps={{ className: CSS.BE("range-overview", "channel-dialog") }}
          className={CSS.BE("range-overview", "field-input")}
        />
      }
      actions={<RemoveButton onClick={onDone} />}
    >
      <Input.Text
        variant="shadow"
        placeholder="Alias"
        value={alias}
        onChange={setAlias}
        onKeyDown={handleKeyDown}
        className={CSS.BE("range-overview", "field-value")}
      />
    </Field>
  );
};

interface AliasesProps extends RangeKeyProps, AliasList {
  adding: boolean;
  onAdd: (adding: boolean) => void;
}

const Aliases = ({
  rangeKey,
  aliases,
  inherited,
  adding,
  onAdd,
}: AliasesProps): ReactElement => (
  <Section
    title="Channel aliases"
    icon={<Icon.Channel />}
    actions={<AddButton tooltip="Add alias" onClick={() => onAdd(true)} />}
  >
    <Flex.Box y empty className={CSS.BE("range-overview", "fields")}>
      {Object.entries(aliases)
        .toSorted(([, a], [, b]) => a.localeCompare(b))
        .map(([key, alias]) => (
          <Alias
            key={key}
            rangeKey={rangeKey}
            channel={Number(key)}
            alias={alias}
            inherited={inherited.has(Number(key))}
          />
        ))}
      {adding && <AliasDraft rangeKey={rangeKey} onDone={() => onAdd(false)} />}
    </Flex.Box>
  </Section>
);

interface Document {
  id: resource.ID;
  name: string;
  snapshot: boolean;
}

interface Kind {
  name: string;
  icon: ReactElement;
}

const KINDS: Record<string, Kind> = {
  [lineplot.TYPE_RESOURCE_ID.type]: { name: "Line plot", icon: <Icon.LinePlot /> },
  [schematic.TYPE_RESOURCE_ID.type]: { name: "Schematic", icon: <Icon.Schematic /> },
  [table.TYPE_RESOURCE_ID.type]: { name: "Table", icon: <Icon.Table /> },
  [log.TYPE_RESOURCE_ID.type]: { name: "Log", icon: <Icon.Log /> },
  [task.TYPE_RESOURCE_ID.type]: { name: "Task", icon: <Icon.Task /> },
};

interface Owned {
  key: string;
  name: string;
  snapshot?: boolean;
}

const NO_RECORDS: Owned[] = [];
const noop = () => {};

const useOwned = (
  list: query.List<Owned, unknown> | undefined,
  type: resource.Type,
): Document[] => {
  const window = useSyncExternalStore(
    (callback) => list?.onChange(callback) ?? noop,
    () => list?.getCached(),
  );
  useEffect(() => {
    list?.retrieve().catch(console.error);
  }, [list]);
  return (window?.records ?? NO_RECORDS).map((r) => ({
    id: { type, key: r.key },
    name: r.name,
    snapshot: r.snapshot === true,
  }));
};

/** @returns every document that the range owns, by kind and then by name. */
const useDocuments = (rangeKey: ranger.Key): Document[] => {
  const client = Synnax.use();
  const where = { where: { ranges: [rangeKey] } };
  const plots = useOwned(
    client?.lineplots.watch(where),
    lineplot.TYPE_RESOURCE_ID.type,
  );
  const schematics = useOwned(
    client?.schematics.watch(where),
    schematic.TYPE_RESOURCE_ID.type,
  );
  const tables = useOwned(client?.tables.watch(where), table.TYPE_RESOURCE_ID.type);
  const logs = useOwned(client?.logs.watch(where), log.TYPE_RESOURCE_ID.type);
  const tasks = useOwned(client?.tasks.watch(where), task.TYPE_RESOURCE_ID.type);
  return [plots, schematics, tables, logs, tasks].flatMap((docs) =>
    docs.toSorted((a, b) => a.name.localeCompare(b.name)),
  );
};

const Documents = ({ documents }: { documents: Document[] }): ReactElement => {
  const openTab = Panel.useOpenTab();
  return (
    <Section title="Attached" icon={<Icon.Attachment />}>
      <Flex.Box y empty className={CSS.BE("range-overview", "fields")}>
        {documents.map(({ id, name, snapshot }) => {
          const kind = KINDS[id.type];
          return (
            <Button.Button
              key={`${id.type}:${id.key}`}
              variant="text"
              gap="medium"
              justify="start"
              className={CSS.BE("range-overview", "document")}
              onClick={() => openTab({ variant: "resource", resource: id })}
            >
              <Text.Text
                el="span"
                color={9}
                className={CSS.BE("range-overview", "document-icon")}
              >
                {kind.icon}
              </Text.Text>
              <Text.Text
                el="span"
                overflow="fade"
                className={CSS.BE("range-overview", "document-name")}
              >
                {name}
              </Text.Text>
              {snapshot && <Badge.Badge>Snapshot</Badge.Badge>}
              <Text.Text
                el="span"
                level="small"
                color={8}
                className={CSS.BE("range-overview", "document-kind")}
              >
                {kind.name}
              </Text.Text>
            </Button.Button>
          );
        })}
      </Flex.Box>
    </Section>
  );
};

type Adding = "metadata" | "alias" | null;

const NO_PAIRS: ranger.kv.Pair[] = [];

export const Overview: Panel.Content = () => {
  const { key } = PlutoPanel.useTabResource();
  const range = Ranger.use({ key });
  const [openedAt] = useState(() => TimeStamp.now());
  const [chosenLayout, setLayout] = useState<view.RangeLayout | null>(null);
  const layout = chosenLayout ?? defaultLayout(range, openedAt);
  const children = Flux.useList((client) => client.ranges, {
    name: "ranges",
    query: childrenQuery(key, childOrder(layout)),
    pageSize: 100,
  });
  useEffect(() => {
    children.more();
  }, [key]);
  const pairs = Ranger.useResultMetaData({ rangeKey: key }).data ?? NO_PAIRS;
  const aliasList = useAliases(key, range.parent ?? undefined);
  const documents = useDocuments(key);
  const [adding, setAdding] = useState<Adding>(null);
  const openCreate = Range.useCreateModal();
  const canCreate = Access.useCreateGranted(ranger.TYPE_RESOURCE_ID);
  const handleCreate = canCreate ? () => openCreate({ parent: key }) : undefined;
  const hasChildren = children.data.length > 0;
  const showMetaData = pairs.length > 0 || adding === "metadata";
  const aliasCount = aliasList == null ? 0 : Object.keys(aliasList.aliases).length;
  const showAliases = aliasCount > 0 || adding === "alias";
  const setAddingTo = (to: Adding) => (next: boolean) => setAdding(next ? to : null);
  return (
    <div key={key} className={CSS.BE("range-overview", "scroll")}>
      <Flex.Box y empty className={CSS.B("range-overview")}>
        <Flex.Box y empty className={CSS.BE("range-overview", "top")}>
          <Header rangeKey={key} />
        </Flex.Box>
        {hasChildren && (
          <Children
            range={range}
            list={children}
            layout={layout}
            onLayoutChange={setLayout}
            onCreate={handleCreate}
          />
        )}
        {children.answered && !hasChildren && handleCreate != null && (
          <AddLine onClick={handleCreate}>Child range</AddLine>
        )}
        {showMetaData ? (
          <MetaData
            rangeKey={key}
            pairs={pairs}
            adding={adding === "metadata"}
            onAdd={setAddingTo("metadata")}
          />
        ) : (
          <AddLine onClick={() => setAdding("metadata")}>Metadata</AddLine>
        )}
        {aliasList != null &&
          (showAliases ? (
            <Aliases
              rangeKey={key}
              {...aliasList}
              adding={adding === "alias"}
              onAdd={setAddingTo("alias")}
            />
          ) : (
            <AddLine onClick={() => setAdding("alias")}>Channel alias</AddLine>
          ))}
        {documents.length > 0 && <Documents documents={documents} />}
      </Flex.Box>
    </div>
  );
};
