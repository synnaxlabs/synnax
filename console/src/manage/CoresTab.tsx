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

interface CoreEntry {
  name: string;
  ip: string;
  variant: status.Variant;
  label: string;
}

const STATUS_COLORS: Partial<Record<status.Variant, string>> = {
  success: "var(--pluto-primary-z)",
  error: "var(--pluto-error-z)",
  warning: "var(--pluto-warning-m1)",
  loading: "var(--pluto-warning-m1)",
  disabled: "var(--pluto-gray-l6)",
};

const statusColor = (entry: CoreEntry): string =>
  STATUS_COLORS[entry.variant] ?? "var(--pluto-gray-l7)";

const CONSOLE_CONNECTION: CoreEntry = {
  name: "Repeater West",
  ip: "10.0.1.12:9090",
  variant: "success",
  label: "",
};

const PEERS: CoreEntry[] = [
  { name: "Primary", ip: "10.0.1.10:9090", variant: "success", label: "" },
  { name: "Backup 1", ip: "10.0.1.11:9090", variant: "loading", label: "Syncing" },
  { name: "Repeater East", ip: "10.0.1.13:9090", variant: "success", label: "" },
];

const FORWARDING_TO: CoreEntry[] = [
  { name: "HQ 1", ip: "203.0.113.10:9090", variant: "success", label: "" },
  { name: "HQ 2", ip: "203.0.113.11:9090", variant: "disabled", label: "Offline" },
];

const StatusLabel = ({ core }: { core: CoreEntry }): ReactElement => (
  <Flex.Box x align="center" style={{ gap: "0.5rem", flexShrink: 0 }}>
    {core.label !== "" && (
      <Text.Text level="small" weight={450} style={{ color: statusColor(core) }}>
        {core.label}
      </Text.Text>
    )}
    <Status.Indicator variant={core.variant} />
  </Flex.Box>
);

const CoreRow = ({ core }: { core: CoreEntry }): ReactElement => (
  <Flex.Box
    x
    align="center"
    justify="between"
    style={{ padding: "0.5rem 0.25rem" }}
  >
    <Flex.Box y style={{ gap: "0.1rem", minWidth: 0 }}>
      <Text.Text level="p" weight={450} style={{ color: "var(--pluto-gray-l11)" }}>
        {core.name}
      </Text.Text>
      <Text.Text level="small" style={{ color: "var(--pluto-gray-l6)" }}>
        {core.ip}
      </Text.Text>
    </Flex.Box>
    <StatusLabel core={core} />
  </Flex.Box>
);

const SectionLabel = ({ children }: { children: string }): ReactElement => (
  <Text.Text
    level="small"
    weight={500}
    style={{ color: "var(--pluto-gray-l7)", padding: "0.25rem 0" }}
  >
    {children}
  </Text.Text>
);

const Divider = (): ReactElement => (
  <div style={{ borderBottom: "var(--pluto-border)", marginTop: "1rem", marginBottom: "0rem" }} />
);

export const CoresTab = (): ReactElement => (
  <Flex.Box
    y
    grow
    style={{ overflow: "auto", padding: "1.25rem 10rem" }}
  >
    <SectionLabel>Console Connection</SectionLabel>
    <Flex.Box
      x
      align="center"
      justify="between"
      style={{
        padding: "0.75rem",
        borderRadius: "0.5rem",
        background: "var(--pluto-gray-l1)",
        marginBottom: "0.25rem",
      }}
    >
      <Flex.Box y style={{ gap: "0.1rem", minWidth: 0 }}>
        <Text.Text level="p" weight={500} style={{ color: "var(--pluto-gray-l11)" }}>
          {CONSOLE_CONNECTION.name}
        </Text.Text>
        <Text.Text level="small" style={{ color: "var(--pluto-gray-l6)" }}>
          {CONSOLE_CONNECTION.ip}
        </Text.Text>
      </Flex.Box>
      <StatusLabel core={CONSOLE_CONNECTION} />
    </Flex.Box>
    <Divider />
    <SectionLabel>Peers</SectionLabel>
    {PEERS.map((core) => (
      <CoreRow key={core.ip} core={core} />
    ))}
    <Divider />
    <SectionLabel>Forwarding To</SectionLabel>
    {FORWARDING_TO.map((core) => (
      <CoreRow key={core.ip} core={core} />
    ))}
    <Divider />
    <SectionLabel>Receiving From</SectionLabel>
    <Flex.Box
      x
      align="center"
      justify="center"
      style={{ padding: "0.75rem 0" }}
    >
      <Button.Button
        variant="text"
        size="small"
        style={{ color: "var(--pluto-gray-l6)" }}
      >
        <Icon.Add style={{ marginRight: "0.35rem" }} />
        Configure incoming connection
      </Button.Button>
    </Flex.Box>
  </Flex.Box>
);
