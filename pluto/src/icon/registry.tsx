// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/icon/registry.css";

import { deep } from "@synnaxlabs/x";
import clsx from "clsx";
import { cloneElement, type FC } from "react";
import {
  AiFillDelete,
  AiFillFolder,
  AiFillGithub,
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
import { BiLinkExternal, BiMath, BiRename } from "react-icons/bi";
import {
  BsBorderWidth,
  BsFillInfoSquareFill,
  BsLightbulbFill,
  BsShiftFill,
} from "react-icons/bs";
import {
  FaAlignCenter,
  FaAlignLeft,
  FaAlignRight,
  FaApple,
  FaBezierCurve,
  FaCarSide,
  FaCreativeCommonsZero,
  FaDocker,
  FaLinux,
  FaMicrophone,
  FaRegStar,
  FaStar,
  FaStream,
  FaWind,
  FaWindows,
} from "react-icons/fa";
import { FaBridge, FaGaugeHigh, FaGear } from "react-icons/fa6";
import { FiTable } from "react-icons/fi";
import { GiHamburgerMenu } from "react-icons/gi";
import { GoNumber } from "react-icons/go";
import { GrAttachment, GrDrag, GrPan, GrRotateRight } from "react-icons/gr";
import { HiDownload, HiLightningBolt, HiOutlinePlus } from "react-icons/hi";
import { HiSquare3Stack3D } from "react-icons/hi2";
import { IoMdHeart, IoMdRefresh } from "react-icons/io";
import {
  IoBookSharp,
  IoCopy,
  IoNotificationsOff,
  IoShapes,
  IoTerminal,
  IoTime,
} from "react-icons/io5";
import {
  MdAlignHorizontalCenter,
  MdAlignHorizontalLeft,
  MdAlignHorizontalRight,
  MdAlignVerticalBottom,
  MdAlignVerticalCenter,
  MdAlignVerticalTop,
  MdAreaChart,
  MdArrowOutward,
  MdAutoAwesome,
  MdBlurLinear,
  MdBook,
  MdCalendarToday,
  MdCommit,
  MdDataArray,
  MdDataObject,
  MdEdit,
  MdEditOff,
  MdFeedback,
  MdFiberManualRecord,
  MdFileUpload,
  MdFilterCenterFocus,
  MdHardware,
  MdHive,
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
  MdNewReleases,
  MdOutlineControlCamera,
  MdOutlineDeviceHub,
  MdOutlineExplore,
  MdOutlineFilterList,
  MdOutlineLinearScale,
  MdOutlineMap,
  MdOutlineMotionPhotosOff,
  MdOutlineMotionPhotosOn,
  MdOutlineOpenInNew,
  MdOutlineTableRows,
  MdOutlineTimelapse,
  MdOutlineVisibility,
  MdOutlineVisibilityOff,
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
  MdTextFields,
  MdTypeSpecimen,
  MdWorkspacesFilled,
} from "react-icons/md";
import {
  PiBinary,
  PiCaretDown,
  PiCaretLeft,
  PiCaretRight,
  PiCaretUpBold,
  PiDownloadSimple,
  PiMagnifyingGlassBold,
  PiSelectionPlusBold,
  PiThermometerSimpleFill,
  PiWaveSawtoothBold,
  PiWaveSineBold,
  PiWaveSquareBold,
  PiWaveTriangleBold,
} from "react-icons/pi";
import { RiSettings3Fill as RiSettingsFill, RiWeightFill } from "react-icons/ri";
import {
  SiGooglenearby,
  SiNpm,
  SiPnpm,
  SiPython,
  SiSpringCreators,
  SiTypescript,
  SiYarn,
} from "react-icons/si";
import {
  TbArrowAutofitWidth,
  TbArrowDown,
  TbArrowLeft,
  TbArrowRight,
  TbArrowUp,
  TbCircleDashed,
  TbCircleLetterAFilled,
  TbCircleLetterVFilled,
  TbCircuitResistor,
  TbDecimal,
  TbLivePhoto,
  TbMathFunction,
  TbPlugConnected,
  TbPlugConnectedX,
  TbRadarFilled,
  TbVariable,
} from "react-icons/tb";
import {
  VscSplitHorizontal,
  VscSplitVertical,
  VscSymbolConstant,
  VscSymbolString,
} from "react-icons/vsc";

import { CSS } from "@/css";
import {
  type IconProps,
  type ReactElement,
  type SVGFC,
  wrapSVGIcon,
} from "@/icon/Icon";

const LabJack: SVGFC = (props) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    viewBox="0 0 96 96"
    fill="currentColor"
    {...props}
  >
    <path
      d="M62.638 80.81C71.378 80.47 76.888 74.61 76.918 65.96C76.9247 65.0267 76.978 50.5434 77.078 22.51C77.078 22.3827 77.1275 22.2606 77.2157 22.1706C77.3038 22.0806 77.4234 22.03 77.548 22.03H94.358C94.533 22.03 94.7009 22.0996 94.8247 22.2233C94.9485 22.3471 95.018 22.515 95.018 22.69C95.018 25.0367 95.0547 37.6 95.128 60.38C95.1547 68.1334 94.648 73.4734 93.608 76.4C89.578 87.71 78.278 91.56 66.858 91.83C61.8647 91.95 52.098 92 37.558 91.98C37.3379 91.98 37.1268 91.8926 36.9711 91.7369C36.8155 91.5813 36.728 91.3702 36.728 91.15V81.26C36.728 81.1539 36.7701 81.0522 36.8452 80.9772C36.9202 80.9022 37.0219 80.86 37.128 80.86C48.1747 81.0534 56.678 81.0367 62.638 80.81Z"
      stroke="none"
    />
    <path
      d="M0.027809 24.91L0.117809 0.5C0.117809 0.166667 0.284476 0 0.617809 0H17.3078C17.4775 0 17.6403 0.0674271 17.7604 0.18745C17.8804 0.307474 17.9478 0.470261 17.9478 0.639999L18.0178 26.89C18.0578 30.75 18.0711 34.5999 18.0578 38.4399C18.0511 43.9999 18.4211 47.8066 19.1678 49.8599C20.5944 53.7399 23.2744 56.3866 27.2078 57.7999C29.2144 58.5199 32.7411 58.8966 37.7878 58.9299C43.8144 58.9632 50.3878 58.9799 57.5078 58.9799C57.8811 58.9799 58.0678 59.1666 58.0678 59.5399V69.3899C58.0678 69.7699 57.8744 69.9599 57.4878 69.9599C39.7478 70.0266 29.6077 70.0166 27.0678 69.9299C25.4478 69.8699 23.1078 69.5432 20.0478 68.9499C17.1544 68.3832 14.8978 67.7266 13.2778 66.9799C7.59109 64.3666 3.68442 60.3032 1.55775 54.7899C0.571086 52.2432 0.0577555 47.2932 0.0177555 39.9399C-0.0089112 34.9332 -0.00552432 29.9233 0.027809 24.91Z"
      stroke="none"
    />
  </svg>
);

const NI: SVGFC = (props) => (
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
      />
    </g>
    <defs>
      <clipPath id="clip0_327_656">
        <rect width="77.8615" height="51" fill="white" />
      </clipPath>
    </defs>
  </svg>
);

const OPC: SVGFC = ({ className, ...rest }) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    version="1.1"
    className={clsx(className, "logo")}
    {...rest}
    stroke="currentColor"
    fill="currentColor"
    viewBox="0 0 512 325.74567"
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

const Modbus: SVGFC = ({ className, ...rest }) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    viewBox="0 0 52 60"
    className={clsx(className, "logo")}
    {...rest}
    stroke="currentColor"
    fill="currentColor"
  >
    <circle cx="6" cy="18" r="6" />
    <circle cx="26" cy="6" r="6" />
    <circle cx="46" cy="18" r="6" />
    <circle cx="46" cy="39" r="6" />
    <circle cx="26" cy="51" r="6" />
    <circle cx="6" cy="39" r="6" />
    <path d="M26 30L23.1132 35H28.8868L26 30ZM26.5 51V34.5H25.5V51H26.5Z" />
    <path d="M26 28L28.8868 23H23.1132L26 28ZM25.5 6V23.5H26.5V6H25.5Z" />
    <path d="M24 30L18.2369 29.6541L20.8189 34.8181L24 30ZM6.22361 39.4472L20.1987 32.4597L19.7515 31.5652L5.77639 38.5528L6.22361 39.4472Z" />
    <path d="M28 30L31.1811 34.8181L33.7631 29.6541L28 30ZM46.2236 38.5528L32.2485 31.5652L31.8013 32.4597L45.7764 39.4472L46.2236 38.5528Z" />
    <path d="M28 28L33.7727 28.0953L30.9689 23.0483L28 28ZM45.7572 17.5629L31.6909 25.3775L32.1765 26.2517L46.2428 18.4371L45.7572 17.5629Z" />
    <path d="M24 28L21.0311 23.0483L18.2273 28.0953L24 28ZM5.75718 18.4371L19.8235 26.2517L20.3091 25.3775L6.24282 17.5629L5.75718 18.4371Z" />
  </svg>
);

export const Pause = wrapSVGIcon(MdPause, "pause");
export const Play = wrapSVGIcon(MdPlayArrow, "play");
export const Circle = wrapSVGIcon(MdFiberManualRecord, "circle");
export const Edit = wrapSVGIcon(MdEdit, "edit");
export const EditOff = wrapSVGIcon(MdEditOff, "edit-off");
export const Add = wrapSVGIcon(HiOutlinePlus, "add");
export const Subtract = wrapSVGIcon(AiOutlineMinus, "subtract");
export const Copy = wrapSVGIcon(IoCopy, "copy");
export const Close = wrapSVGIcon(AiOutlineClose, "close");
export const Info = wrapSVGIcon(BsFillInfoSquareFill, "info");
export const Warning = wrapSVGIcon(AiFillWarning, "warning");
export const Check = wrapSVGIcon(AiOutlineCheck, "check");
export const Refresh = wrapSVGIcon(IoMdRefresh, "refresh");
export const Delete = wrapSVGIcon(AiFillDelete, "delete");
export const Time = wrapSVGIcon(IoTime, "time");
export const Acquire = wrapSVGIcon(FaStream, "acquire");
export const Analyze = wrapSVGIcon(FaBezierCurve, "analyze");
export const Concepts = wrapSVGIcon(BsLightbulbFill, "concepts");
export const Visualize = wrapSVGIcon(MdAreaChart, "visualize");
export const LinePlot = wrapSVGIcon(MdAreaChart, "line-plot");
export const Expand = wrapSVGIcon(AiOutlineExpand, "expand");
export const Cluster = wrapSVGIcon(HiSquare3Stack3D, "cluster");
export const Loading = wrapSVGIcon(AiOutlineLoading, "loading", {
  className: CSS.M("spin"),
});
export const Schematic = wrapSVGIcon(IoShapes, "schematic");
export const Caret = {
  Right: wrapSVGIcon(PiCaretRight, "caret-right"),
  Bottom: wrapSVGIcon(PiCaretDown, "caret-bottom"),
  Left: wrapSVGIcon(PiCaretLeft, "caret-left"),
  Up: wrapSVGIcon(PiCaretUpBold, "caret-up"),
  Top: wrapSVGIcon(PiCaretUpBold, "caret-top"),
  Down: wrapSVGIcon(PiCaretDown, "caret-down"),
};
export const Settings = wrapSVGIcon(RiSettingsFill, "settings");
export const Reference = wrapSVGIcon(IoBookSharp, "reference");
export const Bolt = wrapSVGIcon(HiLightningBolt, "bolt");
export const Download = wrapSVGIcon(HiDownload, "download");
export const Import = wrapSVGIcon(MdFileUpload, "import");
export const Export = wrapSVGIcon(PiDownloadSimple, "export");
export const Range = wrapSVGIcon(MdOutlineTimelapse, "range");
export const Node = wrapSVGIcon(MdOutlineDeviceHub, "node");
export const Channel = wrapSVGIcon(MdSensors, "channel");
export const Resources = wrapSVGIcon(AiFillFolder, "resources");
export const Group = wrapSVGIcon(AiFillFolder, "group");
export const Workspace = wrapSVGIcon(MdWorkspacesFilled, "workspace");
export const Box = wrapSVGIcon(AiOutlineBorder, "box");
export const Python = wrapSVGIcon(SiPython, "python");
export const TypeScript = wrapSVGIcon(SiTypescript, "typescript");
export const NPM = wrapSVGIcon(SiNpm, "npm");
export const PNPM = wrapSVGIcon(SiPnpm, "pnpm");
export const Yarn = wrapSVGIcon(SiYarn, "yarn");
export const QuestionMark = wrapSVGIcon(MdQuestionMark, "question-mark");
export const Menu = wrapSVGIcon(GiHamburgerMenu, "menu");
export const Logo = {
  Apple: wrapSVGIcon(FaApple, "logo-apple"),
  Docker: wrapSVGIcon(FaDocker, "logo-docker"),
  Github: wrapSVGIcon(AiFillGithub, "logo-github"),
  LabJack: wrapSVGIcon(LabJack, "logo-labjack"),
  LinkedIn: wrapSVGIcon(AiFillLinkedin, "logo-linkedin"),
  Linux: wrapSVGIcon(FaLinux, "logo-linux"),
  NI: wrapSVGIcon(NI, "logo-ni"),
  OPC: wrapSVGIcon(OPC, "logo-opc"),
  Windows: wrapSVGIcon(FaWindows, "logo-windows"),
  Modbus: wrapSVGIcon(Modbus, "logo-modbus"),
};
export const Arrow = {
  Right: wrapSVGIcon(TbArrowRight, "arrow-right"),
  Down: wrapSVGIcon(TbArrowDown, "arrow-down"),
  Bottom: wrapSVGIcon(TbArrowDown, "arrow-bottom"),
  Up: wrapSVGIcon(TbArrowUp, "arrow-up"),
  Left: wrapSVGIcon(TbArrowLeft, "arrow-left"),
  Top: wrapSVGIcon(TbArrowUp, "arrow-top"),
};
export const Keyboard = {
  Command: wrapSVGIcon(MdKeyboardCommandKey, "keyboard-command"),
  Windows: wrapSVGIcon(FaWindows, "keyboard-windows"),
  Tab: wrapSVGIcon(MdKeyboardTab, "keyboard-tab"),
  Return: wrapSVGIcon(MdKeyboardReturn, "keyboard-return"),
  Backspace: wrapSVGIcon(MdKeyboardBackspace, "keyboard-backspace"),
  Capslock: wrapSVGIcon(MdKeyboardCapslock, "keyboard-capslock"),
  Hide: wrapSVGIcon(MdKeyboardHide, "keyboard-hide"),
  Control: wrapSVGIcon(MdKeyboardControlKey, "keyboard-control"),
  Arrow: {
    Up: wrapSVGIcon(MdKeyboardArrowUp, "keyboard-arrow-up"),
    Down: wrapSVGIcon(MdKeyboardArrowDown, "keyboard-arrow-down"),
    Left: wrapSVGIcon(MdKeyboardArrowLeft, "keyboard-arrow-left"),
    Right: wrapSVGIcon(MdKeyboardArrowRight, "keyboard-arrow-right"),
  },
  Alt: wrapSVGIcon(MdKeyboardAlt, "keyboard-alt"),
  Option: wrapSVGIcon(MdKeyboardOptionKey, "keyboard-option"),
  Shift: wrapSVGIcon(BsShiftFill, "keyboard-shift"),
};
export const Tooltip = wrapSVGIcon(MdInsights, "tooltip");
export const Annotate = wrapSVGIcon(MdLabel, "annotate");
export const Zoom = wrapSVGIcon(PiMagnifyingGlassBold, "zoom");
export const Selection = wrapSVGIcon(PiSelectionPlusBold, "selection");
export const Pan = wrapSVGIcon(GrPan, "pan");
export const Rule = wrapSVGIcon(MdSquareFoot, "rule");
export const User = wrapSVGIcon(MdPerson, "user");
export const Rename = wrapSVGIcon(BiRename, "rename");
export const Snapshot = wrapSVGIcon(MdPictureInPicture, "snapshot");
export const Sync = wrapSVGIcon(AiOutlineSync, "sync");
export const Search = wrapSVGIcon(PiMagnifyingGlassBold, "search");
export const Auto = wrapSVGIcon(MdAutoAwesome, "auto");
export const Table = wrapSVGIcon(FiTable, "table");
export const Wave = {
  Sawtooth: wrapSVGIcon(PiWaveSawtoothBold, "wave-sawtooth"),
  Sine: wrapSVGIcon(PiWaveSineBold, "wave-sine"),
  Triangle: wrapSVGIcon(PiWaveTriangleBold, "wave-triangle"),
  Square: wrapSVGIcon(PiWaveSquareBold, "wave-square"),
};
export const Align = {
  Right: wrapSVGIcon(MdAlignHorizontalRight, "align-right"),
  Left: wrapSVGIcon(MdAlignHorizontalLeft, "align-left"),
  XCenter: wrapSVGIcon(MdAlignHorizontalCenter, "align-x-center"),
  YCenter: wrapSVGIcon(MdAlignVerticalCenter, "align-y-center"),
  Top: wrapSVGIcon(MdAlignVerticalTop, "align-top"),
  Bottom: wrapSVGIcon(MdAlignVerticalBottom, "align-bottom"),
};
export const TextAlign = {
  Center: wrapSVGIcon(FaAlignCenter, "text-align-center"),
  Left: wrapSVGIcon(FaAlignLeft, "text-align-left"),
  Right: wrapSVGIcon(FaAlignRight, "text-align-right"),
};
export const Connect = wrapSVGIcon(TbPlugConnected, "connect");
export const Disconnect = wrapSVGIcon(TbPlugConnectedX, "disconnect");
export const Hardware = wrapSVGIcon(MdHardware, "hardware");
export const Save = wrapSVGIcon(MdSaveAlt, "save");
export const Task = wrapSVGIcon(TbRadarFilled, "task");
export const Device = wrapSVGIcon(SiGooglenearby, "device");
export const Link = wrapSVGIcon(MdLink, "link");
export const Attachment = wrapSVGIcon(GrAttachment, "attachment");
export const Drag = wrapSVGIcon(GrDrag, "drag");
export const Dynamic = wrapSVGIcon(TbLivePhoto, "dynamic");
export const Enable = wrapSVGIcon(MdOutlineMotionPhotosOn, "enable");
export const Disable = wrapSVGIcon(MdOutlineMotionPhotosOff, "disable");
export const Variable = wrapSVGIcon(TbVariable, "variable");
export const Type = wrapSVGIcon(MdTypeSpecimen, "type");
export const Array = wrapSVGIcon(MdDataArray, "array");
export const Label = wrapSVGIcon(MdLabel, "label");
export const Details = wrapSVGIcon(MdOutlineTableRows, "details");
export const LinkExternal = wrapSVGIcon(BiLinkExternal, "link-external");
export const Access = wrapSVGIcon(MdShield, "access");
export const JSON = wrapSVGIcon(MdDataObject, "json");
export const Guide = wrapSVGIcon(MdBook, "guide");
export const Focus = wrapSVGIcon(MdFilterCenterFocus, "focus");
export const OpenInNewWindow = wrapSVGIcon(MdOutlineOpenInNew, "open-in-new-window");
export const MoveToMainWindow = wrapSVGIcon(MdOutlineWebAsset, "move-to-main-window");
export const SplitX = wrapSVGIcon(VscSplitHorizontal, "split-x");
export const SplitY = wrapSVGIcon(VscSplitVertical, "split-y");
export const AutoFitWidth = wrapSVGIcon(TbArrowAutofitWidth, "auto-fit-width");
export const Commit = wrapSVGIcon(MdCommit, "commit");
export const Snooze = wrapSVGIcon(IoNotificationsOff, "snooze");
export const Log = wrapSVGIcon(FaStream, "log");
export const Tare = wrapSVGIcon(FaCreativeCommonsZero, "tare");
export const Rotate = wrapSVGIcon(GrRotateRight, "rotate");
export const Text = wrapSVGIcon(MdTextFields, "text");
export const Value = wrapSVGIcon(GoNumber, "value");
export const Calendar = wrapSVGIcon(MdCalendarToday, "calendar");
export const Release = wrapSVGIcon(MdNewReleases, "release");
export const OpenExternal = wrapSVGIcon(MdArrowOutward, "open-external");
export const Feedback = wrapSVGIcon(MdFeedback, "feedback");
export const Calculation = wrapSVGIcon(BiMath, "calculation");
export const Binary = wrapSVGIcon(PiBinary, "binary");
export const Index = wrapSVGIcon(IoTime, "index");
export const Decimal = wrapSVGIcon(TbDecimal, "decimal");
export const String = wrapSVGIcon(VscSymbolString, "string");
export const Control = wrapSVGIcon(MdOutlineControlCamera, "control");
export const Rack = wrapSVGIcon(MdHive, "rack");
export const Units = {
  Acceleration: wrapSVGIcon(FaCarSide, "units-acceleration"),
  Current: wrapSVGIcon(TbCircleLetterAFilled, "units-current"),
  Force: wrapSVGIcon(RiWeightFill, "units-force"),
  Pressure: wrapSVGIcon(FaGaugeHigh, "units-pressure"),
  Resistance: wrapSVGIcon(TbCircuitResistor, "units-resistance"),
  Strain: wrapSVGIcon(SiSpringCreators, "units-strain"),
  Temperature: wrapSVGIcon(PiThermometerSimpleFill, "units-temperature"),
  Torque: wrapSVGIcon(FaGear, "units-torque"),
  Velocity: wrapSVGIcon(FaWind, "units-velocity"),
  Voltage: wrapSVGIcon(TbCircleLetterVFilled, "units-voltage"),
};
export const Bridge = wrapSVGIcon(FaBridge, "bridge");
export const Sound = wrapSVGIcon(FaMicrophone, "sound");
export const Function = wrapSVGIcon(TbMathFunction, "function");
export const Visible = wrapSVGIcon(MdOutlineVisibility, "visible");
export const Hidden = wrapSVGIcon(MdOutlineVisibilityOff, "invisible");
export const Virtual = wrapSVGIcon(TbCircleLetterVFilled, "virtual");
export const Explore = wrapSVGIcon(MdOutlineExplore, "explore");
export const Filter = wrapSVGIcon(MdOutlineFilterList, "filter");
export const StarFilled = wrapSVGIcon(FaStar, "star-filled");
export const StarOutlined = wrapSVGIcon(FaRegStar, "star-outlined");
export const Heart = wrapSVGIcon(IoMdHeart, "heart");
export const Map = wrapSVGIcon(MdOutlineMap, "map");
export const Linear = wrapSVGIcon(MdOutlineLinearScale, "linear");
export const None = wrapSVGIcon(TbCircleDashed, "none");
export const Constant = wrapSVGIcon(VscSymbolConstant, "constant");
export const StrokeWidth = wrapSVGIcon(BsBorderWidth, "stroke-width");
export const Downsample = wrapSVGIcon(MdBlurLinear, "downsample");
export const Terminal = wrapSVGIcon(IoTerminal, "terminal");

export interface CreateProps extends Omit<IconProps, "topRight"> {}

interface Resolve {
  (icon?: ReactElement | string, overrides?: IconProps): ReactElement | undefined;
  (icon: ReactElement | string, overrides?: IconProps): ReactElement;
}

const icons = {
  Pause,
  Play,
  Circle,
  Edit,
  EditOff,
  Add,
  Subtract,
  Copy,
  Close,
  Info,
  Warning,
  Check,
  Refresh,
  Delete,
  Time,
  Acquire,
  Analyze,
  Concepts,
  Visualize,
  LinePlot,
  Expand,
  Cluster,
  Loading,
  Schematic,
  Caret,
  Settings,
  Reference,
  Bolt,
  Download,
  Import,
  Export,
  Range,
  Node,
  Channel,
  Resources,
  Group,
  Workspace,
  Box,
  Python,
  TypeScript,
  NPM,
  PNPM,
  Yarn,
  QuestionMark,
  Menu,
  Logo,
  Arrow,
  Keyboard,
  Tooltip,
  Annotate,
  Zoom,
  Selection,
  Pan,
  Rule,
  User,
  Rename,
  Snapshot,
  Sync,
  Search,
  Auto,
  Table,
  Wave,
  Align,
  TextAlign,
  Connect,
  Disconnect,
  Hardware,
  Save,
  Task,
  Device,
  Link,
  Attachment,
  Drag,
  Dynamic,
  Enable,
  Disable,
  Variable,
  Type,
  Array,
  Label,
  Details,
  LinkExternal,
  Access,
  JSON,
  Guide,
  Focus,
  OpenInNewWindow,
  MoveToMainWindow,
  SplitX,
  SplitY,
  AutoFitWidth,
  Commit,
  Snooze,
  Log,
  Tare,
  Rotate,
  Text,
  Value,
  Calendar,
  Release,
  OpenExternal,
  Feedback,
  Calculation,
  Binary,
  Index,
  Decimal,
  String,
  Control,
  Rack,
  Units,
  Bridge,
  Sound,
  Function,
  Visible,
  Hidden,
  Virtual,
  Explore,
  Filter,
  StarFilled,
  StarOutlined,
  Heart,
  Map,
  Linear,
  None,
  Constant,
  Terminal,
};

export const resolve = ((
  icon?: ReactElement | string | undefined,
  overrides?: IconProps,
): ReactElement | undefined => {
  if (icon == null) return;
  if (typeof icon === "string")
    try {
      const C = deep.get<FC<IconProps>>(icons, icon);
      return <C {...overrides} />;
    } catch {
      throw new Error(`Unable to find icon with path ${icon} in registry`);
    }

  return cloneElement(icon, overrides);
}) as Resolve;
