// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { http } from "@synnaxlabs/client";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import type { Tree } from "@synnaxlabs/lyra/tree";
import type { deep } from "@synnaxlabs/x";
import type { ReactNode } from "react";

import { Task } from "@/platform/task";

export interface EndpointLabelProps {
  /** The endpoint's form path. */
  path: deep.Path<{ method: http.Method; path: string }>;
}

// The API-docs convention: each method gets its own hue.
const METHOD_COLORS: Partial<Record<http.Method, string>> = {
  GET: "var(--pluto-secondary-z)",
  POST: "var(--pluto-warning-z)",
  PUT: "var(--pluto-primary-z)",
  PATCH: "var(--pluto-success-z)",
};

/** Names an endpoint by its method and path; an empty path reads as a new endpoint. */
export const EndpointLabel = ({ path }: EndpointLabelProps) => {
  const method = PForm.useFieldValue(path.method);
  const epPath = PForm.useFieldValue(path.path);
  return (
    <Task.Views.ItemLabel
      kind={method}
      kindColor={METHOD_COLORS[method]}
      color={epPath === "" ? 8 : 10}
      variant="code"
    >
      {epPath === "" ? "New endpoint" : epPath}
    </Task.Views.ItemLabel>
  );
};

export interface EndpointItemProps extends Tree.ItemRenderProps<string> {
  /** The endpoint's form path. */
  path: deep.Path<{ method: http.Method; path: string; disabled: boolean }>;
  /** Marks before the checkbox, such as a problem with the message. */
  children?: ReactNode;
}

/** An endpoint in the tree of a task, with the checkbox that switches it on and off. */
export const EndpointItem = ({ path, children, ...props }: EndpointItemProps) => {
  const disabled = PForm.useFieldValue(path.disabled, { optional: true });
  if (disabled == null) return null;
  return (
    <Task.Views.TreeGroupItem
      {...props}
      label={<EndpointLabel path={path} />}
      off={disabled}
    >
      {children}
      <Task.EnabledCheckbox path={path.disabled} />
    </Task.Views.TreeGroupItem>
  );
};
