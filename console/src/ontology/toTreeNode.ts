// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { type Tree as Core } from "@synnaxlabs/pluto";
import { isValidElement } from "react";

import { type Services } from "@/ontology/service";

export const toTreeNode = (
  services: Services,
  resource: ontology.Resource,
): Core.Node => {
  const { id, name } = resource;
  const { icon, hasChildren, haulItems } = services[id.type];
  return {
    key: id.toString(),
    name,
    icon: isValidElement(icon) ? icon : icon(resource),
    hasChildren,
    haulItems: haulItems(resource),
    allowRename: services[id.type].allowRename(resource),
    extraData: resource.data ?? undefined,
  };
};

export const toTreeNodes = (
  services: Services,
  resources: ontology.Resource[],
): Core.Node[] => resources.map((res) => toTreeNode(services, res));
