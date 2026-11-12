// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/secret/Toolbar.css";

import { secret } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Access } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Lists } from "@/feature/secret/Lists";
import { useCreateModal } from "@/feature/secret/useCreateModal";
import { useRetrievable } from "@/feature/secret/useLists";
import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { type Nav } from "@/platform/nav";
import { Toolbar } from "@/platform/toolbar";

const EmptyContent = (): ReactElement => {
  const openCreate = useCreateModal();
  const canCreate = Access.useCreateGranted(secret.TYPE_RESOURCE_ID);
  return (
    <Empty.Action
      message="No credentials"
      action={canCreate ? "Create secret" : undefined}
      onClick={() => openCreate()}
    />
  );
};

const EMPTY_CONTENT = <EmptyContent />;

const Content = (): ReactElement => {
  const openCreate = useCreateModal();
  const canCreate = Access.useCreateGranted(secret.TYPE_RESOURCE_ID);
  return (
    <Toolbar.Content className={CSS.B("credentials")}>
      <Toolbar.Header>
        <Toolbar.Title>
          <Icon.Secret />
          Credentials
        </Toolbar.Title>
        {canCreate && (
          <Toolbar.Actions>
            <Toolbar.CreateAction
              tooltip="Create secret"
              onClick={() => openCreate()}
            />
          </Toolbar.Actions>
        )}
      </Toolbar.Header>
      <Toolbar.Body>
        <Lists emptyContent={EMPTY_CONTENT} />
      </Toolbar.Body>
    </Toolbar.Content>
  );
};

const useVisible = (): boolean => {
  const { secrets, certificates } = useRetrievable();
  return secrets || certificates;
};

export const TOOLBAR: Nav.Toolbar = {
  key: "credentials",
  icon: <Icon.Secret />,
  content: <Content />,
  trigger: ["K"],
  tooltip: "Credentials",
  initialSize: 300,
  sizeBounds: { lower: 225, upper: 400 },
  useVisible,
};
