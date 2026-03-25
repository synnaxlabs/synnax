// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { user } from "@synnaxlabs/client";
import { Access, Button, Flex, Icon, Text, User } from "@synnaxlabs/pluto";
import { type ReactElement, type ReactNode, useState } from "react";

import { ExportContent } from "@/export/ExportModal";
import { ImportContent } from "@/import/ImportModal";
import { Layout } from "@/layout";
import { BackupsTab } from "@/manage/BackupsTab";
import { CoresTab } from "@/manage/CoresTab";
import { Ontology } from "@/ontology";
import { REGISTER_LAYOUT } from "@/user/Register";


export const MANAGE_CORE_LAYOUT_TYPE = "manageCore";

export interface ManageCoreArgs {
  initialTab?: string;
}

export const MANAGE_CORE_LAYOUT: Layout.BaseState<ManageCoreArgs> = {
  key: MANAGE_CORE_LAYOUT_TYPE,
  type: MANAGE_CORE_LAYOUT_TYPE,
  name: "Manage Core and Data",
  icon: "Settings",
  location: "modal",
  window: { resizable: true, size: { height: 650, width: 800 }, navTop: true },
};

export const createManageCoreLayout = (
  initialTab?: string,
): Layout.BaseState<ManageCoreArgs> => ({
  ...MANAGE_CORE_LAYOUT,
  args: initialTab != null ? { initialTab } : undefined,
});

interface TabDef {
  key: string;
  label: string;
}

const TABS: TabDef[] = [
  { key: "export", label: "Export" },
  { key: "import", label: "Import" },
  { key: "backups", label: "Archive Policies" },
  { key: "delete", label: "Delete Data" },
  { key: "users", label: "Users" },
  { key: "cores", label: "Manage Cores" },
];

const Placeholder = ({ label }: { label: string }): ReactElement => (
  <Flex.Box y grow justify="center" align="center">
    <Text.Text level="p" style={{ color: "var(--pluto-gray-l5)" }}>
      {label}
    </Text.Text>
  </Flex.Box>
);

const UsersTab = (): ReactElement => {
  const { data: groupID } = User.useRetrieveGroupID({});
  const placer = Layout.usePlacer();
  const canCreate = Access.useCreateGranted(user.TYPE_ONTOLOGY_ID);
  return (
    <Flex.Box y grow style={{ overflow: "hidden" }}>
      <Flex.Box
        x
        align="center"
        justify="between"
        style={{ padding: "0.75rem 1rem", borderBottom: "var(--pluto-border)" }}
      >
        <Text.Text level="p" weight={500}>
          Users
        </Text.Text>
        {canCreate && (
          <Button.Button
            variant="text"
            size="small"
            onClick={() => placer(REGISTER_LAYOUT)}
            tooltip="Register a user"
          >
            <Icon.Add />
          </Button.Button>
        )}
      </Flex.Box>
      <Flex.Box y grow style={{ overflow: "auto" }}>
        <Ontology.Tree root={groupID} />
      </Flex.Box>
    </Flex.Box>
  );
};

const renderContent = (key: string): ReactNode => {
  switch (key) {
    case "export":
      return <ExportContent />;
    case "import":
      return <ImportContent />;
    case "backups":
      return <BackupsTab />;
    case "cores":
      return <CoresTab />;
    case "delete":
      return <Placeholder label="Delete Data" />;
    case "users":
      return <UsersTab />;
    default:
      return null;
  }
};

export const ManageCoreModal = ({ layoutKey }: Layout.RendererProps): ReactElement => {
  const args = Layout.useSelectArgs<ManageCoreArgs>(layoutKey);
  const [selected, setSelected] = useState(args?.initialTab ?? "export");
  return (
    <Flex.Box x style={{ height: "100%", overflow: "hidden" }}>
      <Flex.Box
        y
        style={{
          borderRight: "var(--pluto-border)",
          padding: "0.5rem 0",
          minWidth: "12rem",
        }}
      >
        {TABS.map((tab) => {
          const active = selected === tab.key;
          return (
            <Button.Button
              key={tab.key}
              variant="text"
              sharp
              onClick={() => setSelected(tab.key)}
              style={{
                justifyContent: "flex-start",
                padding: "0.5rem 1.5rem",
                width: "100%",
                background: active ? "var(--pluto-gray-l2)" : undefined,
                borderRight: `2px solid ${active ? "var(--pluto-primary-z)" : "transparent"}`,
              }}
            >
              <Text.Text
                level="p"
                weight={active ? 500 : 400}
                style={{
                  color: active
                    ? "var(--pluto-gray-l11)"
                    : "var(--pluto-gray-l9)",
                }}
              >
                {tab.label}
              </Text.Text>
            </Button.Button>
          );
        })}
      </Flex.Box>
      <Flex.Box y grow style={{ overflow: "hidden", minWidth: 0 }}>
        {renderContent(selected)}
      </Flex.Box>
    </Flex.Box>
  );
};
