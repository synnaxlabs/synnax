// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type certificate, type resource, type secret } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Text } from "@synnaxlabs/lyra/text";
import { Resource, Secret } from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactElement, Suspense } from "react";

import { CSS } from "@/platform/css";
import { Tree } from "@/platform/tree";

/** Selects a certificate in the toolbar instead of opening it. */
export interface SelectCertificate {
  (key: certificate.Key): void;
}

/**
 * Suspends a read of the uses of a secret without a loading indicator. A failed read
 * reaches the toolbar boundary.
 */
export const UsesBoundary = ({ children }: PropsWithChildren): ReactElement => (
  <Suspense fallback={null}>{children}</Suspense>
);

const NO_USES: resource.ID[] = [];

/** Every secret's uses, read in one request that a deleted secret cannot fail. */
const ALL_USES: Secret.RetrieveMultipleQuery = { includeUses: true };

/**
 * Returns the IDs of the resources that use the secret and that the caller can read.
 * Suspends while the uses load.
 */
export const useUses = (key: secret.Key): resource.ID[] =>
  Secret.useMultiple(ALL_USES).find((s) => s.key === key)?.uses ?? NO_USES;

interface UseProps {
  resource: resource.Resource;
  onSelectCertificate?: SelectCertificate;
}

/** A resource that uses a secret, shown with the icon of its type. */
const Use = ({ resource, onSelectCertificate }: UseProps): ReactElement => {
  const Item = Tree.useItems()[resource.id.type] ?? Tree.DefaultItem;
  const renderButton = (onClick: () => void) => (
    <Button.Button
      variant="text"
      size="small"
      justify="start"
      full="x"
      className={CSS.BE("credentials", "resource")}
      onClick={onClick}
    >
      {Item.icon(resource)}
      <Text.Text level="small" overflow="fade" color={10}>
        {resource.name}
      </Text.Text>
    </Button.Button>
  );
  if (resource.id.type === "certificate" && onSelectCertificate != null)
    return renderButton(() => onSelectCertificate(resource.id.key));
  return (
    <Item.OnSelect>
      {(onSelect) => renderButton(() => onSelect(resource))}
    </Item.OnSelect>
  );
};

interface UsesProps extends Pick<UseProps, "onSelectCertificate"> {
  ids: resource.ID[];
}

const Unused = (): ReactElement => (
  <Text.Text level="small" color={9} className={CSS.BE("credentials", "unused")}>
    Nothing uses this secret.
  </Text.Text>
);

const Uses = ({ ids, onSelectCertificate }: UsesProps): ReactElement => {
  // A resource in the uses can be deleted between the read of the uses and this read.
  const resources = Resource.useMultiple({ ids });
  if (resources.length === 0) return <Unused />;
  return (
    <Flex.Box y empty className={CSS.BE("credentials", "resources")}>
      {resources.map((r) => (
        <Use
          key={`${r.id.type}:${r.id.key}`}
          resource={r}
          onSelectCertificate={onSelectCertificate}
        />
      ))}
    </Flex.Box>
  );
};

export interface UsedByProps extends Pick<UseProps, "onSelectCertificate"> {
  secretKey: secret.Key;
}

/** Lists the resources that use a secret. Suspends while they load. */
export const UsedBy = ({
  secretKey,
  onSelectCertificate,
}: UsedByProps): ReactElement => {
  const uses = useUses(secretKey);
  if (uses.length === 0) return <Unused />;
  return <Uses ids={uses} onSelectCertificate={onSelectCertificate} />;
};
