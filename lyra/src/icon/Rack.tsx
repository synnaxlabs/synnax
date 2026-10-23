// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { wrapSVGIcon } from "@/icon/Icon";

export const Rack = wrapSVGIcon(
  (props) => (
    <svg
      viewBox="0 0 24 24"
      xmlns="http://www.w3.org/2000/svg"
      stroke="currentColor"
      fill="currentColor"
      strokeWidth={0}
      {...props}
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M4.25 11C3.283 11 2.5 10.217 2.5 9.25V5.25C2.5 4.283 3.283 3.5 4.25 3.5H19.75C20.717 3.5 21.5 4.283 21.5 5.25V9.25C21.5 10.217 20.717 11 19.75 11ZM6.5 8.75C7.328 8.75 8 8.078 8 7.25C8 6.422 7.328 5.75 6.5 5.75C5.672 5.75 5 6.422 5 7.25C5 8.078 5.672 8.75 6.5 8.75ZM17.25 8.25C17.802 8.25 18.25 7.802 18.25 7.25C18.25 6.698 17.802 6.25 17.25 6.25H14.25C13.698 6.25 13.25 6.698 13.25 7.25C13.25 7.802 13.698 8.25 14.25 8.25Z"
      />
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M4.25 20.5C3.283 20.5 2.5 19.717 2.5 18.75V14.75C2.5 13.783 3.283 13 4.25 13H19.75C20.717 13 21.5 13.783 21.5 14.75V18.75C21.5 19.717 20.717 20.5 19.75 20.5ZM6.5 18.25C7.328 18.25 8 17.578 8 16.75C8 15.922 7.328 15.25 6.5 15.25C5.672 15.25 5 15.922 5 16.75C5 17.578 5.672 18.25 6.5 18.25ZM17.25 17.75C17.802 17.75 18.25 17.302 18.25 16.75C18.25 16.198 17.802 15.75 17.25 15.75H14.25C13.698 15.75 13.25 16.198 13.25 16.75C13.25 17.302 13.698 17.75 14.25 17.75Z"
      />
    </svg>
  ),
  "rack",
);
