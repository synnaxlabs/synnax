// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type user } from "@synnaxlabs/client";

import { useMemoSelect } from "@/hooks";
import { Permissions } from "@/permissions";
import { SLICE_NAME, type StoreState } from "@/user/slice";

export const selectHasPermission = (state: Permissions.StoreState): boolean =>
  Permissions.selectCanUseType(state, "user");

export const useSelectHasPermission = (): boolean =>
  useMemoSelect(selectHasPermission, []);

export const selectSliceState = (state: StoreState) => state[SLICE_NAME];

export const select = (state: StoreState): user.User => selectSliceState(state).user;

export const selectUsername = (state: StoreState): string => select(state).username;

export const selectKey = (state: StoreState): string => select(state).key;

export const selectIsRoot = (state: StoreState): boolean => select(state).rootUser;

export const selectFullName = (state: StoreState): string => {
  const currentUser = select(state);
  return (
    `${currentUser.firstName} ${currentUser.lastName}`.trim() || currentUser.username
  );
};

export const useSelect = (): user.User => useMemoSelect(select, []);

export const useSelectUsername = (): string => useMemoSelect(selectUsername, []);

export const useSelectKey = (): string => useMemoSelect(selectKey, []);

export const useSelectIsRoot = (): boolean => useMemoSelect(selectIsRoot, []);

export const useSelectFullName = (): string => useMemoSelect(selectFullName, []);
