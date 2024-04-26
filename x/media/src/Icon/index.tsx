// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type FC } from "react";

import { type runtime } from "@synnaxlabs/x";
import clsx from "clsx";
import {
  AiFillCaretDown,
  AiFillCaretLeft,
  AiFillCaretRight,
  AiFillCaretUp,
  AiFillDelete,
  AiFillFolder,
  AiFillInfoCircle,
  AiFillWarning,
  AiOutlineBorder,
  AiOutlineCheck,
  AiOutlineClose,
  AiOutlineMinus,
  AiOutlineExpand,
  AiFillGithub,
  AiFillLinkedin,
  AiOutlineLoading,
  AiOutlineSync,
} from "react-icons/ai";
import { BiRename } from "react-icons/bi";
import { BsLightbulbFill, BsShiftFill, BsSave } from "react-icons/bs";
import {
  FaApple,
  FaBezierCurve,
  FaDocker,
  FaLinux,
  FaStream,
  FaWindows,
} from "react-icons/fa";
import { GrPan } from "react-icons/gr";
import { HiDownload, HiLightningBolt, HiOutlinePlus } from "react-icons/hi";
import { HiSquare3Stack3D } from "react-icons/hi2";
import { IoMdRefresh } from "react-icons/io";
import { IoBookSharp, IoCopySharp, IoTime } from "react-icons/io5";
import { GiHamburgerMenu } from "react-icons/gi";
import {
  MdEdit,
  MdOutlineDeviceHub,
  MdQuestionMark,
  MdSensors,
  MdWorkspacesFilled,
  MdAreaChart,
  MdEditOff,
  MdOutlineAccessTimeFilled,
  MdKeyboardAlt,
  MdKeyboardArrowDown,
  MdKeyboardArrowLeft,
  MdKeyboardArrowRight,
  MdKeyboardArrowUp,
  MdKeyboardBackspace,
  MdKeyboardCapslock,
  MdKeyboardHide,
  MdKeyboardReturn,
  MdKeyboardTab,
  MdKeyboardCommandKey,
  MdKeyboardOptionKey,
  MdFiberManualRecord,
  MdInsights,
  MdLabel,
  MdSquareFoot,
  MdKeyboardControlKey,
  MdPerson,
  MdPause,
  MdPlayArrow,
  MdPictureInPicture,
  MdAutoAwesome,
  MdOutlineTableRows,
  MdAlignHorizontalLeft,
  MdAlignHorizontalRight,
  MdAlignHorizontalCenter,
  MdAlignVerticalCenter,
  MdAlignVerticalTop,
  MdAlignVerticalBottom,
  MdHardware,
  MdSaveAlt,
} from "react-icons/md";
import { PiSelectionPlusBold, PiMagnifyingGlassBold } from "react-icons/pi";
import { RiSettings3Fill as RiSettingsFill, RiSoundModuleFill } from "react-icons/ri";
import { SiNpm, SiPnpm, SiPython, SiTypescript, SiYarn } from "react-icons/si";
import {
  TbArrowRight,
  TbArrowDown,
  TbArrowLeft,
  TbArrowUp,
  TbPlugConnected,
  TbPlugConnectedX,
  TbSubtask,
} from "react-icons/tb";

import "@/Icon/Icon.css";

export type IconProps = React.SVGProps<SVGSVGElement>;
type IconFC = FC<IconProps>;

export const NI: IconFC = (props) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    viewBox="0 0 78 51"
    fill="none"
    {...props}
    className={clsx(props.className, "logo")}
  >
    <g clipPath="url(#clip0_327_656)">
      <path
        d="M17.3026 17.0048V51.0427H0V17.0048H17.3026ZM34.6051 -0.0140575C36.8587 -0.0327593 39.0939 0.392621 41.1831 1.23779C43.2723 2.08297 45.1746 3.33138 46.7813 4.91175C48.388 6.49211 49.6677 8.37348 50.5473 10.4484C51.4269 12.5234 51.8891 14.7512 51.9077 17.0048V51.0427H34.6051V17.0048H17.3026V-0.0140575H34.6051ZM77.8615 -0.0140575V51.0427C75.6074 51.0632 73.3714 50.6391 71.2813 49.7946C69.1913 48.9501 67.2883 47.7018 65.6812 46.1211C64.0741 44.5403 62.7945 42.6582 61.9156 40.5824C61.0366 38.5066 60.5756 36.2779 60.559 34.0238V-0.0140575H77.8615Z"
        fill="#03B584"
      ></path>
    </g>
    <defs>
      <clipPath id="clip0_327_656">
        <rect width="77.8615" height="51" fill="white"></rect>
      </clipPath>
    </defs>
  </svg>
);

interface WrapIconOpts {
  className?: string;
}

export const wrapIcon =
  (Icon: IconFC, { className }: WrapIconOpts = {}): IconFC =>
  (props) => (
    <Icon {...props} className={clsx(props.className, className, "synnax-icon")} />
  );

const IconOS: Record<runtime.OS, IconFC> = {
  Linux: FaLinux,
  MacOS: FaApple,
  Windows: FaWindows,
  Docker: FaDocker,
};

export const Icon: IconType = {
  Pause: wrapIcon(MdPause),
  Play: wrapIcon(MdPlayArrow),
  Circle: wrapIcon(wrapIcon(MdFiberManualRecord)),
  Edit: wrapIcon(wrapIcon(MdEdit)),
  EditOff: wrapIcon(MdEditOff),
  Add: wrapIcon(HiOutlinePlus),
  Subtract: wrapIcon(AiOutlineMinus),
  Copy: wrapIcon(IoCopySharp),
  Close: wrapIcon(AiOutlineClose),
  Info: wrapIcon(AiFillInfoCircle),
  Warning: wrapIcon(AiFillWarning),
  Check: wrapIcon(AiOutlineCheck),
  Refresh: wrapIcon(IoMdRefresh),
  Delete: wrapIcon(AiFillDelete),
  Time: wrapIcon(IoTime),
  Acquire: wrapIcon(FaStream),
  Analyze: wrapIcon(FaBezierCurve),
  Concepts: wrapIcon(BsLightbulbFill),
  Visualize: wrapIcon(MdAreaChart),
  Expand: wrapIcon(AiOutlineExpand),
  Cluster: wrapIcon(HiSquare3Stack3D),
  Loading: wrapIcon(AiOutlineLoading, { className: "media--spin" }),
  PID: wrapIcon(FaStream),
  Caret: {
    Right: wrapIcon(AiFillCaretRight),
    Left: wrapIcon(AiFillCaretLeft),
    Up: wrapIcon(AiFillCaretUp),
    Down: wrapIcon(AiFillCaretDown),
  },
  Settings: wrapIcon(RiSettingsFill),
  Reference: wrapIcon(IoBookSharp),
  Bolt: wrapIcon(HiLightningBolt),
  Download: wrapIcon(HiDownload),
  Range: wrapIcon(MdOutlineAccessTimeFilled),
  Node: wrapIcon(MdOutlineDeviceHub),
  Channel: wrapIcon(MdSensors),
  Resources: wrapIcon(AiFillFolder),
  Group: wrapIcon(AiFillFolder),
  Workspace: wrapIcon(MdWorkspacesFilled),
  OS: {
    Linux: wrapIcon(IconOS.Linux),
    MacOS: wrapIcon(IconOS.MacOS),
    Windows: wrapIcon(IconOS.Windows),
    Docker: wrapIcon(IconOS.Docker),
  },
  Box: wrapIcon(AiOutlineBorder),
  Python: wrapIcon(SiPython),
  TypeScript: wrapIcon(SiTypescript),
  NPM: wrapIcon(SiNpm),
  PNPM: wrapIcon(SiPnpm),
  Yarn: wrapIcon(SiYarn),
  QuestionMark: wrapIcon(MdQuestionMark),
  Menu: wrapIcon(GiHamburgerMenu),
  Logo: {
    Github: wrapIcon(AiFillGithub),
    LinkedIn: wrapIcon(AiFillLinkedin),
    NI: wrapIcon(NI),
  },
  Arrow: {
    Right: wrapIcon(TbArrowRight),
    Down: wrapIcon(TbArrowDown),
    Up: wrapIcon(TbArrowUp),
    Left: wrapIcon(TbArrowLeft),
  },
  Keyboard: {
    Command: wrapIcon(MdKeyboardCommandKey),
    Windows: wrapIcon(FaWindows),
    Tab: wrapIcon(MdKeyboardTab),
    Return: wrapIcon(MdKeyboardReturn),
    Backspace: wrapIcon(MdKeyboardBackspace),
    Capslock: wrapIcon(MdKeyboardCapslock),
    Hide: wrapIcon(MdKeyboardHide),
    Control: wrapIcon(MdKeyboardControlKey),
    Arrow: {
      Up: wrapIcon(MdKeyboardArrowUp),
      Down: wrapIcon(MdKeyboardArrowDown),
      Left: wrapIcon(MdKeyboardArrowLeft),
      Right: wrapIcon(MdKeyboardArrowRight),
    },
    Alt: wrapIcon(MdKeyboardAlt),
    Option: wrapIcon(MdKeyboardOptionKey),
    Shift: wrapIcon(BsShiftFill),
  },
  Tooltip: wrapIcon(MdInsights),
  Annotate: wrapIcon(MdLabel),
  Zoom: wrapIcon(PiMagnifyingGlassBold),
  Selection: wrapIcon(PiSelectionPlusBold),
  Pan: wrapIcon(GrPan),
  Rule: wrapIcon(MdSquareFoot),
  User: wrapIcon(MdPerson),
  Rename: wrapIcon(BiRename),
  Snapshot: wrapIcon(MdPictureInPicture),
  Sync: wrapIcon(AiOutlineSync),
  Search: wrapIcon(PiMagnifyingGlassBold),
  Auto: wrapIcon(MdAutoAwesome),
  Table: wrapIcon(MdOutlineTableRows),
  Align: {
    Right: wrapIcon(MdAlignHorizontalRight),
    Left: wrapIcon(MdAlignHorizontalLeft),
    XCenter: wrapIcon(MdAlignHorizontalCenter),
    YCenter: wrapIcon(MdAlignVerticalCenter),
    Top: wrapIcon(MdAlignVerticalTop),
    Bottom: wrapIcon(MdAlignVerticalBottom),
  },
  Connect: wrapIcon(TbPlugConnected),
  Disconnect: wrapIcon(TbPlugConnectedX),
  Hardware: wrapIcon(MdHardware),
  Save: wrapIcon(MdSaveAlt),
  Task: wrapIcon(TbSubtask),
};

export interface IconType {
  Pause: IconFC;
  Play: IconFC;
  Circle: IconFC;
  Edit: IconFC;
  EditOff: IconFC;
  Add: IconFC;
  Subtract: IconFC;
  Copy: IconFC;
  Close: IconFC;
  Info: IconFC;
  Warning: IconFC;
  Check: IconFC;
  Refresh: IconFC;
  Delete: IconFC;
  Time: IconFC;
  Acquire: IconFC;
  Analyze: IconFC;
  Concepts: IconFC;
  Expand: IconFC;
  Visualize: IconFC;
  Cluster: IconFC;
  PID: IconFC;
  Settings: IconFC;
  Table: IconFC;
  Caret: {
    Right: IconFC;
    Left: IconFC;
    Up: IconFC;
    Down: IconFC;
  };
  Arrow: {
    Right: IconFC;
    Left: IconFC;
    Up: IconFC;
    Down: IconFC;
  };
  Reference: IconFC;
  Bolt: IconFC;
  Download: IconFC;
  Range: IconFC;
  Node: IconFC;
  Channel: IconFC;
  Resources: IconFC;
  Group: IconFC;
  Workspace: IconFC;
  OS: Record<runtime.OS, IconFC>;
  Box: IconFC;
  Python: IconFC;
  TypeScript: IconFC;
  NPM: IconFC;
  PNPM: IconFC;
  Yarn: IconFC;
  QuestionMark: IconFC;
  Menu: IconFC;
  Logo: {
    Github: IconFC;
    LinkedIn: IconFC;
    NI: IconFC;
  };
  Keyboard: {
    Command: IconFC;
    Windows: IconFC;
    Tab: IconFC;
    Return: IconFC;
    Backspace: IconFC;
    Capslock: IconFC;
    Hide: IconFC;
    Control: IconFC;
    Arrow: {
      Up: IconFC;
      Down: IconFC;
      Left: IconFC;
      Right: IconFC;
    };
    Alt: IconFC;
    Option: IconFC;
    Shift: IconFC;
  };
  Zoom: IconFC;
  Pan: IconFC;
  Selection: IconFC;
  Tooltip: IconFC;
  Annotate: IconFC;
  Rule: IconFC;
  User: IconFC;
  Rename: IconFC;
  Snapshot: IconFC;
  Loading: IconFC;
  Sync: IconFC;
  Search: IconFC;
  Auto: IconFC;
  Align: {
    Right: IconFC;
    Left: IconFC;
    XCenter: IconFC;
    YCenter: IconFC;
    Top: IconFC;
    Bottom: IconFC;
  };
  Hardware: IconFC;
  Task: IconFC;
  Save: IconFC;
  Connect: IconFC;
  Disconnect: IconFC;
}
