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

export const Sift = wrapSVGIcon(
  ({ className, ...rest }) => (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      version="1.1"
      className={CSS(className, "logo")}
      {...rest}
      stroke="currentColor"
      fill="currentColor"
      viewBox="0 0 832 513"
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M460 0.516204C407.078 3.1722 363.332 20.1982 323 53.8362C304.377 69.3682 285.917 88.3622 255.129 123.665C215.37 169.256 198.5 185.401 172.39 202.85C156.716 213.325 141.933 220.542 122.75 227.085C120.137 227.977 118 229.051 118 229.473C118 229.895 278.682 230.268 475.071 230.302C717.523 230.344 831.94 230.036 831.512 229.344C831.166 228.783 830.24 228.324 829.456 228.324C826.999 228.324 807.858 221.61 799.695 217.886C789.827 213.383 777.772 205.785 767.5 197.594C754.251 187.03 720.867 153.252 702.499 131.826C667.095 90.5282 645.649 69.0802 621 50.3222C582.81 21.2582 543.665 5.5672 499 1.4202C481.379 -0.216797 476.733 -0.323796 460 0.516204ZM0 282.731C0 282.955 5.269 284.798 11.709 286.826C37.533 294.959 60.242 309.792 87.39 336.258C99.296 347.864 108.317 357.742 145 399.336C192.77 453.503 230.794 482.282 276.37 498.767C327.438 517.239 387.531 516.794 442 497.541C451.665 494.125 474.103 482.328 487 473.884C514.5 455.878 534.038 436.931 581.931 381.824C629.443 327.156 664.884 299.367 702.5 287.287C708 285.52 712.927 283.681 713.45 283.2C713.993 282.699 561.553 282.324 357.2 282.324C160.74 282.324 0 282.507 0 282.731Z"
      />
    </svg>
  ),
  "logo-sift",
);
