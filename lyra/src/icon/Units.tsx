// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { wrapSVGIcon } from "@/icon/Icon";

export const Units = {
  Acceleration: wrapSVGIcon(
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
          d="M22.108 12.454C22.369 12.151 22.369 11.848 22.108 11.545L16.642 5.204C16.381 4.901 16.25 4.95 16.25 5.35V18.65C16.25 19.05 16.381 19.099 16.642 18.796Z"
        />
        <path
          fillRule="evenodd"
          clipRule="evenodd"
          d="M8.424 18.826C8.141 19.109 8 19.05 8 18.65V15.25L10.826 12.424C11.109 12.141 11.109 11.858 10.826 11.575L8 8.749V5.349C8 4.949 8.141 4.89 8.424 5.173L14.825 11.574C15.108 11.857 15.108 12.14 14.825 12.423Z"
        />
        <path
          fillRule="evenodd"
          clipRule="evenodd"
          d="M2.25 13.125C1.974 13.125 1.75 12.901 1.75 12.625V11.375C1.75 11.099 1.974 10.875 2.25 10.875H11C11.276 10.875 11.5 11.099 11.5 11.375V12.625C11.5 12.901 11.276 13.125 11 13.125Z"
        />
      </svg>
    ),
    "units-acceleration",
  ),
  Current: wrapSVGIcon(
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
          d="M5.983 13.125C5.571 13.941 4.726 14.5 3.75 14.5C2.369 14.5 1.25 13.381 1.25 12C1.25 10.619 2.369 9.5 3.75 9.5C4.726 9.5 5.571 10.059 5.983 10.875L18.017 10.875C18.429 10.059 19.274 9.5 20.25 9.5C21.631 9.5 22.75 10.619 22.75 12C22.75 13.381 21.631 14.5 20.25 14.5C19.274 14.5 18.429 13.941 18.017 13.125ZM3.75 12.01C3.756 12.01 3.76 12.006 3.76 12C3.76 11.994 3.756 11.99 3.75 11.99C3.744 11.99 3.74 11.994 3.74 12C3.74 12.006 3.744 12.01 3.75 12.01Z"
        />
        <path
          fillRule="evenodd"
          clipRule="evenodd"
          d="M15.533 12.377C15.844 12.126 15.844 11.874 15.533 11.623L9.967 7.127C9.656 6.876 9.5 6.95 9.5 7.35V16.65C9.5 17.05 9.656 17.124 9.967 16.873Z"
        />
      </svg>
    ),
    "units-current",
  ),
  Force: wrapSVGIcon(
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
          d="M16.5 20C15.533 20 14.75 19.217 14.75 18.25V5.75C14.75 4.783 15.533 4 16.5 4H20.5C21.467 4 22.25 4.783 22.25 5.75V18.25C22.25 19.217 21.467 20 20.5 20Z"
        />
        <path
          fillRule="evenodd"
          clipRule="evenodd"
          d="M2.25 13.125C1.974 13.125 1.75 12.901 1.75 12.625V11.375C1.75 11.099 1.974 10.875 2.25 10.875H7.25C7.526 10.875 7.75 11.099 7.75 11.375V12.625C7.75 12.901 7.526 13.125 7.25 13.125Z"
        />
        <path
          fillRule="evenodd"
          clipRule="evenodd"
          d="M12.334 12.432C12.611 12.144 12.611 11.855 12.334 11.567L6.916 5.932C6.639 5.644 6.5 5.699 6.5 6.1V17.9C6.5 18.3 6.639 18.356 6.916 18.068Z"
        />
      </svg>
    ),
    "units-force",
  ),
  Pressure: wrapSVGIcon(
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
          d="M4.399 19.601C1.324 16.527 0.404 11.903 2.068 7.886C3.732 3.869 7.652 1.25 12 1.25C16.348 1.25 20.268 3.869 21.932 7.886C23.596 11.903 22.676 16.527 19.601 19.601L17.834 17.834C21.056 14.612 21.055 9.389 17.834 6.167C14.612 2.945 9.389 2.945 6.167 6.167C2.945 9.389 2.945 14.612 6.167 17.834Z"
        />
        <path
          fillRule="evenodd"
          clipRule="evenodd"
          d="M12 12C11.561 11.561 11.561 10.848 12 10.409L14.828 7.581C15.267 7.142 15.98 7.142 16.419 7.581V7.581C16.858 8.02 16.858 8.733 16.419 9.172L13.591 12C13.152 12.439 12.439 12.439 12 12Z"
        />
        <path
          fillRule="evenodd"
          clipRule="evenodd"
          d="M9.25 12C9.25 10.481 10.481 9.25 12 9.25C13.519 9.25 14.75 10.481 14.75 12C14.75 13.519 13.519 14.75 12 14.75C10.481 14.75 9.25 13.519 9.25 12Z"
        />
      </svg>
    ),
    "units-pressure",
  ),
  Resistance: wrapSVGIcon(
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
          d="M1.75 13.125H5.784L6.75 10.504L10.25 20.004L13.75 10.504L17.25 20.004L19.784 13.125H22.25V10.875H18.216L17.25 13.496L13.75 3.996L10.25 13.496L6.75 3.996L4.216 10.875H1.75Z"
        />
      </svg>
    ),
    "units-resistance",
  ),
  Strain: wrapSVGIcon(
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
          d="M4.625 18C4.349 18 4.125 17.776 4.125 17.5L4.125 4.5C4.125 4.5 4.125 4.5 4.125 4.5V3.5C4.125 3.224 4.349 3 4.625 3H10.375C10.651 3 10.875 3.224 10.875 3.5V13.75L13.125 13.75L13.125 4.5C13.125 4.5 13.125 4.5 13.125 4.5V3.5C13.125 3.224 13.349 3 13.625 3H19.375C19.651 3 19.875 3.224 19.875 3.5V17.5C19.875 17.776 19.651 18 19.375 18H18.125C17.849 18 17.625 17.776 17.625 17.5V5.25L15.375 5.25V15.5C15.375 15.776 15.151 16 14.875 16H9.125C8.849 16 8.625 15.776 8.625 15.5V5.25L6.375 5.25V17.5C6.375 17.776 6.151 18 5.875 18Z"
        />
        <path
          fillRule="evenodd"
          clipRule="evenodd"
          d="M3.25 21.75C2.836 21.75 2.5 21.414 2.5 21V18.75C2.5 18.336 2.836 18 3.25 18H7.25C7.664 18 8 18.336 8 18.75V21C8 21.414 7.664 21.75 7.25 21.75Z"
        />
        <path
          fillRule="evenodd"
          clipRule="evenodd"
          d="M16.75 21.75C16.336 21.75 16 21.414 16 21V18.75C16 18.336 16.336 18 16.75 18H20.75C21.164 18 21.5 18.336 21.5 18.75V21C21.5 21.414 21.164 21.75 20.75 21.75Z"
        />
      </svg>
    ),
    "units-strain",
  ),
  Temperature: wrapSVGIcon(
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
          d="M8.75 12.5V5C8.75 3.205 10.205 1.75 12 1.75C13.795 1.75 15.25 3.205 15.25 5V12.5C15.25 12.796 15.21 13.083 15.136 13.356C16.273 14.273 17 15.677 17 17.25C17 20.011 14.761 22.25 12 22.25C9.239 22.25 7 20.011 7 17.25C7 15.676 7.727 14.272 8.864 13.356C8.79 13.083 8.75 12.796 8.75 12.5ZM12 11.25C12.621 11.25 13.125 10.746 13.125 10.125V5.375C13.125 4.754 12.621 4.25 12 4.25C11.379 4.25 10.875 4.754 10.875 5.375V10.125C10.875 10.746 11.379 11.25 12 11.25Z"
        />
      </svg>
    ),
    "units-temperature",
  ),
  Torque: wrapSVGIcon(
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
          d="M20.877 6.875C23.707 11.778 22.027 18.046 17.125 20.877C12.223 23.707 5.954 22.028 3.123 17.125C0.293 12.223 1.972 5.954 6.875 3.123L8.125 5.288C4.418 7.428 3.148 12.168 5.288 15.875C7.428 19.582 12.168 20.852 15.875 18.712C19.582 16.572 20.852 11.832 18.712 8.126Z"
        />
        <path
          fillRule="evenodd"
          clipRule="evenodd"
          d="M9.42 6.498C9.224 6.846 9.025 6.847 8.825 6.501L6.175 1.911C5.975 1.565 6.075 1.393 6.475 1.397L11.663 1.449C12.063 1.453 12.165 1.629 11.968 1.978Z"
        />
        <path
          fillRule="evenodd"
          clipRule="evenodd"
          d="M8.75 12C8.75 10.205 10.205 8.75 12 8.75C13.795 8.75 15.25 10.205 15.25 12C15.25 13.795 13.795 15.25 12 15.25C10.205 15.25 8.75 13.795 8.75 12Z"
        />
      </svg>
    ),
    "units-torque",
  ),
  Velocity: wrapSVGIcon(
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
          d="M21.799 12.396C22.099 12.132 22.099 11.868 21.799 11.604L14.45 5.146C14.15 4.882 13.999 4.95 13.999 5.35V18.65C13.999 19.05 14.149 19.118 14.45 18.854Z"
        />
        <path
          fillRule="evenodd"
          clipRule="evenodd"
          d="M6.5 13.125C6.224 13.125 6 12.901 6 12.625V11.375C6 11.099 6.224 10.875 6.5 10.875H14.5C14.776 10.875 15 11.099 15 11.375V12.625C15 12.901 14.776 13.125 14.5 13.125Z"
        />
        <path
          fillRule="evenodd"
          clipRule="evenodd"
          d="M2.875 8.5C2.254 8.5 1.75 7.996 1.75 7.375V7.375C1.75 6.754 2.254 6.25 2.875 6.25H9.125C9.746 6.25 10.25 6.754 10.25 7.375V7.375C10.25 7.996 9.746 8.5 9.125 8.5Z"
        />
        <path
          fillRule="evenodd"
          clipRule="evenodd"
          d="M2.875 17.75C2.254 17.75 1.75 17.246 1.75 16.625V16.625C1.75 16.004 2.254 15.5 2.875 15.5H9.125C9.746 15.5 10.25 16.004 10.25 16.625V16.625C10.25 17.246 9.746 17.75 9.125 17.75Z"
        />
      </svg>
    ),
    "units-velocity",
  ),
  Voltage: wrapSVGIcon(
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
          d="M14.379 2.338C14.46 1.946 14.372 1.904 14.117 2.212L5.133 13.039C4.878 13.347 4.95 13.501 5.35 13.501H10.65C11.05 13.501 11.211 13.697 11.132 14.089L9.617 21.662C9.539 22.054 9.625 22.095 9.878 21.785L18.871 10.716C19.123 10.406 19.049 10.25 18.649 10.25H13.349C12.949 10.25 12.789 10.054 12.87 9.662Z"
        />
      </svg>
    ),
    "units-voltage",
  ),
};
