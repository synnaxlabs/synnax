// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { Change } from "@/feature/vcs/diff/change";
import { CSS } from "@/platform/css";
import { Link } from "@/platform/link";

interface PartProps {
  part: Change.Part;
}

const Reference = ({
  text,
  resource,
}: Required<Change.Part>): ReactElement | string => {
  const open = Link.useOpen(resource.type);
  if (open == null) return text;
  return (
    <span
      role="link"
      tabIndex={0}
      className={CSS.BE("vcs-diff", "link")}
      onClick={() => open(resource.key)}
      onKeyDown={(e) => {
        if (e.key === "Enter") open(resource.key);
      }}
    >
      {text}
    </span>
  );
};

const Part = ({ part: { text, resource } }: PartProps): ReactElement | string =>
  resource == null ? text : <Reference text={text} resource={resource} />;

interface ValueProps {
  side: "before" | "after";
  parts: Change.Part[];
}

const Value = ({ side, parts }: ValueProps): ReactElement => (
  <Text.Text
    level="small"
    overflow="fade"
    title={Change.text(parts)}
    className={CSS.BEM("vcs-diff", "value", side)}
  >
    {parts.map((part, i) => (
      <Part key={i} part={part} />
    ))}
  </Text.Text>
);

const KIND_TEXT: Record<"added" | "removed", string> = {
  added: "Added",
  removed: "Removed",
};

interface PropertyProps {
  property: Change.Property;
}

const Property = ({ property }: PropertyProps): ReactElement => {
  const { path, kind, before, after } = property;
  const parents = path.slice(0, -1);
  const leaf = path.at(-1) ?? "";
  let value: ReactElement;
  if (kind === "changed")
    value = (
      <>
        {before != null && <Value side="before" parts={before} />}
        {before != null && after != null && <Icon.Arrow.Right />}
        {after != null && <Value side="after" parts={after} />}
        {before == null && after == null && (
          <Text.Text level="small" color={9}>
            Changed
          </Text.Text>
        )}
      </>
    );
  else if (after != null) value = <Value side="after" parts={after} />;
  else if (before != null) value = <Value side="before" parts={before} />;
  else
    value = (
      <Text.Text level="small" status={kind === "added" ? "success" : "error"}>
        {KIND_TEXT[kind]}
      </Text.Text>
    );
  return (
    <div className={CSS.BE("vcs-diff", "property")}>
      <Text.Text level="small" overflow="fade" title={Change.formatPath(property)}>
        {parents.length > 0 && (
          <Text.Text el="span" level="small" color={8}>
            {`${parents.join(" › ")} › `}
          </Text.Text>
        )}
        {leaf}
      </Text.Text>
      <Flex.Box x align="center" gap="small" className={CSS.BE("vcs-diff", "values")}>
        {value}
      </Flex.Box>
    </div>
  );
};

export interface PropertyListProps {
  properties: Change.Property[];
}

/**
 * Lists properties, each with its path and its old and new values. Shallow properties
 * come first, such as a task's sample rate before the settings of its channels.
 */
export const PropertyList = ({ properties }: PropertyListProps): ReactElement => (
  <Flex.Box y empty className={CSS.BE("vcs-diff", "properties")}>
    {properties
      .toSorted((a, b) => a.path.length - b.path.length)
      .map((p) => (
        <Property key={`${p.kind}:${Change.formatPath(p)}`} property={p} />
      ))}
  </Flex.Box>
);
