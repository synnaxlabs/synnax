// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/vcs/Toolbar.css";

import { type vcs } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { Tabs } from "@synnaxlabs/lyra/tabs";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Synnax } from "@synnaxlabs/pluto";
import { type ReactElement, useState } from "react";

import { Changes } from "@/feature/vcs/Changes";
import { Compare } from "@/feature/vcs/Compare";
import { HeaderMenu } from "@/feature/vcs/HeaderMenu";
import { History } from "@/feature/vcs/History";
import { Review } from "@/feature/vcs/Plan";
import { useConnectModal } from "@/feature/vcs/useConnectModal";
import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { type Nav } from "@/platform/nav";
import { Toolbar } from "@/platform/toolbar";
import { Vcs } from "@/platform/vcs";

const FetchAction = (): ReactElement => {
  const client = Synnax.use();
  const setStatus = Vcs.useSetStatus();
  const handleError = Status.useErrorHandler();
  const [loading, setLoading] = useState(false);
  const handleClick = async () => {
    if (client == null) return;
    setLoading(true);
    try {
      setStatus(await client.vcs.fetch());
    } catch (e) {
      handleError(e, "Failed to fetch");
    }
    setLoading(false);
  };
  return (
    <Toolbar.Action
      tooltip="Fetch from the remote"
      aria-label="Fetch from the remote"
      onClick={() => void handleClick()}
      status={loading ? "loading" : undefined}
    >
      <Icon.Refresh />
    </Toolbar.Action>
  );
};

const Actions = (): ReactElement | null => {
  const { status } = Vcs.useStatus();
  if (status?.enabled !== true) return null;
  return (
    <Toolbar.Actions>
      {status.remoteUrl.length > 0 && <FetchAction />}
      <HeaderMenu />
    </Toolbar.Actions>
  );
};

interface SyncStateProps {
  status: vcs.Status;
}

/** How far the configuration is from the remote, in a few quiet words. */
const SyncState = ({ status }: SyncStateProps): ReactElement => {
  const { incoming, waiting, remoteUrl } = status;
  if (remoteUrl.length === 0)
    return (
      <Text.Text level="tiny" color={8} title="Commits stay on this Core">
        Local
      </Text.Text>
    );
  if (incoming.length === 0 && waiting === 0)
    return (
      <Text.Text level="tiny" color={8}>
        Up to date
      </Text.Text>
    );
  return (
    <Flex.Box x align="center" gap="small">
      {incoming.length > 0 && (
        <Text.Text
          level="tiny"
          status="warning"
          gap="tiny"
          className={CSS.BE("vcs-toolbar", "sync")}
        >
          <Icon.Arrow.Down />
          {incoming.length} behind
        </Text.Text>
      )}
      {waiting > 0 && (
        <Text.Text
          level="tiny"
          status="warning"
          gap="tiny"
          className={CSS.BE("vcs-toolbar", "sync")}
        >
          <Icon.Arrow.Up />
          {waiting} to push
        </Text.Text>
      )}
    </Flex.Box>
  );
};

interface ConfigurationBarProps {
  status: vcs.Status;
}

const ConfigurationBar = ({ status }: ConfigurationBarProps): ReactElement => {
  const { configuration, head } = status;
  return (
    <Flex.Box
      y
      gap="tiny"
      shrink={false}
      className={CSS.BE("vcs-toolbar", "configuration")}
    >
      <Flex.Box x align="center" justify="between" gap="small">
        <Text.Text weight={450} gap="small" overflow="fade">
          <Icon.Branch className={CSS.BE("vcs-toolbar", "branch-icon")} />
          {configuration}
        </Text.Text>
        <SyncState status={status} />
      </Flex.Box>
      {head != null && (
        <Text.Text
          level="small"
          color={9}
          gap="small"
          overflow="fade"
          className={CSS.BE("vcs-toolbar", "head")}
        >
          <Text.Text el="span" level="tiny" variant="code" color={8}>
            {head.shortHash}
          </Text.Text>
          {head.message}
        </Text.Text>
      )}
    </Flex.Box>
  );
};

interface TabLabelProps {
  label: string;
  count?: number;
}

const TabLabel = ({ label, count }: TabLabelProps): ReactElement => (
  <Text.Text el="span" level="small" gap="small">
    {label}
    {count != null && count > 0 && (
      <span className={CSS.BE("vcs-toolbar", "count")}>{count}</span>
    )}
  </Text.Text>
);

interface MainProps {
  status: vcs.Status;
}

const Main = ({ status }: MainProps): ReactElement => {
  const { tab } = Vcs.useUI();
  const update = Vcs.useUpdateUI();
  return (
    <Tabs.Frame
      value={tab}
      onChange={(next) =>
        update((prev) => ({ ...prev, tab: next === "history" ? "history" : "changes" }))
      }
      y
      full
      empty
    >
      <ConfigurationBar status={status} />
      <Tabs.Selector size="small" className={CSS.BE("vcs-toolbar", "tabs")}>
        <Tabs.Tab itemKey="changes">
          <TabLabel label="Changes" count={status.changes.length} />
        </Tabs.Tab>
        <Tabs.Tab itemKey="history">
          <TabLabel label="History" />
        </Tabs.Tab>
      </Tabs.Selector>
      <Tabs.Content itemKey="changes" full empty y>
        <Changes status={status} />
      </Tabs.Content>
      <Tabs.Content itemKey="history" full empty y>
        <History status={status} />
      </Tabs.Content>
    </Tabs.Frame>
  );
};

const Start = (): ReactElement => {
  const openConnect = useConnectModal();
  const canConnect = Access.useDeleteGranted(Vcs.RESOURCE_ID);
  return (
    <Flex.Box y center gap="huge" className={CSS.BE("vcs-toolbar", "start")}>
      <Flex.Box y align="center" gap="medium">
        <Icon.SourceControl className={CSS.BE("vcs-toolbar", "start-icon")} />
        <Flex.Box y align="center" gap="small">
          <Text.Text level="h5" weight={500}>
            Version control is off
          </Text.Text>
          <Text.Text level="small" color={9} className={CSS.BE("vcs", "empty-text")}>
            Track the channels, tasks, Arc programs, and visualizations of this Core in
            Git.
          </Text.Text>
        </Flex.Box>
      </Flex.Box>
      {canConnect && (
        <Button.Button variant="filled" onClick={() => openConnect()}>
          <Icon.Add />
          Start version control
        </Button.Button>
      )}
    </Flex.Box>
  );
};

const Body = (): ReactElement => {
  const { status, error } = Vcs.useStatus();
  const { view } = Vcs.useUI();
  if (status == null) {
    if (error != null)
      return (
        <Empty.Action
          message="Version control is not available on this Core"
          className={CSS.BE("vcs-toolbar", "unavailable")}
        />
      );
    return (
      <Status.Summary
        variant="loading"
        message="Loading"
        className={CSS.BE("vcs-toolbar", "loading")}
      />
    );
  }
  if (!status.enabled) return <Start />;
  if (view.variant === "plan") return <Review view={view} />;
  if (view.variant === "compare") return <Compare view={view} />;
  return <Main status={status} />;
};

const Content = (): ReactElement => (
  <Toolbar.Content>
    <Toolbar.Header>
      <Toolbar.Title>
        <Icon.SourceControl />
        Version control
      </Toolbar.Title>
      <Actions />
    </Toolbar.Header>
    <Toolbar.Body>
      <Body />
    </Toolbar.Body>
  </Toolbar.Content>
);

export const TOOLBAR: Nav.Toolbar = {
  key: Vcs.TOOLBAR_KEY,
  icon: <Icon.SourceControl />,
  content: <Content />,
  tooltip: "Version control",
  trigger: ["G"],
  initialSize: 320,
  sizeBounds: { lower: 250, upper: 520 },
  useVisible: () => Access.useRetrieveGranted(Vcs.RESOURCE_ID),
};
