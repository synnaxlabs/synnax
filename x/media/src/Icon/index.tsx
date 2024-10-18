// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/Icon/Icon.css";

import { type runtime } from "@synnaxlabs/x";
import clsx from "clsx";
import { type FC } from "react";
import {
  AiFillDelete,
  AiFillFolder,
  AiFillGithub,
  AiFillInfoCircle,
  AiFillLinkedin,
  AiFillWarning,
  AiOutlineBorder,
  AiOutlineCheck,
  AiOutlineClose,
  AiOutlineExpand,
  AiOutlineLoading,
  AiOutlineMinus,
  AiOutlineSync,
} from "react-icons/ai";
import { BiLinkExternal, BiRename } from "react-icons/bi";
import { BsLightbulbFill, BsShiftFill } from "react-icons/bs";
import {
  FaAlignCenter,
  FaAlignLeft,
  FaAlignRight,
  FaApple,
  FaBezierCurve,
  FaBookOpen,
  FaDocker,
  FaLinux,
  FaStream,
  FaWindows,
} from "react-icons/fa";
import { GiHamburgerMenu } from "react-icons/gi";
import { GrAttachment, GrDrag, GrPan } from "react-icons/gr";
import { HiDownload, HiLightningBolt, HiOutlinePlus } from "react-icons/hi";
import { HiSquare3Stack3D } from "react-icons/hi2";
import { IoMdRefresh, IoMdReturnRight } from "react-icons/io";
import { IoBookSharp, IoCopy, IoTime } from "react-icons/io5";
import {
  MdAlignHorizontalCenter,
  MdAlignHorizontalLeft,
  MdAlignHorizontalRight,
  MdAlignVerticalBottom,
  MdAlignVerticalCenter,
  MdAlignVerticalTop,
  MdAreaChart,
  MdAutoAwesome,
  MdBook,
  MdCommit,
  MdDataArray,
  MdDataObject,
  MdEdit,
  MdEditOff,
  MdFiberManualRecord,
  MdFileUpload,
  MdFilterCenterFocus,
  MdHardware,
  MdInsights,
  MdKeyboardAlt,
  MdKeyboardArrowDown,
  MdKeyboardArrowLeft,
  MdKeyboardArrowRight,
  MdKeyboardArrowUp,
  MdKeyboardBackspace,
  MdKeyboardCapslock,
  MdKeyboardCommandKey,
  MdKeyboardControlKey,
  MdKeyboardHide,
  MdKeyboardOptionKey,
  MdKeyboardReturn,
  MdKeyboardTab,
  MdLabel,
  MdLink,
  MdOutlineAccessTimeFilled,
  MdOutlineDeviceHub,
  MdOutlineMotionPhotosOff,
  MdOutlineMotionPhotosOn,
  MdOutlineOpenInNew,
  MdOutlineTableRows,
  MdOutlineWebAsset,
  MdPause,
  MdPerson,
  MdPictureInPicture,
  MdPlayArrow,
  MdQuestionMark,
  MdSaveAlt,
  MdSensors,
  MdShield,
  MdSquareFoot,
  MdTypeSpecimen,
  MdWorkspacesFilled,
} from "react-icons/md";
import {
  PiCaretDown,
  PiCaretLeft,
  PiCaretRight,
  PiCaretUpBold,
  PiDownloadSimple,
  PiMagnifyingGlassBold,
  PiSelectionPlusBold,
} from "react-icons/pi";
import { RiSettings3Fill as RiSettingsFill } from "react-icons/ri";
import {
  SiGooglenearby,
  SiNpm,
  SiPnpm,
  SiPython,
  SiTypescript,
  SiYarn,
} from "react-icons/si";
import {
  TbArrowAutofitWidth,
  TbArrowDown,
  TbArrowLeft,
  TbArrowRight,
  TbArrowUp,
  TbLivePhoto,
  TbPlugConnected,
  TbPlugConnectedX,
  TbRadarFilled,
  TbVariable,
} from "react-icons/tb";
import { VscSplitHorizontal, VscSplitVertical } from "react-icons/vsc";

export interface IconProps extends React.SVGProps<SVGSVGElement> {}
type IconFC = FC<IconProps>;

export const NI: IconFC = (props) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    viewBox="0 0 78 51"
    {...props}
    className={clsx(props.className, "logo")}
    stroke="currentColor"
    fill="currentColor"
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

const OPC: IconFC = ({ className, style, ...props }) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    version="1.1"
    className={clsx(className, "logo")}
    {...props}
    stroke="currentColor"
    fill="currentColor"
    viewBox="0 0 512 325.74567"
    style={{
      // Runs a little small
      transform: "scale(1.25)",
      ...style,
    }}
  >
    <g transform="translate(-151.28831,-121.30134)" id="layer1">
      <g transform="matrix(2.7387317,0,0,2.7387317,-938.60269,-766.86241)" id="g3606">
        <path
          d="m 405.36669,375.73826 c 4.0225,-3.7175 9.16125,-7.18625 15.20125,-10.32375 -0.74875,-2.4425 -1.085,-5.12 -1.085,-7.64375 0,-7.5075 2.73,-16.49375 11.71625,-16.49375 8.5125,0 11.5125,8.06125 11.80625,15.2925 10.53625,-3.17125 22.36875,-5.64 35.15375,-7.165 2.115,-6.09125 6.77375,-8.1275 10.68125,-8.1275 4.88875,0 7.94625,2.72625 9.6675,6.4675 3.5775,-0.1425 7.2475,-0.15875 10.915,-0.15875 9.17001,0 18.09501,0.40125 26.64001,1.25875 1.5525,-4.33625 4.76,-7.5675 11.15875,-7.5675 4.89125,0 9.32875,3.52625 9.4425,8.53125 l 18.76875,0 c -1.36625,-15.585 -14.4475,-23.32 -29.12125,-23.32 -14.0825,0 -24.29,7.3775 -28.6275,18.84125 -3.36625,-10.705 -11.15626,-18.84125 -24.40876,-18.84125 -6.825,0 -12.51125,2.95875 -16.265,8.5325 l -0.2275,0 0,-6.82625 -18.4275,0 0,13.57625 c -5.16625,-9.26625 -14.9475,-15.2825 -27.15625,-15.2825 -18.3125,0 -31.05375,13.5375 -31.05375,31.2825 0,6.79375 1.89,12.95125 5.22125,17.9675"
          id="path3299"
          fillRule="nonzero"
          fillOpacity={1}
        />
        <path
          d="m 426.05169,372.92451 c 1.42625,0.78 3.13,1.2275 5.1475,1.2275 6.175,0 9.45875,-4.1875 10.9025,-9.24375 10.05,-4.0125 21.88125,-7.235 34.94875,-9.4375 -0.0275,0.5225 -0.04,1.06125 -0.04,1.6175 0,10.3525 2.9575,17.06375 11.71625,17.06375 8.645,0 11.6025,-8.98625 11.6025,-17.06375 0,-1.42875 -0.0937,-2.88625 -0.32,-4.31 5.86,-0.40625 11.865,-0.61625 18.00001,-0.61625 5.82375,0 11.54375,0.19125 17.1225,0.5575 -0.30875,1.95125 -0.4225,3.98625 -0.4225,5.9625 0,7.62125 3.3,15.47 11.48875,15.47 6.7125,0 10.125,-3.8675 11.03375,-10.12375 l 18.76875,0 c -1.705,15.81125 -13.99125,24.9125 -29.68875,24.9125 -13.22625,0 -23.925,-6.76375 -28.51625,-17.6975 -3.2875,10.0525 -10.92501,17.6975 -23.04001,17.6975 -7.28,0 -12.85375,-2.73125 -16.83625,-8.0775 l -0.2275,0 0,28.92 -19.33625,0 0,-36.04 c -5.16625,9.2325 -14.9475,15.1975 -27.15625,15.1975 -6.8475,0 -12.8975,-1.8775 -17.8125,-5.155 3.10875,-3.875 7.38125,-7.51875 12.665,-10.86125"
          id="path3301"
          fillOpacity={1}
          fillRule="nonzero"
        />
        <path
          d="m 582.7117,362.38451 c -19.015,-7.0775 -43.9375,-11.36875 -71.22126,-11.36875 -59.675,0 -108.05,20.525 -108.05,45.84375 0,21.07875 33.53,38.83125 79.19375,44.1875 -41.985,-6.12625 -72.15125,-22.9375 -72.15125,-42.7175 0,-24.96 48.005,-45.19375 107.22251,-45.19375 24.4475,0 46.97,3.4425 65.00625,9.24875"
          id="path3303"
          fillOpacity={1}
          fillRule="nonzero"
        />
      </g>
    </g>
  </svg>
);

interface WrapIconOpts {
  className?: string;
}

export const wrapIcon = (
  Icon: IconFC,
  name: string,
  { className }: WrapIconOpts = {},
): IconFC => {
  const O: IconFC = (props) => (
    <Icon
      {...props}
      className={clsx(props.className, `synnax-icon-${name}`, className, "synnax-icon")}
    />
  );
  O.displayName = Icon.displayName || Icon.name;
  return O;
};

const IconOS: Record<runtime.OS, IconFC> = {
  Linux: FaLinux,
  MacOS: FaApple,
  Windows: FaWindows,
  Docker: FaDocker,
};

export const Icon: IconType = {
  Pause: wrapIcon(MdPause, "pause"),
  Play: wrapIcon(MdPlayArrow, "play"),
  Circle: wrapIcon(MdFiberManualRecord, "circle"),
  Edit: wrapIcon(MdEdit, "edit"),
  EditOff: wrapIcon(MdEditOff, "edit-off"),
  Add: wrapIcon(HiOutlinePlus, "add"),
  Subtract: wrapIcon(AiOutlineMinus, "subtract"),
  Copy: wrapIcon(IoCopy, "copy"),
  Close: wrapIcon(AiOutlineClose, "close"),
  Info: wrapIcon(AiFillInfoCircle, "info"),
  Warning: wrapIcon(AiFillWarning, "warning"),
  Check: wrapIcon(AiOutlineCheck, "check"),
  Refresh: wrapIcon(IoMdRefresh, "refresh"),
  Delete: wrapIcon(AiFillDelete, "delete"),
  Time: wrapIcon(IoTime, "time"),
  Acquire: wrapIcon(FaStream, "acquire"),
  Analyze: wrapIcon(FaBezierCurve, "analyze"),
  Concepts: wrapIcon(BsLightbulbFill, "concepts"),
  Visualize: wrapIcon(MdAreaChart, "visualize"),
  LinePlot: wrapIcon(MdAreaChart, "line-plot"),
  Expand: wrapIcon(AiOutlineExpand, "expand"),
  Cluster: wrapIcon(HiSquare3Stack3D, "cluster"),
  Loading: wrapIcon(AiOutlineLoading, "loading", { className: "media--spin" }),
  Schematic: wrapIcon(FaStream, "schematic"),
  Caret: {
    Right: wrapIcon(PiCaretRight, "caret-right"),
    Bottom: wrapIcon(PiCaretDown, "caret-bottom"),
    Left: wrapIcon(PiCaretLeft, "caret-left"),
    Up: wrapIcon(PiCaretUpBold, "caret-up"),
    Top: wrapIcon(PiCaretUpBold, "caret-top"),
    Down: wrapIcon(PiCaretDown, "caret-down"),
  },
  Settings: wrapIcon(RiSettingsFill, "settings"),
  Reference: wrapIcon(IoBookSharp, "reference"),
  Bolt: wrapIcon(HiLightningBolt, "bolt"),
  Download: wrapIcon(HiDownload, "download"),
  Import: wrapIcon(MdFileUpload, "import"),
  Export: wrapIcon(PiDownloadSimple, "export"),
  Range: wrapIcon(MdOutlineAccessTimeFilled, "range"),
  Node: wrapIcon(MdOutlineDeviceHub, "node"),
  Channel: wrapIcon(MdSensors, "channel"),
  Resources: wrapIcon(AiFillFolder, "resources"),
  Group: wrapIcon(AiFillFolder, "group"),
  Workspace: wrapIcon(MdWorkspacesFilled, "workspace"),
  OS: {
    Linux: wrapIcon(IconOS.Linux, "os-linux"),
    MacOS: wrapIcon(IconOS.MacOS, "os-macos"),
    Windows: wrapIcon(IconOS.Windows, "os-windows"),
    Docker: wrapIcon(IconOS.Docker, "os-docker"),
  },
  Box: wrapIcon(AiOutlineBorder, "box"),
  Python: wrapIcon(SiPython, "python"),
  TypeScript: wrapIcon(SiTypescript, "typescript"),
  NPM: wrapIcon(SiNpm, "npm"),
  PNPM: wrapIcon(SiPnpm, "pnpm"),
  Yarn: wrapIcon(SiYarn, "yarn"),
  QuestionMark: wrapIcon(MdQuestionMark, "question-mark"),
  Menu: wrapIcon(GiHamburgerMenu, "menu"),
  Logo: {
    Github: wrapIcon(AiFillGithub, "logo-github"),
    LinkedIn: wrapIcon(AiFillLinkedin, "logo-linkedin"),
    NI: wrapIcon(NI, "logo-ni"),
    OPC: wrapIcon(OPC, "logo-opc"),
  },
  Arrow: {
    Right: wrapIcon(TbArrowRight, "arrow-right"),
    Down: wrapIcon(TbArrowDown, "arrow-down"),
    Bottom: wrapIcon(TbArrowDown, "arrow-bottom"),
    Up: wrapIcon(TbArrowUp, "arrow-up"),
    Left: wrapIcon(TbArrowLeft, "arrow-left"),
    Top: wrapIcon(TbArrowUp, "arrow-top"),
  },
  Keyboard: {
    Command: wrapIcon(MdKeyboardCommandKey, "keyboard-command"),
    Windows: wrapIcon(FaWindows, "keyboard-windows"),
    Tab: wrapIcon(MdKeyboardTab, "keyboard-tab"),
    Return: wrapIcon(MdKeyboardReturn, "keyboard-return"),
    Backspace: wrapIcon(MdKeyboardBackspace, "keyboard-backspace"),
    Capslock: wrapIcon(MdKeyboardCapslock, "keyboard-capslock"),
    Hide: wrapIcon(MdKeyboardHide, "keyboard-hide"),
    Control: wrapIcon(MdKeyboardControlKey, "keyboard-control"),
    Arrow: {
      Up: wrapIcon(MdKeyboardArrowUp, "keyboard-arrow-up"),
      Down: wrapIcon(MdKeyboardArrowDown, "keyboard-arrow-down"),
      Left: wrapIcon(MdKeyboardArrowLeft, "keyboard-arrow-left"),
      Right: wrapIcon(MdKeyboardArrowRight, "keyboard-arrow-right"),
    },
    Alt: wrapIcon(MdKeyboardAlt, "keyboard-alt"),
    Option: wrapIcon(MdKeyboardOptionKey, "keyboard-option"),
    Shift: wrapIcon(BsShiftFill, "keyboard-shift"),
  },
  Tooltip: wrapIcon(MdInsights, "tooltip"),
  Annotate: wrapIcon(MdLabel, "annotate"),
  Zoom: wrapIcon(PiMagnifyingGlassBold, "zoom"),
  Selection: wrapIcon(PiSelectionPlusBold, "selection"),
  Pan: wrapIcon(GrPan, "pan"),
  Rule: wrapIcon(MdSquareFoot, "rule"),
  User: wrapIcon(MdPerson, "user"),
  Rename: wrapIcon(BiRename, "rename"),
  Snapshot: wrapIcon(MdPictureInPicture, "snapshot"),
  Sync: wrapIcon(AiOutlineSync, "sync"),
  Search: wrapIcon(PiMagnifyingGlassBold, "search"),
  Auto: wrapIcon(MdAutoAwesome, "auto"),
  Table: wrapIcon(MdOutlineTableRows, "table"),
  Align: {
    Right: wrapIcon(MdAlignHorizontalRight, "align-right"),
    Left: wrapIcon(MdAlignHorizontalLeft, "align-left"),
    XCenter: wrapIcon(MdAlignHorizontalCenter, "align-x-center"),
    YCenter: wrapIcon(MdAlignVerticalCenter, "align-y-center"),
    Top: wrapIcon(MdAlignVerticalTop, "align-top"),
    Bottom: wrapIcon(MdAlignVerticalBottom, "align-bottom"),
  },
  TextAlign: {
    Center: wrapIcon(FaAlignCenter, "text-align-center"),
    Left: wrapIcon(FaAlignLeft, "text-align-left"),
    Right: wrapIcon(FaAlignRight, "text-align-right"),
  },
  Connect: wrapIcon(TbPlugConnected, "connect"),
  Disconnect: wrapIcon(TbPlugConnectedX, "disconnect"),
  Hardware: wrapIcon(MdHardware, "hardware"),
  Save: wrapIcon(MdSaveAlt, "save"),
  Task: wrapIcon(TbRadarFilled, "task"),
  Device: wrapIcon(SiGooglenearby, "device"),
  Link: wrapIcon(MdLink, "link"),
  Attachment: wrapIcon(GrAttachment, "attachment"),
  Drag: wrapIcon(GrDrag, "drag"),
  Dynamic: wrapIcon(TbLivePhoto, "dynamic"),
  Enable: wrapIcon(MdOutlineMotionPhotosOn, "enable"),
  Disable: wrapIcon(MdOutlineMotionPhotosOff, "disable"),
  Variable: wrapIcon(TbVariable, "variable"),
  Object: wrapIcon(MdDataObject, "object"),
  Type: wrapIcon(MdTypeSpecimen, "type"),
  Array: wrapIcon(MdDataArray, "array"),
  Label: wrapIcon(MdLabel, "label"),
  Details: wrapIcon(MdOutlineTableRows, "details"),
  LinkExternal: wrapIcon(BiLinkExternal, "link-external"),
  Access: wrapIcon(MdShield, "access"),
  JSON: wrapIcon(MdDataObject, "json"),
  Guide: wrapIcon(MdBook, "guide"),
  Focus: wrapIcon(MdFilterCenterFocus, "focus"),
  OpenInNewWindow: wrapIcon(MdOutlineOpenInNew, "open-in-new-window"),
  MoveToMainWindow: wrapIcon(MdOutlineWebAsset, "move-to-main-window"),
  SplitX: wrapIcon(VscSplitHorizontal, "split-x"),
  SplitY: wrapIcon(VscSplitVertical, "split-y"),
  AutoFitWidth: wrapIcon(TbArrowAutofitWidth, "auto-fit-width"),
  ReturnRight: wrapIcon(IoMdReturnRight, "return-right"),
  Commit: wrapIcon(MdCommit, "commit"),
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
  Schematic: IconFC;
  Settings: IconFC;
  ReturnRight: IconFC;
  Table: IconFC;
  Caret: {
    Right: IconFC;
    Left: IconFC;
    Up: IconFC;
    Down: IconFC;
    Top: IconFC;
    Bottom: IconFC;
  };
  Arrow: {
    Right: IconFC;
    Left: IconFC;
    Up: IconFC;
    Down: IconFC;
    Top: IconFC;
    Bottom: IconFC;
  };
  Reference: IconFC;
  Bolt: IconFC;
  Download: IconFC;
  Import: IconFC;
  Export: IconFC;
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
    OPC: IconFC;
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
  TextAlign: {
    Left: IconFC;
    Center: IconFC;
    Right: IconFC;
  };
  Hardware: IconFC;
  Device: IconFC;
  Task: IconFC;
  Save: IconFC;
  Connect: IconFC;
  Disconnect: IconFC;
  Drag: IconFC;
  Link: IconFC;
  Attachment: IconFC;
  Disable: IconFC;
  Enable: IconFC;
  Dynamic: IconFC;
  Variable: IconFC;
  Object: IconFC;
  Type: IconFC;
  LinePlot: IconFC;
  Array: IconFC;
  Label: IconFC;
  Details: IconFC;
  LinkExternal: IconFC;
  Access: IconFC;
  JSON: IconFC;
  Guide: IconFC;
  Focus: IconFC;
  OpenInNewWindow: IconFC;
  MoveToMainWindow: IconFC;
  SplitX: IconFC;
  SplitY: IconFC;
  AutoFitWidth: IconFC;
  Commit: IconFC;
}
