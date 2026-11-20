// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type enumeration, type resource } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Enumeration, Resource } from "@synnaxlabs/pluto";
import { type ReactElement, Suspense, useState } from "react";

import { Modals } from "@/platform/modals";
import { Tree } from "@/platform/tree";

const NO_USERS: resource.ID[] = [];
const IN_USE = "Each of these moves to the enumeration you pick.";
const NOT_IN_USE = "Nothing uses it.";
const CANNOT_UNDO = "This action cannot be undone.";
const REPLACEMENT_REQUIRED = "Replacement is required";
const TRIGGER_PROPS = { "aria-label": "Replace with" };

interface UserProps {
  resource: resource.Resource;
}

const User = ({ resource: r }: UserProps): ReactElement => {
  const Item = Tree.useItems()[r.id.type] ?? Tree.DefaultItem;
  return (
    <Text.Text overflow="fade">
      {Item.icon(r)}
      {r.name}
    </Text.Text>
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
  const res = Enumeration.useResult({ key: e.key, includeUses: true });
  const users = res.data?.uses ?? NO_USERS;
  const [replacement, setReplacement] = useState<enumeration.Key>();
  const [confirmed, setConfirmed] = useState(false);
  const { update: remove, variant: removeVariant } = Enumeration.useDelete({
    afterSuccess: () => close(),
  });
  const { update: replace, variant: replaceVariant } = Enumeration.useReplace({
    afterSuccess: () => close(),
  });
  const busy = removeVariant === "loading" || replaceVariant === "loading";
  Modals.useBusy(busy);
  const inUse = users.length > 0;
  const error = confirmed && replacement == null ? REPLACEMENT_REQUIRED : undefined;
  const handleConfirm = () => {
    if (!inUse) return remove(e.key);
    setConfirmed(true);
    if (replacement != null) replace({ from: e.key, to: replacement });
  };
  return (
    <Modals.ConfirmLayout
      title="Enumeration.Delete"
      icon={<Icon.Enumeration />}
      message={`Are you sure you want to delete ${e.name}?`}
      description={
        res.variant === "success"
          ? `${inUse ? IN_USE : NOT_IN_USE} ${CANNOT_UNDO}`
          : CANNOT_UNDO
      }
      confirm={{
        label: inUse ? "Replace" : "Delete",
        variant: busy ? "loading" : "error",
        disabled: res.variant !== "success",
      }}
      cancel={{ disabled: busy }}
      onConfirm={handleConfirm}
      onCancel={() => close()}
    >
      {res.variant === "error" && <Status.Summary level="small" status={res.status} />}
      {inUse && (
        <>
          <Suspense>
            <Users ids={users} />
          </Suspense>
          <Input.Item
            label="Replace with"
            helpText={error}
            status={error == null ? undefined : "error"}
            padHelpText
          >
            <Enumeration.SelectSingle
              triggerProps={TRIGGER_PROPS}
              value={replacement}
              onChange={setReplacement}
              filter={(other) => other.key !== e.key}
            />
          </Input.Item>
        </>
      )}
    </Modals.ConfirmLayout>
  );
};

/** Opens a dialog that deletes an unused enumeration, or replaces one in use. */
export const useDeleteModal = Modals.create(Delete);
