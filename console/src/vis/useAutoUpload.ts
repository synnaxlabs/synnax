// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PayloadAction } from "@reduxjs/toolkit";
import { type project } from "@synnaxlabs/client";
import { type Flux } from "@synnaxlabs/pluto";
import { useCallback, useEffect, useRef } from "react";
import { useDispatch, useStore } from "react-redux";

import { Layout } from "@/layout";
import { Project } from "@/project";
import { type RootState } from "@/store";

export interface ClearPendingUploadPayload {
  key: string;
}

export interface UploadFields {
  key: string;
  // name is the hosting layout's name, when the upload is hosted by a window or
  // modal layout. Panel tabs have no layout-slice entry, so the staged body's own
  // name is used as-is.
  name?: string;
  project?: project.Key;
}

export interface CreateUseAutoUploadArgs<
  Pending,
  Input extends Flux.Data,
  Output extends Flux.Data & { key: string },
> {
  /** Selects the staged upload body for the given layout key, or undefined once landed. */
  useSelectPendingUpload: (key: string) => Pending | undefined;
  /** Merges the layout-resolved fields into the staged body to form the create body. */
  toCreateParams: (pending: Pending, fields: UploadFields) => Input;
  /** The Flux create hook for the visualization's resource. */
  useCreate: Flux.UseUpdate<Input, Output>;
  /** The slice action that drops the staged body once the upload has landed. */
  clearPendingUpload: (
    payload: ClearPendingUploadPayload,
  ) => PayloadAction<ClearPendingUploadPayload>;
}

/**
 * Builds a useAutoUpload hook for a visualization that stages a pendingUpload body on
 * the client until it has landed on the server. The returned hook sends the body once
 * on first mount, clears it on success, and reports whether the upload has settled.
 *
 * The upload is keyed to the layout: the visualization's resource key is set to its
 * layout key, so a layout deterministically maps to one resource.
 */
export const createUseAutoUpload =
  <Pending, Input extends Flux.Data, Output extends Flux.Data & { key: string }>({
    useSelectPendingUpload,
    toCreateParams,
    useCreate,
    clearPendingUpload,
  }: CreateUseAutoUploadArgs<Pending, Input, Output>): ((key: string) => boolean) =>
  (key) => {
    const store = useStore<RootState>();
    const pendingUpload = useSelectPendingUpload(key);
    const project = Project.useSelectActiveKey() ?? undefined;
    const dispatch = useDispatch();
    // Guards against a second create while the first is in flight: the effect re-runs if
    // project changes before afterSuccess clears pendingUpload. A remount (new
    // instance) resets the ref, so a failed upload still retries.
    const startedRef = useRef(false);
    const { update: create } = useCreate({
      afterSuccess: useCallback(
        ({ data: { key } }: Flux.AfterSuccessParams<Output>) => {
          dispatch(clearPendingUpload({ key }));
        },
        [dispatch],
      ),
    });
    useEffect(() => {
      if (pendingUpload == null || startedRef.current) return;
      startedRef.current = true;
      const name = Layout.select(store.getState(), key)?.name;
      create(
        toCreateParams(pendingUpload, {
          key,
          project,
          ...(name != null ? { name } : {}),
        }),
      );
    }, [pendingUpload, project, key, create, toCreateParams]);
    return pendingUpload == null;
  };
