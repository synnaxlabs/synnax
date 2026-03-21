// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Button,
  Flex,
  Haul,
  Icon,
  Nav,
  Status,
  Synnax,
  Telem,
  Text,
  Tree,
} from "@synnaxlabs/pluto";
import { DataType } from "@synnaxlabs/x";
import { open } from "@tauri-apps/plugin-dialog";
import { readFile } from "@tauri-apps/plugin-fs";
import { type ReactElement, useCallback, useMemo, useState } from "react";

import { Cluster } from "@/cluster";
import {
  type AnalysisItem,
  analyzeBackup,
  type AnalyzeResponse,
  type ConflictPolicy,
  type ConflictStatus,
  executeImport,
  type ImportResponse,
} from "@/import/upload";
import { type Layout } from "@/layout";
import { Modals } from "@/modals";
import { Triggers } from "@/triggers";

export const IMPORT_LAYOUT_TYPE = "importSynnax";

export const IMPORT_LAYOUT: Layout.BaseState = {
  key: IMPORT_LAYOUT_TYPE,
  type: IMPORT_LAYOUT_TYPE,
  name: "Import Synnax",
  icon: "Import",
  location: "modal",
  window: { resizable: true, size: { height: 600, width: 700 }, navTop: true },
};

type Step = "upload" | "review" | "results";

const SY_FILTERS = [{ name: "Synnax Backup", extensions: ["sy"] }];

const STATUS_COLORS: Record<string, string> = {
  new: "var(--pluto-secondary-z)",
  conflict: "var(--pluto-warning-z)",
  identical: "var(--pluto-gray-l7)",
};

const STATUS_LABELS: Record<string, string> = {
  new: "New",
  conflict: "Conflict",
  identical: "Identical",
};

const SECTION_ICONS: Record<string, ReactElement> = {
  workspace: <Icon.Workspace />,
  channel: <Icon.Channel />,
  device: <Icon.Device />,
  task: <Icon.Task />,
  range: <Icon.Range />,
  user: <Icon.User />,
  lineplot: <Icon.LinePlot />,
  schematic: <Icon.Schematic />,
  table: <Icon.Table />,
  log: <Icon.Log />,
};

const TYPE_LABELS: Record<string, string> = {
  workspace: "Workspaces",
  channel: "Channels",
  device: "Devices",
  task: "Tasks",
  range: "Ranges",
  user: "Users",
  lineplot: "Line Plots",
  schematic: "Schematics",
  table: "Tables",
  log: "Logs",
};

const WORKSPACE_CHILD_TYPES = new Set([
  "lineplot",
  "schematic",
  "table",
  "log",
]);

const SECTION_ORDER = [
  "workspace",
  "user",
  "device",
  "task",
  "range",
  "channel",
];

interface ItemLookup {
  items: Map<string, AnalysisItem>;
  sections: Map<string, string>;
  childItems: Map<string, AnalysisItem[]>;
}

const STATUS_PRIORITY: Record<string, number> = {
  conflict: 2,
  new: 1,
  identical: 0,
};

const aggregateStatus = (
  item: AnalysisItem,
  childItems: AnalysisItem[],
): ConflictStatus => {
  let maxPriority = STATUS_PRIORITY[item.status] ?? 0;
  let maxStatus = item.status;
  for (const child of childItems) {
    const p = STATUS_PRIORITY[child.status] ?? 0;
    if (p > maxPriority) {
      maxPriority = p;
      maxStatus = child.status;
    }
  }
  return maxStatus;
};

const buildTreeNodes = (
  items: AnalysisItem[],
): { nodes: Tree.Node[]; lookup: ItemLookup } => {
  const lookup: ItemLookup = {
    items: new Map(items.map((i) => [i.archive_key, i])),
    sections: new Map(),
    childItems: new Map(),
  };

  const topLevel: Map<string, AnalysisItem[]> = new Map();
  // Items nested under a parent (workspace children + grouped channels)
  const nestedChildren: Map<string, AnalysisItem[]> = new Map();

  const addNested = (parentName: string, item: AnalysisItem) => {
    const arr = nestedChildren.get(parentName) ?? [];
    if (arr.length === 0) nestedChildren.set(parentName, arr);
    arr.push(item);
  };

  const addTopLevel = (section: string, item: AnalysisItem) => {
    const arr = topLevel.get(section) ?? [];
    if (arr.length === 0) topLevel.set(section, arr);
    arr.push(item);
  };

  for (const item of items)
    if (WORKSPACE_CHILD_TYPES.has(item.type) && item.parent_name != null)
      addNested(item.parent_name, item);
    else if (
      item.type === "channel" &&
      item.parent_name != null &&
      item.parent_name.length > 0
    )
      addNested(item.parent_name, item);
    else addTopLevel(item.type === "workspace" ? "workspace" : item.type, item);

  const sectionNodes: Tree.Node[] = [];

  for (const sectionType of SECTION_ORDER) {
    const sectionItems = topLevel.get(sectionType);
    if (sectionItems == null || sectionItems.length === 0) continue;

    const sectionKey = `section:${sectionType}`;
    lookup.sections.set(sectionKey, sectionType);

    let children: Tree.Node[];
    if (sectionType === "channel") {
      // Group channels by parent_name (channel groups)
      const groupMap: Map<string, Tree.Node[]> = new Map();
      const ungrouped: Tree.Node[] = [];
      for (const item of sectionItems) ungrouped.push({ key: item.archive_key });

      for (const [groupName, groupItems] of nestedChildren.entries()) {
        // Only include channel groups here (workspace children handled below)
        if (!groupItems.some((i) => i.type === "channel")) continue;
        const groupKey = `group:${groupName}`;
        lookup.sections.set(groupKey, "group");
        groupMap.set(
          groupKey,
          groupItems.map((i) => ({ key: i.archive_key })),
        );
      }

      children = [];
      for (const [groupKey, groupChildren] of groupMap)
        children.push({ key: groupKey, children: groupChildren });
      children.push(...ungrouped);
    } else 
      children = sectionItems.map((item) => {
        const childNodes = nestedChildren.get(item.name);
        if (childNodes != null)
          lookup.childItems.set(item.archive_key, childNodes);
        return {
          key: item.archive_key,
          children:
            item.type === "workspace" && childNodes != null
              ? childNodes.map((child) => ({ key: child.archive_key }))
              : undefined,
        };
      });
    

    sectionNodes.push({ key: sectionKey, children });
  }

  return { nodes: sectionNodes, lookup };
};

// --- Upload Step ---

interface UploadStepProps {
  onAnalyzed: (response: AnalyzeResponse) => void;
}

const UploadStep = ({ onAnalyzed }: UploadStepProps): ReactElement => {
  const client = Synnax.use();
  const cluster = Cluster.useSelect();
  const handleError = Status.useErrorHandler();
  const [loading, setLoading] = useState(false);
  const [draggingOver, setDraggingOver] = useState(false);

  const analyze = useCallback(
    async (fileBytes: Uint8Array) => {
      if (client == null || cluster == null) return;
      setLoading(true);
      try {
        const response = await analyzeBackup({ client, cluster, fileBytes });
        onAnalyzed(response);
      } catch (e) {
        handleError(e);
      } finally {
        setLoading(false);
      }
    },
    [client, cluster, handleError, onAnalyzed],
  );

  const handleFileSelect = useCallback(
    () =>
      handleError(async () => {
        const path = await open({
          directory: false,
          filters: SY_FILTERS,
          multiple: false,
        });
        if (path == null) return;
        const bytes = await readFile(path);
        await analyze(new Uint8Array(bytes));
      }, "Failed to read file"),
    [analyze, handleError],
  );

  const handleFileDrop = useCallback(
    ({ items, event }: Haul.OnDropProps): Haul.Item[] => {
      if (event == null) return items;
      event.preventDefault();
      setDraggingOver(false);
      if (event.dataTransfer.files.length === 0) return items;
      const file = event.dataTransfer.files[0];
      if (!file.name.toLowerCase().endsWith(".sy")) return items;
      handleError(async () => {
        const buffer = await file.arrayBuffer();
        await analyze(new Uint8Array(buffer));
      }, "Failed to read dropped file");
      return items;
    },
    [analyze, handleError],
  );

  const canDrop: Haul.CanDrop = useCallback(
    ({ items }) =>
      items.some((item) => item.type === Haul.FILE_TYPE) && items.length === 1,
    [],
  );

  const dropProps = Haul.useDrop({
    type: Haul.FILE_TYPE,
    onDrop: handleFileDrop,
    canDrop,
    onDragOver: () => setDraggingOver(true),
  });

  return (
    <Flex.Box
      {...dropProps}
      onDragLeave={() => setDraggingOver(false)}
      y
      grow
      justify="center"
      align="center"
      onClick={handleFileSelect}
      style={{
        cursor: "pointer",
        border: draggingOver
          ? "2px dashed var(--pluto-primary-z)"
          : "2px dashed var(--pluto-gray-l4)",
        borderRadius: "0.5rem",
        margin: "1rem",
        padding: "2rem",
      }}
    >
      {loading ? (
        <>
          <Icon.Loading style={{ fontSize: "3rem" }} />
          <Text.Text level="p" style={{ marginTop: "1rem" }}>
            Analyzing...
          </Text.Text>
        </>
      ) : (
        <>
          <Icon.Import
            style={{ fontSize: "3rem", color: "var(--pluto-gray-l7)" }}
          />
          <Text.Text level="p" style={{ marginTop: "1rem" }}>
            Drop a .sy file here or click to select
          </Text.Text>
        </>
      )}
    </Flex.Box>
  );
};

// --- Review Step ---

interface ReviewItemProps extends Tree.ItemProps<string> {
  lookup: ItemLookup;
  overrides: Record<string, ConflictPolicy>;
  defaultPolicy: ConflictPolicy;
  onOverrideChange: (archiveKey: string, policy: ConflictPolicy) => void;
}

const ReviewItem = ({
  lookup,
  overrides,
  defaultPolicy,
  onOverrideChange,
  ...props
}: ReviewItemProps): ReactElement => {
  const { itemKey } = props;

  // Section header or channel group
  const sectionType = lookup.sections.get(itemKey);
  if (sectionType != null) {
    const isGroup = sectionType === "group";
    const icon = isGroup ? <Icon.Group /> : SECTION_ICONS[sectionType];
    const label = isGroup
      ? itemKey.replace("group:", "")
      : (TYPE_LABELS[sectionType] ?? sectionType);
    return (
      <Tree.Item {...props}>
        {icon}
        <Text.Text level="small" weight={500} style={{ userSelect: "none" }}>
          {label}
        </Text.Text>
      </Tree.Item>
    );
  }

  // Resource item
  const item = lookup.items.get(itemKey);
  if (item == null) return <Tree.Item {...props} />;

  let icon: ReactElement | undefined = SECTION_ICONS[item.type];
  if (item.type === "channel" && item.data_type != null) {
    const DataTypeIcon =
      Telem.resolveDataTypeIcon(new DataType(item.data_type)) ?? Icon.Channel;
    icon = <DataTypeIcon />;
  }
  const children = lookup.childItems.get(item.archive_key);
  const effectiveStatus =
    children != null ? aggregateStatus(item, children) : item.status;
  const disabled = item.disabled === true;
  const dimmed = disabled || effectiveStatus === "identical";
  const textColor = dimmed ? "var(--pluto-gray-l5)" : undefined;
  const statusColor = disabled
    ? "var(--pluto-gray-l5)"
    : STATUS_COLORS[effectiveStatus];
  const policy = overrides[item.archive_key] ?? defaultPolicy;

  return (
    <Tree.Item {...props}>
      {icon}
      <Text.Text
        level="small"
        style={{ userSelect: "none", flexGrow: 1, width: 0, color: textColor }}
        overflow="ellipsis"
      >
        {item.name}
        {disabled && item.details != null && (
          <span style={{ color: "var(--pluto-gray-l5)", marginLeft: "0.5rem" }}>
            — {item.details}
          </span>
        )}
      </Text.Text>
      <Text.Text
        level="small"
        style={{ color: statusColor, flexShrink: 0, userSelect: "none" }}
      >
        {disabled ? "Skipped" : STATUS_LABELS[effectiveStatus]}
      </Text.Text>
      {!disabled && item.status === "conflict" && (
        <Button.Button
          variant="text"
          size="small"
          onClick={(e) => {
            e.stopPropagation();
            onOverrideChange(
              item.archive_key,
              policy === "skip" ? "replace" : "skip",
            );
          }}
          style={{
            flexShrink: 0,
            color:
              policy === "replace"
                ? "var(--pluto-warning-z)"
                : "var(--pluto-gray-l7)",
          }}
        >
          {policy === "replace" ? "Replace" : "Skip"}
        </Button.Button>
      )}
    </Tree.Item>
  );
};

interface ReviewStepProps {
  analysis: AnalyzeResponse;
  overrides: Record<string, ConflictPolicy>;
  defaultPolicy: ConflictPolicy;
  onDefaultPolicyChange: (policy: ConflictPolicy) => void;
  onOverrideChange: (archiveKey: string, policy: ConflictPolicy) => void;
}

const ReviewStep = ({
  analysis,
  overrides,
  defaultPolicy,
  onDefaultPolicyChange,
  onOverrideChange,
}: ReviewStepProps): ReactElement => {
  const { nodes, lookup } = useMemo(
    () => buildTreeNodes(analysis.items),
    [analysis.items],
  );

  const [selected, setSelected] = useState<string[]>([]);

  const treeProps = Tree.use({
    nodes,
    selected,
    onSelectedChange: setSelected,
    initialExpanded: nodes.map((n) => n.key),
  });

  const conflicts = analysis.items.filter((i) => i.status === "conflict");

  return (
    <Flex.Box y grow style={{ overflow: "hidden", minHeight: 0 }}>
      <Flex.Box x grow style={{ overflow: "hidden", minHeight: 0 }}>
        {/* Left panel: tree */}
        <Flex.Box
          y
          grow
          style={{ overflow: "auto", padding: "1rem", minWidth: 0, flex: 1 }}
        >
          <Tree.Tree {...treeProps} showRules>
            {(itemProps: Tree.ItemProps<string>) => (
              <ReviewItem
                {...itemProps}
                lookup={lookup}
                overrides={overrides}
                defaultPolicy={defaultPolicy}
                onOverrideChange={onOverrideChange}
              />
            )}
          </Tree.Tree>
        </Flex.Box>

        {/* Right panel: conflict details */}
        {conflicts.length > 0 && (
          <Flex.Box
            y
            style={{
              overflow: "auto",
              padding: "1rem",
              borderLeft: "1px solid var(--pluto-gray-l3)",
              flex: 1,
              minWidth: 0,
            }}
          >
            <Text.Text
              level="small"
              weight={500}
              style={{ marginBottom: "0.5rem" }}
            >
              Conflicts ({conflicts.length})
            </Text.Text>
            {conflicts.map((item) => (
              <Flex.Box
                key={item.archive_key}
                y
                style={{
                  padding: "0.5rem",
                  marginBottom: "0.5rem",
                  borderRadius: "0.25rem",
                  background: "var(--pluto-gray-l1)",
                }}
              >
                <Text.Text level="small" weight={450}>
                  {item.name}
                </Text.Text>
                {item.details != null && (
                  <Text.Text
                    level="small"
                    style={{ color: "var(--pluto-gray-l7)" }}
                  >
                    {item.details}
                  </Text.Text>
                )}
              </Flex.Box>
            ))}
          </Flex.Box>
        )}
      </Flex.Box>

      {/* Default policy toggle */}
      <Flex.Box
        x
        align="center"
        justify="center"
        style={{
          padding: "0.5rem 1rem",
          borderTop: "1px solid var(--pluto-gray-l3)",
          gap: "0.5rem",
        }}
      >
        <Text.Text level="small" style={{ color: "var(--pluto-gray-l7)" }}>
          Default policy:
        </Text.Text>
        <Button.Button
          variant={defaultPolicy === "skip" ? "filled" : "text"}
          size="small"
          onClick={() => onDefaultPolicyChange("skip")}
        >
          Skip All
        </Button.Button>
        <Button.Button
          variant={defaultPolicy === "replace" ? "filled" : "text"}
          size="small"
          onClick={() => onDefaultPolicyChange("replace")}
        >
          Replace All
        </Button.Button>
      </Flex.Box>
    </Flex.Box>
  );
};

// --- Results Step ---

interface ResultsStepProps {
  results: ImportResponse;
}

const ResultsStep = ({ results }: ResultsStepProps): ReactElement => (
  <Flex.Box y grow justify="center" align="center" style={{ padding: "2rem" }}>
    <Icon.Check
      style={{ fontSize: "3rem", color: "var(--pluto-secondary-z)" }}
    />
    <Text.Text level="h4" style={{ marginTop: "1rem" }}>
      Import Complete
    </Text.Text>
    <Flex.Box x style={{ gap: "2rem", marginTop: "1rem" }}>
      <Flex.Box y align="center">
        <Text.Text level="h3">{results.imported}</Text.Text>
        <Text.Text level="small" style={{ color: "var(--pluto-gray-l7)" }}>
          Imported
        </Text.Text>
      </Flex.Box>
      <Flex.Box y align="center">
        <Text.Text level="h3">{results.replaced}</Text.Text>
        <Text.Text level="small" style={{ color: "var(--pluto-gray-l7)" }}>
          Replaced
        </Text.Text>
      </Flex.Box>
      <Flex.Box y align="center">
        <Text.Text level="h3">{results.skipped}</Text.Text>
        <Text.Text level="small" style={{ color: "var(--pluto-gray-l7)" }}>
          Skipped
        </Text.Text>
      </Flex.Box>
      <Flex.Box y align="center">
        <Text.Text level="h3">{results.identical}</Text.Text>
        <Text.Text level="small" style={{ color: "var(--pluto-gray-l7)" }}>
          Identical
        </Text.Text>
      </Flex.Box>
    </Flex.Box>
    {results.errors.length > 0 && (
      <Flex.Box y style={{ marginTop: "1rem", width: "100%" }}>
        <Text.Text
          level="small"
          weight={500}
          style={{ color: "var(--pluto-error-z)", marginBottom: "0.25rem" }}
        >
          Errors:
        </Text.Text>
        {results.errors.map((err, i) => (
          <Text.Text
            key={i}
            level="small"
            style={{ color: "var(--pluto-error-z)" }}
          >
            {err}
          </Text.Text>
        ))}
      </Flex.Box>
    )}
  </Flex.Box>
);

// --- Main Modal ---

export const ImportModal = (_: Layout.RendererProps): ReactElement => {
  const client = Synnax.use();
  const cluster = Cluster.useSelect();
  const handleError = Status.useErrorHandler();

  const [step, setStep] = useState<Step>("upload");
  const [analysis, setAnalysis] = useState<AnalyzeResponse | null>(null);
  const [results, setResults] = useState<ImportResponse | null>(null);
  const [defaultPolicy, setDefaultPolicy] = useState<ConflictPolicy>("skip");
  const [overrides, setOverrides] = useState<Record<string, ConflictPolicy>>(
    {},
  );
  const [importing, setImporting] = useState(false);

  const handleAnalyzed = useCallback((response: AnalyzeResponse) => {
    setAnalysis(response);
    setStep("review");
  }, []);

  const handleOverrideChange = useCallback(
    (archiveKey: string, policy: ConflictPolicy) =>
      setOverrides((prev) => ({ ...prev, [archiveKey]: policy })),
    [],
  );

  const handleImport = useCallback(async () => {
    if (client == null || cluster == null || analysis == null) return;
    setImporting(true);
    try {
      const response = await executeImport({
        client,
        cluster,
        request: {
          session_id: analysis.session_id,
          default_policy: defaultPolicy,
          overrides,
        },
      });
      setResults(response);
      setStep("results");
    } catch (e) {
      handleError(e);
    } finally {
      setImporting(false);
    }
  }, [client, cluster, analysis, defaultPolicy, overrides, handleError]);

  return (
    <Flex.Box y style={{ height: "100%", overflow: "hidden" }}>
      {step === "upload" && <UploadStep onAnalyzed={handleAnalyzed} />}
      {step === "review" && analysis != null && (
        <ReviewStep
          analysis={analysis}
          overrides={overrides}
          defaultPolicy={defaultPolicy}
          onDefaultPolicyChange={setDefaultPolicy}
          onOverrideChange={handleOverrideChange}
        />
      )}
      {step === "results" && results != null && (
        <ResultsStep results={results} />
      )}
      {step === "review" && (
        <Modals.BottomNavBar>
          <Nav.Bar.End>
            <Button.Button
              variant="filled"
              disabled={importing || client == null || cluster == null}
              trigger={Triggers.SAVE}
              onClick={() => void handleImport()}
            >
              {importing ? "Importing..." : "Import"}
            </Button.Button>
          </Nav.Bar.End>
        </Modals.BottomNavBar>
      )}
    </Flex.Box>
  );
};
