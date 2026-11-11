// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/vcs/Compare.css";

import { type vcs } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Synnax } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useState } from "react";

import {
  Empty,
  ItemList,
  Loading,
  ResourceItem,
  SectionHeader,
  ViewHeader,
} from "@/feature/vcs/common";
import { Diff } from "@/feature/vcs/diff";
import { CSS } from "@/platform/css";
import { Vcs } from "@/platform/vcs";

export interface CompareProps {
  view: Vcs.CompareView;
}

/** Lists the resources that differ between two revs. Each one opens its diff. */
export const Compare = ({ view }: CompareProps): ReactElement => {
  const client = Synnax.use();
  const update = Vcs.useUpdateUI();
  const openDiff = Diff.useOpen();
  const shows = Diff.useShows();
  const [changes, setChanges] = useState<vcs.Change[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const { from, to, fromLabel, toLabel, title } = view;
  useEffect(() => {
    if (client == null) return;
    let cancelled = false;
    setChanges(null);
    setError(null);
    client.vcs
      .compare(from, to)
      .then((c) => {
        if (!cancelled) setChanges(c.sort((a, b) => a.name.localeCompare(b.name)));
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(errors.fromUnknown(e).message);
      });
    return () => {
      cancelled = true;
    };
  }, [client, from, to]);
  const back = () => update((prev) => ({ ...prev, view: { variant: "main" } }));
  let body: ReactElement;
  if (error != null)
    body = (
      <Status.Summary
        variant="error"
        level="small"
        center={false}
        message="Failed to compare"
        description={error}
        className={CSS.BE("vcs", "status")}
      />
    );
  else if (changes == null) body = <Loading />;
  else if (changes.length === 0)
    body = (
      <Empty
        message="No differences"
        description={`${fromLabel} and ${toLabel} hold the same configuration.`}
      />
    );
  else
    body = (
      <ItemList>
        <SectionHeader title={Vcs.count(changes.length, "difference")} />
        {changes.map((change) => (
          <ResourceItem
            key={change.resource}
            itemKey={change.resource}
            type={change.type}
            name={change.name}
            end={<Vcs.KindMarker kind={change.kind} />}
            selected={shows(change.resource, from, to)}
            onClick={() =>
              openDiff({
                resource: change.resource,
                type: change.type,
                name: change.name,
                before: from,
                beforeLabel: fromLabel,
                after: to,
                afterLabel: toLabel,
              })
            }
          />
        ))}
      </ItemList>
    );
  return (
    <Flex.Box y full empty className={CSS.B("vcs-compare")}>
      <ViewHeader title={title} onBack={back} />
      <Flex.Box
        x
        align="center"
        gap="small"
        shrink={false}
        className={CSS.BE("vcs-compare", "revs")}
      >
        <Text.Text level="small" color={9} gap="tiny" overflow="fade">
          <Icon.Branch />
          {fromLabel}
        </Text.Text>
        <Icon.Arrow.Right className={CSS.BE("vcs-compare", "arrow")} />
        <Text.Text level="small" color={9} gap="tiny" overflow="fade">
          <Icon.Branch />
          {toLabel}
        </Text.Text>
      </Flex.Box>
      {body}
    </Flex.Box>
  );
};
