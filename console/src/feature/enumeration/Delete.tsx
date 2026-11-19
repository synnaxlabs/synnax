// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type enumeration, type resource } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Triggers as PTriggers } from "@synnaxlabs/lyra/triggers";
import { Enumeration, Resource } from "@synnaxlabs/pluto";
import { type ReactElement, Suspense, useState } from "react";

import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Tree } from "@/platform/tree";
import { Triggers } from "@/platform/triggers";

const NO_USERS: resource.ID[] = [];
const IN_USE = "Each of these moves to the enumeration you pick.";
const NOT_IN_USE = "Nothing uses it.";
const TRIGGER_PROPS = { "aria-label": "Replace with" };

interface UserProps {
  resource: resource.Resource;
}

const User = ({ resource: r }: UserProps): ReactElement => {
  const Item = Tree.useItems()[r.id.type] ?? Tree.DefaultItem;
  return (
    <Flex.Box x align="center" gap="small">
      {Item.icon(r)}
      <Text.Text overflow="fade">{r.name}</Text.Text>
    </Flex.Box>
  );
};

interface UsersProps {
  ids: resource.ID[];
}

/** Lists the records that use the enumeration. Suspends while they load. */
const Users = ({ ids }: UsersProps): ReactElement => (
  <Flex.Box y gap="small" aria-label="Users">
    {Resource.useMultiple({ ids }).map((r) => (
      <User key={`${r.id.type}:${r.id.key}`} resource={r} />
    ))}
  </Flex.Box>
);

interface DeleteParams {
  enumeration: Pick<enumeration.Enumeration, "key" | "name">;
}

const Delete = ({
  enumeration: e,
  close,
}: Modals.ContentProps<DeleteParams, void>): ReactElement => {
  const dependents = Enumeration.useResultDependents({ key: e.key });
  const users = dependents.data ?? NO_USERS;
  const [replacement, setReplacement] = useState<enumeration.Key>();
  const { update: remove } = Enumeration.useDelete({ afterSuccess: () => close() });
  const { update: replace } = Enumeration.useReplace({ afterSuccess: () => close() });
  const inUse = users.length > 0;
  const handleConfirm = () => {
    if (inUse && replacement != null) replace({ from: e.key, to: replacement });
    else remove(e.key);
  };
  return (
    <Modal.Frame className={CSS.B("confirm")}>
      <Modal.Header icon={<Icon.Enumeration />}>Enumeration.Delete</Modal.Header>
      <Modal.Body gap="medium" className={CSS.BE("confirm", "body")}>
        <Text.Text level="h3" weight={450}>
          Are you sure you want to delete {e.name}?
        </Text.Text>
        {dependents.variant === "error" ? (
          <Status.Summary level="small" status={dependents.status} />
        ) : (
          <Text.Text weight={450}>
            {dependents.variant === "success" && (inUse ? IN_USE : NOT_IN_USE)} This
            action cannot be undone.
          </Text.Text>
        )}
        {inUse && (
          <>
            <Suspense>
              <Users ids={users} />
            </Suspense>
            <Input.Item label="Replace with">
              <Enumeration.SelectSingle
                triggerProps={TRIGGER_PROPS}
                value={replacement}
                onChange={setReplacement}
                filter={(other) => other.key !== e.key}
              />
            </Input.Item>
          </>
        )}
      </Modal.Body>
      <Modal.Footer>
        <Nav.Bar.End x align="center">
          <Button.Button
            variant="outlined"
            triggerIndicator={PTriggers.ESCAPE}
            onClick={() => close()}
          >
            Cancel
          </Button.Button>
          <Button.Button
            variant="filled"
            status="error"
            trigger={Triggers.SAVE}
            triggerIndicator
            disabled={
              dependents.variant !== "success" || (inUse && replacement == null)
            }
            onClick={handleConfirm}
          >
            {inUse ? "Replace" : "Delete"}
          </Button.Button>
        </Nav.Bar.End>
      </Modal.Footer>
    </Modal.Frame>
  );
};

/** Opens a dialog that deletes an unused enumeration, or replaces one in use. */
export const useDeleteModal = Modals.create(Delete);
