// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { wrapSVGIcon } from "@/icon/Icon";

export const Fitting = wrapSVGIcon(
  (props) => (
    <svg
      version="1.1"
      viewBox="0 0 1200 1200"
      xmlns="http://www.w3.org/2000/svg"
      stroke="currentColor"
      fill="currentColor"
      {...props}
    >
      <path d="m815.76 549.6v284.52h104.64v-284.52z" />
      <path d="m526.8 467.52 28.078-27.84-74.039-74.039-201.24 201.36 74.039 74.039 28.32-28.438z" />
      <path d="m670.2 607.56c1.8008 1.6797 1.8008 4.6797 0 6.6016-0.83984 0.71875-2.0391 1.1992-3.1211 1.1992-1.0781 0-2.3984-0.48047-3.2383-1.1992l-133.8-136.68-138.48 138.48 171.72 173.88h243.36l0.003906-195.96h-149.64z" />
    </svg>
  ),
  "fitting",
);
