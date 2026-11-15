// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/credential/KeyTable.css";

import { type auth, type resource, TimeStamp } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Credential } from "@synnaxlabs/pluto";
import { createContext, type ReactElement, use } from "react";

import { ContextMenu } from "@/platform/context-menu";
import { useKeyActions, type UseKeyActionsReturn } from "@/platform/credential/Keys";
import { useCreateKeyModal } from "@/platform/credential/useCreateKeyModal";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Settings } from "@/platform/settings";

const COLUMNS: List.Column[] = [
  { name: "Name", width: "1fr" },
  { name: "Created", width: "34rem" },
  { name: "Live sessions", width: "14rem" },
];

const keyName = (k: auth.APIKey): string => (k.name === "" ? "Unnamed key" : k.name);

interface KeyItemProps extends List.ItemProps<auth.APIKeyKey> {
  sessions: number | null;
}

const KeyItem = ({ sessions, ...rest }: KeyItemProps): ReactElement | null => {
  const k = List.useItem<auth.APIKeyKey, auth.APIKey>(rest.itemKey);
  const fresh = Settings.useFreshClass(rest.itemKey);
  if (k == null) return null;
  return (
    <Select.Item<auth.APIKeyKey>
      {...rest}
      checkable
      align="center"
      aria-label={keyName(k)}
      className={fresh}
    >
      <List.Cells>
        <List.Cell>
          <Text.Text overflow="fade">{keyName(k)}</Text.Text>
        </List.Cell>
        <List.Cell>
          <Text.Text color={9} overflow="fade">
            {new TimeStamp(k.created).toString("dateTime", "local")}
          </Text.Text>
        </List.Cell>
        <List.Cell>
          <Text.Text
            color={sessions == null || sessions === 0 ? 9 : undefined}
            className={CSS.BE("key-table", "count")}
          >
            {sessions ?? "Unknown"}
          </Text.Text>
        </List.Cell>
      </List.Cells>
    </Select.Item>
  );
};

const ActionsContext = createContext<Pick<
  UseKeyActionsReturn,
  "canDelete" | "remove"
> | null>(null);
ActionsContext.displayName = "Credential.KeyActions";

const KeysMenu = ({ keys }: Menu.ContextMenuMenuProps): ReactElement => {
  const actions = use(ActionsContext);
  const { getItem } = List.useUtilContext<auth.APIKeyKey, auth.APIKey>();
  const items = (getItem?.(keys) ?? []).map((k) => ({ ...k, name: keyName(k) }));
  const confirm = Modals.useConfirmDelete({
    type: "API key",
    plural: "API keys",
    icon: "APIKey",
  });
  return (
    <ContextMenu.Menu>
      {items.length > 0 && actions?.canDelete === true && (
        <ContextMenu.DeleteItem
          onClick={() =>
            void confirm(items).then((ok) => {
              if (ok) actions.remove(items.map((i) => i.key));
            })
          }
        />
      )}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const menu = Component.renderProp(KeysMenu);

export interface KeyTableProps {
  /** The user whose API keys to manage. */
  subject: resource.ID;
}

/** The subject's API keys on the columns of a Settings page, with live sessions. */
export const KeyTable = ({ subject }: KeyTableProps): ReactElement => {
  const actions = useKeyActions(subject);
  const openCreate = useCreateKeyModal();
  const keys = Credential.useResultKeys({ subject });
  const sessions = Credential.useResultSessions({ subject });
  const list = List.useStaticData<auth.APIKeyKey, auth.APIKey>({
    data: keys.data ?? [],
  });
  const sessionCount = (key: auth.APIKeyKey): number | null =>
    sessions.variant === "success"
      ? sessions.data.filter((s) => s.credential === key).length
      : null;
  return (
    <ActionsContext value={actions}>
      <Settings.Table<auth.APIKeyKey, auth.APIKey>
        list={{ ...list, answered: keys.variant !== "loading" }}
        columns={COLUMNS}
        noun="API keys"
        menu={menu}
        createLabel={actions.canCreate ? "Create key" : undefined}
        onCreate={async () => await openCreate({ subject })}
      >
        {({ key, ...p }) => <KeyItem key={key} {...p} sessions={sessionCount(key)} />}
      </Settings.Table>
    </ActionsContext>
  );
};
