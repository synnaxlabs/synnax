// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/vcs/ModifiedMarker.css";

import { type resource, type vcs } from "@synnaxlabs/client";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { FLAGS } from "@/flags";
import { CSS } from "@/platform/css";
import { usePolledStatus } from "@/platform/vcs/Provider";
import { idString } from "@/platform/vcs/resource";

const LETTERS: Record<vcs.ChangeKind, string> = {
  added: "A",
  modified: "M",
  deleted: "D",
};

const LABELS: Record<vcs.ChangeKind, string> = {
  added: "Added",
  modified: "Modified",
  deleted: "Deleted",
};

const STATUSES: Record<vcs.ChangeKind, "success" | "warning" | "error"> = {
  added: "success",
  modified: "warning",
  deleted: "error",
};

export interface KindMarkerProps {
  kind: vcs.ChangeKind;
}

/** A one-letter marker for the kind of an uncommitted change. */
export const KindMarker = ({ kind }: KindMarkerProps): ReactElement => (
  <Text.Text
    level="tiny"
    weight={500}
    variant="code"
    status={STATUSES[kind]}
    className={CSS.B("vcs-kind-marker")}
    aria-label={LABELS[kind]}
    title={LABELS[kind]}
  >
    {LETTERS[kind]}
  </Text.Text>
);

export interface ModifiedMarkerProps {
  id: resource.ID;
}

const Marker = ({ id }: ModifiedMarkerProps): ReactElement | null => {
  const status = usePolledStatus();
  if (status?.enabled !== true) return null;
  const key = idString(id);
  const change = status.changes.find(({ resource }) => resource === key);
  if (change == null) return null;
  return <KindMarker kind={change.kind} />;
};

/** Marks a tree item whose resource has uncommitted changes. */
export const ModifiedMarker = (props: ModifiedMarkerProps): ReactElement | null =>
  FLAGS.versionControl ? <Marker {...props} /> : null;
