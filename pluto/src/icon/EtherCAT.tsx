// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CSS } from "@/css";
import { wrapSVGIcon } from "@/icon/Icon";

export const EtherCAT = wrapSVGIcon(
  ({ className, ...rest }) => (
    <svg
      viewBox="0 0 243 129"
      {...rest}
      className={CSS(className, "logo")}
      stroke="currentColor"
      fill="currentColor"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path d="M123.272 50.6L176.172 92.7V72.8H242.272V50.6H123.272Z" fill="white" />
      <path d="M242.272 42.1L189.372 0V19.9H30.872V42.1H242.272Z" fill="white" />
      <path
        d="M0 50.32H51.408V64.468H15.768V81.532H50.112V95.572H15.768V112.852H52.272V127H0V50.32Z"
        fill="white"
      />
      <path
        d="M120.054 98.596C117.57 117.064 106.554 128.728 91.5418 128.728C72.2098 128.728 60.4378 113.284 60.4378 88.768C60.4378 64.036 72.2098 48.592 91.5418 48.592C106.122 48.592 117.03 59.932 119.946 77.86L103.314 78.832C101.37 67.6 97.2658 62.74 91.0018 62.74C81.8218 62.74 77.0698 71.704 77.0698 88.768C77.0698 105.724 81.8218 114.58 91.0018 114.58C97.5898 114.58 101.694 109.504 103.422 97.732L120.054 98.596Z"
        fill="white"
      />
    </svg>
  ),
  "logo-ethercat",
);
