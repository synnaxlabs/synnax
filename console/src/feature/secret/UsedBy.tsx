// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type certificate, type ontology, type secret } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Text } from "@synnaxlabs/lyra/text";
import { Errors, Flux, Ontology, Secret } from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { Tree } from "@/platform/tree";

/** Selects a certificate in the toolbar instead of opening it. */
export interface SelectCertificate {
  (key: certificate.Key): void;
}

// A secret deleted while its uses show unmounts with its item, so the read renders
// nothing. Any other failure reaches the toolbar boundary.
const DeletedFallback = ({ error }: Errors.FallbackProps): null => {
  if (Flux.DeletedError.matches(error)) return null;
  throw error;
};

/** Suspends a read of the uses of a secret without a loading indicator. */
export const UsesBoundary = ({ children }: PropsWithChildren): ReactElement => (
  <Errors.SuspenseBoundary loading={null} FallbackComponent={DeletedFallback}>
    {children}
  </Errors.SuspenseBoundary>
);

const NO_USES: ontology.ID[] = [];

/** Returns the IDs of the resources that use the secret. Suspends while they load. */
export const useUses = (key: secret.Key): ontology.ID[] =>
  Secret.use({ key, includeUses: true }).uses ?? NO_USES;

interface ResourceProps {
  resource: ontology.Resource;
  onSelectCertificate?: SelectCertificate;
}

/** A resource that uses a secret, shown with the icon of its type. */
const Resource = ({ resource, onSelectCertificate }: ResourceProps): ReactElement => {
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

interface ResourcesProps extends Pick<ResourceProps, "onSelectCertificate"> {
  ids: ontology.ID[];
}

const Resources = ({ ids, onSelectCertificate }: ResourcesProps): ReactElement => {
  const resources = Ontology.useResources({ ids });
  return (
    <>
      {resources.map((r) => (
        <Resource
          key={`${r.id.type}:${r.id.key}`}
          resource={r}
          onSelectCertificate={onSelectCertificate}
        />
      ))}
    </>
  );
};

export interface UsedByProps extends Pick<ResourceProps, "onSelectCertificate"> {
  secretKey: secret.Key;
}

/** Lists the resources that use a secret. Suspends while they load. */
export const UsedBy = ({
  secretKey,
  onSelectCertificate,
}: UsedByProps): ReactElement => {
  const uses = useUses(secretKey);
  if (uses.length === 0)
    return (
      <Text.Text level="small" color={9} className={CSS.BE("credentials", "unused")}>
        Nothing uses this secret.
      </Text.Text>
    );
  return (
    <Flex.Box y empty>
      <Resources ids={uses} onSelectCertificate={onSelectCertificate} />
    </Flex.Box>
  );
};
