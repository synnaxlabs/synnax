// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Flex, Icon, Status, Text } from "@synnaxlabs/pluto";
import { type status } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useCallback, useState } from "react";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import "@/manage/BackupsTab.css";
import { createSnapshotsLayout } from "@/manage/SnapshotsModal";

// --- Shared constants & utilities ---

const STATUS_COLORS: Partial<Record<status.Variant, string>> = {
  success: "var(--pluto-primary-z)",
  error: "var(--pluto-error-z)",
  warning: "var(--pluto-warning-m1)",
  loading: "var(--pluto-warning-m1)",
  disabled: "var(--pluto-gray-l6)",
};

const statusColor = (variant: status.Variant): string =>
  STATUS_COLORS[variant] ?? "var(--pluto-gray-l7)";

// --- Shared hooks ---

const useExpandSet = (): [Set<string>, (key: string) => void] => {
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const toggle = useCallback(
    (key: string) =>
      setExpanded((prev) => {
        const next = new Set(prev);
        if (next.has(key)) next.delete(key);
        else next.add(key);
        return next;
      }),
    [],
  );
  return [expanded, toggle];
};

// --- Shared components ---

const HeaderRow = (): ReactElement => (
  <div className={CSS(CSS.BE("backups", "grid-row"), CSS.BE("backups", "header"))}>
    {["Name", "", "Status", ""].map((h, i) => (
      <Text.Text
        key={i}
        level="small"
        weight={500}
        className={CSS.BE("backups", "header-text")}
      >
        {h}
      </Text.Text>
    ))}
  </div>
);

interface RuleRowProps {
  name: string;
  statusVariant: status.Variant;
  statusLabel: string;
  expanded: boolean;
  onToggle: () => void;
  children?: ReactNode;
}

const RuleRow = ({
  name,
  statusVariant,
  statusLabel,
  expanded,
  onToggle,
  children,
}: RuleRowProps): ReactElement => (
  <div>
    <div
      className={CSS(
        CSS.BE("backups", "grid-row"),
        CSS.BE("backups", "row"),
        !expanded && CSS.BEM("backups", "row", "collapsed"),
      )}
      onClick={onToggle}
    >
      <Text.Text
        level="p"
        weight={450}
        className={CSS.BE("backups", "row-name")}
        overflow="ellipsis"
      >
        {name}
      </Text.Text>
      <div />
      <Flex.Box x align="center" style={{ gap: "0.4rem", minWidth: 0 }}>
        <Status.Indicator variant={statusVariant} />
        <Text.Text
          level="small"
          weight={450}
          style={{ color: statusColor(statusVariant) }}
          overflow="ellipsis"
        >
          {statusLabel}
        </Text.Text>
      </Flex.Box>
      <Button.Button
        variant="text"
        size="small"
        onClick={(e) => {
          e.stopPropagation();
          onToggle();
        }}
        className={CSS.BE("backups", "icon-btn")}
      >
        <Icon.Settings />
      </Button.Button>
    </div>
    {expanded && children}
  </div>
);

const DetailPanel = ({
  children,
  editTooltip = "Edit",
}: {
  children: ReactNode;
  editTooltip?: string;
}): ReactElement => (
  <Flex.Box y className={CSS.BE("backups", "detail-panel")}>
    <Flex.Box x align="center" justify="between">
      <Text.Text
        level="small"
        weight={500}
        style={{ color: "var(--pluto-gray-l9)" }}
      >
        Details
      </Text.Text>
      <Button.Button
        variant="text"
        size="small"
        className={CSS.BE("backups", "icon-btn")}
        tooltip={editTooltip}
      >
        <Icon.Edit />
      </Button.Button>
    </Flex.Box>
    {children}
  </Flex.Box>
);

const DetailField = ({ label, value }: { label: string; value: string }): ReactElement => (
  <Flex.Box x align="center" style={{ gap: "0.5rem" }}>
    <Text.Text
      level="small"
      weight={500}
      className={CSS.BE("backups", "detail-label")}
    >
      {label}:
    </Text.Text>
    <Text.Text level="small" className={CSS.BE("backups", "detail-value")}>
      {value}
    </Text.Text>
  </Flex.Box>
);

const AddButton = ({ label }: { label: string }): ReactElement => (
  <Flex.Box x align="center" justify="center" className={CSS.BE("backups", "add-btn")}>
    <Button.Button
      variant="text"
      size="small"
      style={{ color: "var(--pluto-gray-l6)" }}
    >
      <Icon.Add style={{ marginRight: "0.35rem" }} />
      {label}
    </Button.Button>
  </Flex.Box>
);

// --- Snapshots data & section ---

interface BackupPolicy {
  key: string;
  name: string;
  destination: string;
  destinationDetail: string;
  type: string;
  trigger: string;
  triggerDetail: string;
  description?: string;
  statusVariant: status.Variant;
  statusLabel: string;
  lastRun?: string;
  nextRun?: string;
}

const POLICIES: BackupPolicy[] = [
  {
    key: "nightly",
    name: "Nightly Full",
    destination: "S3",
    destinationDetail: "s3://prod-backups/synnax/",
    type: "Data + Metadata",
    trigger: "Scheduled",
    triggerDetail: "Every 6h",
    statusVariant: "success",
    statusLabel: "Last: 03/25 02:00 UTC · 2h ago",
    lastRun: "2026-03-25 02:00 UTC",
    nextRun: "2026-03-26 02:00 UTC",
  },
  {
    key: "manual",
    name: "Task Configs",
    destination: "Local Disk",
    destinationDetail: "/var/backups/synnax/",
    type: "Metadata",
    trigger: "Manual",
    triggerDetail: "On demand",
    statusVariant: "success",
    statusLabel: "Last: 03/21 14:30 UTC · 3d ago",
    lastRun: "2026-03-21 14:30 UTC",
  },
  {
    key: "workspaces",
    name: "Workspace Layouts",
    destination: "Git",
    destinationDetail: "git@github.com:org/synnax-backups.git",
    type: "Metadata",
    trigger: "Scheduled",
    triggerDetail: "Daily at 00:00",
    statusVariant: "success",
    statusLabel: "Last: 03/25 00:00 UTC · 4h ago",
    lastRun: "2026-03-25 00:00 UTC",
    nextRun: "2026-03-26 00:00 UTC",
  },
  {
    key: "full-weekly",
    name: "Weekly Full Backup",
    destination: "S3",
    destinationDetail: "s3://prod-backups/weekly/",
    type: "Data + Metadata",
    trigger: "Scheduled",
    triggerDetail: "Weekly on Sunday",
    statusVariant: "success",
    statusLabel: "Last: 03/23 02:00 UTC · 2d ago",
    lastRun: "2026-03-23 02:00 UTC",
    nextRun: "2026-03-30 02:00 UTC",
  },
  {
    key: "arc",
    name: "Arc Triggered",
    destination: "SFTP",
    destinationDetail: "sftp://backup-host.local/synnax/",
    type: "Data",
    trigger: "Event-Driven (Arc)",
    triggerDetail: "Arc automation",
    description: "Triggers a backup when a range is closed",
    statusVariant: "disabled",
    statusLabel: "Idle",
  },
];

const SnapshotsSection = (): ReactElement => {
  const [expanded, toggle] = useExpandSet();
  const placer = Layout.usePlacer();

  return (
    <Flex.Box y grow style={{ overflow: "auto" }}>
      <div className="console-backups__description">
        <Text.Text level="small" style={{ color: "var(--pluto-gray-l6)" }}>
        Point-in-time backups of configuration, metadata, and data.
      </Text.Text>
      </div>
      <HeaderRow />
      {POLICIES.map((p) => (
        <RuleRow
          key={p.key}
          name={p.name}
          statusVariant={p.statusVariant}
          statusLabel={p.statusLabel}
          expanded={expanded.has(p.key)}
          onToggle={() => toggle(p.key)}
        >
          <DetailPanel editTooltip="Edit policy">
            <DetailField
              label="Destination"
              value={`${p.destination} (${p.destinationDetail})`}
            />
            <DetailField label="Type" value={p.type} />
            <DetailField label="Trigger" value={p.trigger} />
            {p.description != null && (
              <DetailField label="Description" value={p.description} />
            )}
            {p.lastRun != null && <DetailField label="Last Run" value={p.lastRun} />}
            {p.nextRun != null && <DetailField label="Next Run" value={p.nextRun} />}
            <Flex.Box x style={{ gap: "0.5rem", marginTop: "0.5rem" }}>
              <Button.Button
                variant="outlined"
                size="small"
                onClick={(e) => {
                  e.stopPropagation();
                  placer(createSnapshotsLayout(p.key, p.name));
                }}
              >
                <Icon.Menu style={{ marginRight: "0.35rem" }} />
                View Snapshots
              </Button.Button>
              <Button.Button variant="outlined" size="small">
                <Icon.Play style={{ marginRight: "0.35rem" }} />
                Run Now
              </Button.Button>
            </Flex.Box>
          </DetailPanel>
        </RuleRow>
      ))}
      <AddButton label="Add Rule" />
    </Flex.Box>
  );
};

// --- Cold Storage data & section ---

interface ColdStorageRule {
  key: string;
  name: string;
  destination: string;
  destinationDetail: string;
  retention: string;
  decimation: string;
  statusVariant: status.Variant;
  statusLabel: string;
}

const COLD_RULES: ColdStorageRule[] = [
  {
    key: "historical",
    name: "Historical Telemetry",
    destination: "S3",
    destinationDetail: "s3://cold-archive/telemetry/",
    retention: "Offload after 30d",
    decimation: "10x downsample",
    statusVariant: "success",
    statusLabel: "Up to date · 847 GB offloaded",
  },
  {
    key: "raw-archive",
    name: "Raw Sensor Archive",
    destination: "Network Drive",
    destinationDetail: "//nas/synnax-archive/raw/",
    retention: "Offload after 7d",
    decimation: "None (full resolution)",
    statusVariant: "loading",
    statusLabel: "Syncing · 2.3 GB remaining",
  },
  {
    key: "peak-preserve",
    name: "Peak Preservation Archive",
    destination: "S3",
    destinationDetail: "s3://cold-archive/peaks/",
    retention: "Permanent",
    decimation: "5x downsample · min/max peak preservation",
    statusVariant: "success",
    statusLabel: "Up to date · 1.2 TB offloaded",
  },
];

const ColdStorageSection = (): ReactElement => {
  const [expanded, toggle] = useExpandSet();

  return (
    <Flex.Box y grow style={{ overflow: "auto" }}>
      <div className="console-backups__description">
        <Text.Text level="small" style={{ color: "var(--pluto-gray-l6)" }}>
        Offload historical time-series data to queryable remote storage.
      </Text.Text>
      </div>
      <HeaderRow />
      {COLD_RULES.map((r) => (
        <RuleRow
          key={r.key}
          name={r.name}
          statusVariant={r.statusVariant}
          statusLabel={r.statusLabel}
          expanded={expanded.has(r.key)}
          onToggle={() => toggle(r.key)}
        >
          <DetailPanel editTooltip="Edit rule">
            <DetailField
              label="Destination"
              value={`${r.destination} (${r.destinationDetail})`}
            />
            <DetailField label="Retention" value={r.retention} />
            <DetailField label="Decimation" value={r.decimation} />
          </DetailPanel>
        </RuleRow>
      ))}
      <AddButton label="Add Rule" />
    </Flex.Box>
  );
};

// --- Tab selector & main export ---

type ArchiveTab = "snapshots" | "cold";

const TAB_OPTIONS: { key: ArchiveTab; label: string }[] = [
  { key: "snapshots", label: "Snapshots" },
  { key: "cold", label: "Cold Storage" },
];

const TabSelector = ({
  value,
  onChange,
}: {
  value: ArchiveTab;
  onChange: (v: ArchiveTab) => void;
}): ReactElement => (
  <Flex.Box x justify="center" className={CSS.BE("backups", "tab-bar")}>
    {TAB_OPTIONS.map((tab) => {
      const active = value === tab.key;
      return (
        <Button.Button
          key={tab.key}
          variant="text"
          onClick={() => onChange(tab.key)}
          className={CSS(
            CSS.BE("backups", "tab"),
            active
              ? CSS.BEM("backups", "tab", "active")
              : CSS.BEM("backups", "tab", "inactive"),
          )}
        >
          {tab.label}
        </Button.Button>
      );
    })}
  </Flex.Box>
);

export const BackupsTab = (): ReactElement => {
  const [tab, setTab] = useState<ArchiveTab>("snapshots");

  return (
    <Flex.Box y grow style={{ overflow: "hidden" }}>
      <TabSelector value={tab} onChange={setTab} />
      {tab === "snapshots" ? <SnapshotsSection /> : <ColdStorageSection />}
    </Flex.Box>
  );
};
