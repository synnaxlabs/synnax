// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Log } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Tab } from "@/log/tab";
import { type Panel } from "@/panel";

export const useName = (): Panel.UseNameReturn => {
  const { key } = Tab.useArgs();
  const name = Log.useSelectName({ key });
  const { update } = Log.useRename();
  const rename = useCallback((name: string) => update({ key, name }), [key]);
  return { name, rename };
};
