// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, type ontology, policy } from "@synnaxlabs/client";

import { useMemoSelect } from "@/hooks";
import {
  ALLOW_ALL,
  SLICE_NAME,
  type SliceState,
  type StoreState,
} from "@/permissions/slice";

const selectState = (state: StoreState): SliceState => state[SLICE_NAME];

export const selectPolicies = (state: StoreState): policy.Policy[] => {
  const policies = selectState(state).policies;
  if (policies === ALLOW_ALL) return [];
  return policies;
};

export const useSelectPolicies = (): policy.Policy[] =>
  useMemoSelect(selectPolicies, []);

export const selectHasAllPermissions = (state: StoreState): boolean =>
  selectState(state).policies === ALLOW_ALL;

export const useSelectHasAllPermissions = (): boolean =>
  useMemoSelect(selectHasAllPermissions, []);

export const selectCanUseType = (
  state: StoreState,
  type: ontology.ResourceType,
): boolean => {
  if (selectHasAllPermissions(state)) return true;
  const policies = selectPolicies(state);
  return policies.some((p) =>
    p.objects.some((object) => {
      const type_ = object.type;
      return (
        type_ === policy.ALLOW_ALL_ONTOLOGY_TYPE ||
        (type_ === type && p.actions.includes(access.ALL_ACTION))
      );
    }),
  );
};

export const useSelectCanUseType = (type: ontology.ResourceType): boolean =>
  useMemoSelect((state: StoreState) => selectCanUseType(state, type), [type]);

export const selectCanEditPolicies = (state: StoreState): boolean =>
  selectCanUseType(state, policy.ONTOLOGY_TYPE);

export const useSelectCanEditPolicies = (): boolean =>
  useMemoSelect(selectCanEditPolicies, []);
