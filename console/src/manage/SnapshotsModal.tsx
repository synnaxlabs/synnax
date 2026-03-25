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
import { type ReactElement } from "react";

import { Layout } from "@/layout";

export const SNAPSHOTS_LAYOUT_TYPE = "backupSnapshots";

export interface SnapshotsArgs {
  policyKey: string;
  policyName: string;
}

export const createSnapshotsLayout = (
  policyKey: string,
  policyName: string,
): Layout.BaseState<SnapshotsArgs> => ({
  key: `${SNAPSHOTS_LAYOUT_TYPE}-${policyKey}`,
  type: SNAPSHOTS_LAYOUT_TYPE,
  name: `${policyName} · Snapshots`,
  icon: "Menu",
  location: "modal",
  window: { resizable: true, size: { height: 450, width: 500 }, navTop: true },
  args: { policyKey, policyName },
});

interface Snapshot {
  key: string;
  timestamp: string;
  type: string;
  size: string;
  statusVariant: status.Variant;
  statusLabel: string;
}

const STATUS_COLORS: Partial<Record<status.Variant, string>> = {
  success: "var(--pluto-primary-z)",
  error: "var(--pluto-error-z)",
  warning: "var(--pluto-warning-m1)",
  loading: "var(--pluto-warning-m1)",
  disabled: "var(--pluto-gray-l6)",
};

const statusColor = (variant: status.Variant): string =>
  STATUS_COLORS[variant] ?? "var(--pluto-gray-l7)";

const MOCK_SNAPSHOTS: Record<string, Snapshot[]> = {
  nightly: [
    {
      key: "s1",
      timestamp: "2026-03-24 08:00 UTC",
      type: "Data + Metadata",
      size: "1.2 GB",
      statusVariant: "success",
      statusLabel: "Complete",
    },
    {
      key: "s2",
      timestamp: "2026-03-24 02:00 UTC",
      type: "Data + Metadata",
      size: "1.1 GB",
      statusVariant: "success",
      statusLabel: "Complete",
    },
    {
      key: "s3",
      timestamp: "2026-03-23 20:00 UTC",
      type: "Data + Metadata",
      size: "1.3 GB",
      statusVariant: "success",
      statusLabel: "Complete",
    },
    {
      key: "s4",
      timestamp: "2026-03-23 14:00 UTC",
      type: "Data + Metadata",
      size: "980 MB",
      statusVariant: "error",
      statusLabel: "Corrupt",
    },
  ],
  manual: [
    {
      key: "s1",
      timestamp: "2026-03-21 14:30 UTC",
      type: "Metadata",
      size: "24 MB",
      statusVariant: "success",
      statusLabel: "Complete",
    },
    {
      key: "s2",
      timestamp: "2026-03-18 09:15 UTC",
      type: "Metadata",
      size: "23 MB",
      statusVariant: "success",
      statusLabel: "Complete",
    },
    {
      key: "s3",
      timestamp: "2026-03-14 16:45 UTC",
      type: "Metadata",
      size: "22 MB",
      statusVariant: "success",
      statusLabel: "Complete",
    },
    {
      key: "s4",
      timestamp: "2026-03-10 11:00 UTC",
      type: "Metadata",
      size: "21 MB",
      statusVariant: "success",
      statusLabel: "Complete",
    },
    {
      key: "s5",
      timestamp: "2026-03-07 08:20 UTC",
      type: "Metadata",
      size: "20 MB",
      statusVariant: "error",
      statusLabel: "Corrupt",
    },
  ],
  arc: [],
};

const GRID = "3fr 1.5fr 2fr 2rem 2rem";

export const SnapshotsModal = ({
  layoutKey,
}: Layout.RendererProps): ReactElement => {
  const args = Layout.useSelectArgs<SnapshotsArgs>(layoutKey);
  const snapshots = MOCK_SNAPSHOTS[args?.policyKey ?? ""] ?? [];

  return (
    <Flex.Box y grow style={{ overflow: "auto" }}>
      <div
        style={{
          display: "grid",
          gridTemplateColumns: GRID,
          alignItems: "center",
          padding: "0.75rem 2.5rem 0.75rem 1.5rem",
          borderBottom: "var(--pluto-border)",
          gap: "0.5rem",
        }}
      >
        {["Datetime", "Size", "Status", "", ""].map((h, i) => (
          <Text.Text
            key={i}
            level="small"
            weight={500}
            style={{ color: "var(--pluto-gray-l7)" }}
          >
            {h}
          </Text.Text>
        ))}
      </div>
      {snapshots.length === 0 ? (
        <Flex.Box y grow justify="center" align="center" style={{ padding: "2rem" }}>
          <Text.Text level="p" style={{ color: "var(--pluto-gray-l5)" }}>
            No snapshots yet
          </Text.Text>
        </Flex.Box>
      ) : (
        snapshots.map((snap) => (
          <div
            key={snap.key}
            style={{
              display: "grid",
              gridTemplateColumns: GRID,
              alignItems: "center",
              padding: "0.6rem 2.5rem 0.6rem 1.5rem",
              borderBottom: "var(--pluto-border)",
              gap: "0.5rem",
            }}
          >
            <Text.Text
              level="small"
              weight={450}
              style={{ color: "var(--pluto-gray-l11)" }}
            >
              {snap.timestamp}
            </Text.Text>
            <Text.Text level="small" style={{ color: "var(--pluto-gray-l8)" }}>
              {snap.size}
            </Text.Text>
            <Flex.Box x align="center" style={{ gap: "0.4rem" }}>
              <Status.Indicator variant={snap.statusVariant} />
              <Text.Text
                level="small"
                weight={450}
                style={{ color: statusColor(snap.statusVariant) }}
              >
                {snap.statusLabel}
              </Text.Text>
            </Flex.Box>
            <Button.Button
              variant="text"
              size="small"
              style={{ padding: 0, minWidth: 0, color: "var(--pluto-gray-l7)" }}
              tooltip="Compare with current"
            >
              <Icon.SplitX />
            </Button.Button>
            <Button.Button
              variant="text"
              size="small"
              style={{ padding: 0, minWidth: 0, color: "var(--pluto-gray-l7)" }}
              tooltip="Restore this snapshot"
            >
              <Icon.Refresh />
            </Button.Button>
          </div>
        ))
      )}
    </Flex.Box>
  );
};
