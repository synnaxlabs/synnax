// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/vcs/useConfigurationModal.css";

import { type vcs } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Status } from "@synnaxlabs/lyra/status";
import { Telem } from "@synnaxlabs/lyra/telem";
import { Text } from "@synnaxlabs/lyra/text";
import { Synnax } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useState } from "react";

import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";

export interface ConfigurationModalParams {
  title: string;
  icon: Icon.ReactElement;
  /** Says what picking a configuration does. */
  description: string;
}

interface ItemProps {
  configuration: vcs.Configuration;
  onSelect: (name: string) => void;
}

const Item = ({
  configuration: { name, head, current },
  onSelect,
}: ItemProps): ReactElement => (
  <List.Item
    itemKey={name}
    className={CSS.BE("vcs-configurations", "item")}
    y
    gap="tiny"
    disabled={current}
    onClick={() => onSelect(name)}
  >
    <Flex.Box x align="center" gap="small">
      <Icon.Branch className={CSS.BE("vcs-configurations", "icon")} />
      <Text.Text weight={450} overflow="fade" grow>
        {name}
      </Text.Text>
      {current ? (
        <Text.Text level="small" color={8}>
          Current
        </Text.Text>
      ) : (
        <Telem.Text.TimeSpanSince level="small" format="semantic" color={8}>
          {head.time}
        </Telem.Text.TimeSpanSince>
      )}
    </Flex.Box>
    <Text.Text
      level="small"
      color={9}
      gap="small"
      overflow="fade"
      className={CSS.BE("vcs-configurations", "head")}
    >
      <Text.Text el="span" level="tiny" variant="code" color={8}>
        {head.shortHash}
      </Text.Text>
      {head.message}
    </Text.Text>
  </List.Item>
);

const Picker = ({
  title,
  icon,
  description,
  close,
}: Modals.ContentProps<ConfigurationModalParams, string>) => {
  const client = Synnax.use();
  const [configurations, setConfigurations] = useState<vcs.Configuration[] | null>(
    null,
  );
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    if (client == null) return;
    client.vcs
      .configurations()
      .then(setConfigurations)
      .catch((e: unknown) => setError(errors.fromUnknown(e).message));
  }, [client]);
  const others = configurations?.filter(({ current }) => !current) ?? [];
  let content: ReactElement;
  if (error != null)
    content = (
      <Status.Summary
        variant="error"
        center={false}
        message="Failed to load configurations"
        description={error}
      />
    );
  else if (configurations == null)
    content = <Status.Summary variant="loading" message="Loading configurations" />;
  else if (others.length === 0)
    content = (
      <Flex.Box y gap="tiny" className={CSS.BE("vcs-configurations", "empty")}>
        <Text.Text status="disabled">No other configurations</Text.Text>
        <Text.Text level="small" color={8}>
          Create one with Clone configuration in the version control menu.
        </Text.Text>
      </Flex.Box>
    );
  else
    content = (
      <Flex.Box y empty className={CSS.BE("vcs-configurations", "list")}>
        {configurations.map((c) => (
          <Item key={c.name} configuration={c} onSelect={(name) => close(name)} />
        ))}
      </Flex.Box>
    );
  return (
    <Modal.Frame>
      <Modal.Header icon={icon}>{title}</Modal.Header>
      <Modal.Body gap="medium" align="stretch">
        <Text.Text level="small" color={9}>
          {description}
        </Text.Text>
        {content}
      </Modal.Body>
    </Modal.Frame>
  );
};

/** Opens a dialog that picks a configuration. Resolves with its name, or null. */
export const useConfigurationModal = Modals.createPrompt<
  string,
  ConfigurationModalParams
>(Picker);
