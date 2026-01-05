// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * Name: LabJackM.h
 * Desc: Header file describing C-style exposed
 *       API functions for the LabJackM Library
 * Auth: LabJack Corp.
**/

#ifndef LAB_JACK_M_HEADER
#define LAB_JACK_M_HEADER

#define LJM_VERSION 1.2302
	// Format: xx.yyzz
	// xx is the major version (left of the decimal).
	// yy is the minor version (the two places to the right of the decimal).
	// zz is the revision version (the two places to the right of the minor
	//    version).

/******************************************************************************
 * How To Use This Library:
 *
 * See the LJM User's Guide: labjack.com/support/ljm/users-guide
 *
 * Check out the example files for examples of workflow
 *
 * To write/read other Modbus addresses, check out labjack.com/support/modbus
 *
 *****************************************************************************/

#define LJM_ERROR_CODE static const int

#ifdef __cplusplus
extern "C" {
#endif

#ifdef _WIN32
	#define LJM_ERROR_RETURN int __stdcall
	#define LJM_LONG_LONG_RETURN long long __stdcall
	#define LJM_VOID_RETURN void __stdcall
	#define LJM_ERROR_STRING const char * __stdcall
	#define LJM_DOUBLE_RETURN double __stdcall
#else
	#ifdef __APPLE__
		#define LJM_ERROR_RETURN int
		#define LJM_LONG_LONG_RETURN long long
		#define LJM_VOID_RETURN void
		#define LJM_ERROR_STRING const char *
		#define LJM_DOUBLE_RETURN double
	#else // Linux
		#ifdef LABJACK_LJM_ATTRIBUTE_VISIBILITY_DEFAULT
			#define LJM_ERROR_RETURN __attribute__((__visibility__("default"))) int
			#define LJM_LONG_LONG_RETURN __attribute__((__visibility__("default"))) long long
			#define LJM_VOID_RETURN __attribute__((__visibility__("default"))) void
			#define LJM_ERROR_STRING __attribute__((__visibility__("default"))) const char *
			#define LJM_DOUBLE_RETURN __attribute__((__visibility__("default"))) double
		#else
			#define LJM_ERROR_RETURN int
			#define LJM_LONG_LONG_RETURN long long
			#define LJM_VOID_RETURN void
			#define LJM_ERROR_STRING const char *
			#define LJM_DOUBLE_RETURN double
		#endif
	#endif
#endif


/*************
 * Constants *
 *************/

// Read/Write direction constants:
static const int LJM_READ = 0;
static const int LJM_WRITE = 1;

// Data Types:
// These do automatic endianness conversion, if needed by the local machine's
// processor.
static const int LJM_UINT16 = 0; // C type of unsigned short
static const int LJM_UINT32 = 1; // C type of unsigned int
static const int LJM_INT32 = 2; // C type of int
static const int LJM_FLOAT32 = 3; // C type of float

// Advanced users data types:
//     These do not do any endianness conversion.
static const int LJM_BYTE = 99; // Contiguous bytes. If the number of LJM_BYTEs is
                         // odd, the last, (least significant) byte is 0x00.
                         // For example, for 3 LJM_BYTES of values
                         // [0x01, 0x02, 0x03], LJM sends the contiguous byte
                         // array [0x01, 0x02, 0x03, 0x00]
static const int LJM_STRING = 98; // Same as LJM_BYTE, but LJM automatically appends
                           // a null-terminator.

static const unsigned int LJM_STRING_MAX_SIZE = 49;
// Max LJM_STRING size not including the automatic null-terminator

// Max LJM_STRING size with the null-terminator
enum { LJM_STRING_ALLOCATION_SIZE = 50 };

// LJM_NamesToAddresses uses this when a register name is not found
static const int LJM_INVALID_NAME_ADDRESS = -1;

enum { LJM_MAX_NAME_SIZE = 256 };

// 18 = 6 * 2 (number of byte chars) + 5 (number of colons) + 1 (null-terminator)
enum { LJM_MAC_STRING_SIZE = 18 };

// 16 is INET_ADDRSTRLEN
enum { LJM_IPv4_STRING_SIZE = 16 };

static const int LJM_BYTES_PER_REGISTER = 2;

// Device types:
enum {
	LJM_dtANY = 0,
	LJM_dtT4 = 4,
	LJM_dtT7 = 7,
	LJM_dtT8 = 8,
	LJM_dtDIGIT = 200,
	LJM_dtTSERIES = 84
};

// Connection types:
enum {
	LJM_ctANY = 0,
	LJM_ctANY_TCP = LJM_ctANY,

	LJM_ctUSB = 1,

	// TCP
	LJM_ctTCP = 2,
	LJM_ctNETWORK_TCP = LJM_ctTCP,
	LJM_ctETHERNET = 3,
	LJM_ctETHERNET_TCP = LJM_ctETHERNET,
	LJM_ctWIFI = 4,
	LJM_ctWIFI_TCP = LJM_ctWIFI,

	// UDP
	LJM_ctANY_UDP = 11,
	LJM_ctNETWORK_UDP = 5,
	LJM_ctETHERNET_UDP = 6,
	LJM_ctWIFI_UDP = 7,

	// TCP or UDP
	LJM_ctNETWORK_ANY = 8,
	LJM_ctETHERNET_ANY = 9,
	LJM_ctWIFI_ANY = 10
};

// Network constants:
static const int LJM_TCP_PORT = 502;
static const int LJM_ETHERNET_UDP_PORT = 52362;
static const int LJM_WIFI_UDP_PORT = 502;
static const int LJM_NO_IP_ADDRESS = 0;
static const int LJM_NO_PORT = 0;

// Identifier types:
static const char * const LJM_DEMO_MODE = "-2";
static const int LJM_idANY = 0;

// LJM_AddressesToMBFB Constants
enum { LJM_DEFAULT_FEEDBACK_ALLOCATION_SIZE = 62 };
static const int LJM_USE_DEFAULT_MAXBYTESPERMBFB = 0;

// LJM_MBFBComm Constants;
static const int LJM_DEFAULT_UNIT_ID = 1;

// LJM_ListAll Constants
enum { LJM_LIST_ALL_SIZE = 128 };

// Timeout Constants. Times in milliseconds.
static const int LJM_NO_TIMEOUT = 0;
static const int LJM_DEFAULT_USB_SEND_RECEIVE_TIMEOUT_MS = 2600;
static const int LJM_DEFAULT_ETHERNET_OPEN_TIMEOUT_MS = 1000;
static const int LJM_DEFAULT_ETHERNET_SEND_RECEIVE_TIMEOUT_MS = 2600;
static const int LJM_DEFAULT_WIFI_OPEN_TIMEOUT_MS = 1000;
static const int LJM_DEFAULT_WIFI_SEND_RECEIVE_TIMEOUT_MS = 4000;

// Stream Constants
static const int LJM_DUMMY_VALUE = -9999;
static const int LJM_SCAN_NOT_READ = -8888;
static const int LJM_GND = 199;

/*****************************************************************************
* Return Values                                                              *
* Success:                                                                   *
*     Constant: LJME_NOERROR                                                 *
*     Description: The function executed without error.                      *
*     Range: 0                                                               *
*                                                                            *
* Warnings:                                                                  *
*     Prefix: LJME_                                                          *
*     Description: Some or all outputs might be valid.                       *
*     Range: 200-399                                                         *
*                                                                            *
* Modbus Errors:                                                             *
*     Prefix: LJME_MBE                                                       *
*     Description: Errors corresponding to official Modbus errors which are  *
*         returned from the device.                                          *
*     Note: To find the original Modbus error in base 10, subtract 1200.     *
*     Ranges: 1200-1216                                                      *
*                                                                            *
* Library Errors:                                                            *
*     Prefix: LJME_                                                          *
*     Description: Errors where all outputs are null, invalid, 0, or 9999.   *
*     Range: 1220-1399                                                       *
*                                                                            *
* Device Errors:                                                             *
*     Description: Errors returned from the firmware on the device.          *
*     Range: 2000-2999                                                       *
*                                                                            *
* User Area:                                                                 *
*     Description: Errors defined by users.                                  *
*     Range: 3900-3999                                                       *
*                                                                            */

// Success
LJM_ERROR_CODE LJME_NOERROR = 0;

// Warnings:
LJM_ERROR_CODE LJME_WARNINGS_BEGIN = 200;
LJM_ERROR_CODE LJME_WARNINGS_END = 399;
LJM_ERROR_CODE LJME_FRAMES_OMITTED_DUE_TO_PACKET_SIZE = 201;
	// Functions:
	//     LJM_AddressesToMBFB:
	//         Problem: This indicates that the length (in bytes) of the Feedback
	//             command being created was greater than the value passed as
	//             MaxBytesPerMBFB. As a result, the command returned is a valid
	//             Feedback command that includes some of the frames originally
	//             specified, but not all of them. You can check the NumFrames
	//             pointer to find out how many frames were included.
	//         Solutions:
	//             1) Pass a larger value for MaxBytesPerMBFB and make sure
	//                aMBFBCommand has memory allocated of size MaxBytesPerMBFB.
	//                The default size for MaxBytesPerMBFB is 64.
	//             2) Split the command into multiple commands.
	//     Any other function that creates a Feedback command:
	//         Problem: The Feedback command being created was too large for
	//             the device to handle on this connection type.
	//         Solution: Split the command into multiple commands.

LJM_ERROR_CODE LJME_DEBUG_LOG_FAILURE = 202;
LJM_ERROR_CODE LJME_USING_DEFAULT_CALIBRATION = 203;
	// Problem: LJM has detected the device has one or more invalid calibration
	//     constants and is using the default calibration constants. Readings may
	//     inaccurate.
	// Solution: Contact LabJack support.

LJM_ERROR_CODE LJME_DEBUG_LOG_FILE_NOT_OPEN = 204;

// Modbus Errors:
LJM_ERROR_CODE LJME_MODBUS_ERRORS_BEGIN = 1200;
LJM_ERROR_CODE LJME_MODBUS_ERRORS_END = 1216;
LJM_ERROR_CODE LJME_MBE1_ILLEGAL_FUNCTION = 1201;
LJM_ERROR_CODE LJME_MBE2_ILLEGAL_DATA_ADDRESS = 1202;
LJM_ERROR_CODE LJME_MBE3_ILLEGAL_DATA_VALUE = 1203;
LJM_ERROR_CODE LJME_MBE4_SLAVE_DEVICE_FAILURE = 1204;
LJM_ERROR_CODE LJME_MBE5_ACKNOWLEDGE = 1205;
LJM_ERROR_CODE LJME_MBE6_SLAVE_DEVICE_BUSY = 1206;
LJM_ERROR_CODE LJME_MBE8_MEMORY_PARITY_ERROR = 1208;
LJM_ERROR_CODE LJME_MBE10_GATEWAY_PATH_UNAVAILABLE = 1210;
LJM_ERROR_CODE LJME_MBE11_GATEWAY_TARGET_NO_RESPONSE = 1211;

// Library Errors:
LJM_ERROR_CODE LJME_LIBRARY_ERRORS_BEGIN = 1220;
LJM_ERROR_CODE LJME_LIBRARY_ERRORS_END = 1399;

LJM_ERROR_CODE LJME_UNKNOWN_ERROR = 1221;
LJM_ERROR_CODE LJME_INVALID_DEVICE_TYPE = 1222;
LJM_ERROR_CODE LJME_INVALID_HANDLE = 1223;
LJM_ERROR_CODE LJME_DEVICE_NOT_OPEN = 1224;
LJM_ERROR_CODE LJME_STREAM_NOT_INITIALIZED = 1225;
LJM_ERROR_CODE LJME_DEVICE_DISCONNECTED = 1226;
LJM_ERROR_CODE LJME_DEVICE_NOT_FOUND = 1227;
LJM_ERROR_CODE LJME_APERIODIC_STREAM_OUT_NOT_INITIALIZED = 1228;
LJM_ERROR_CODE LJME_DEVICE_ALREADY_OPEN = 1229;
LJM_ERROR_CODE LJME_DEVICE_CURRENTLY_CLAIMED_BY_ANOTHER_PROCESS = 1230;
LJM_ERROR_CODE LJME_CANNOT_CONNECT = 1231;
LJM_ERROR_CODE LJME_STREAM_OUT_INDEX_OUT_OF_RANGE = 1232;
LJM_ERROR_CODE LJME_SOCKET_LEVEL_ERROR = 1233;
LJM_ERROR_CODE LJME_SCAN_RATE_INCONSISTENT = 1234;
LJM_ERROR_CODE LJME_CANNOT_OPEN_DEVICE = 1236;
LJM_ERROR_CODE LJME_CANNOT_DISCONNECT = 1237;
LJM_ERROR_CODE LJME_WINSOCK_FAILURE = 1238;
LJM_ERROR_CODE LJME_RECONNECT_FAILED = 1239;
LJM_ERROR_CODE LJME_CONNECTION_HAS_YIELDED_RECONNECT_FAILED = 1240;
LJM_ERROR_CODE LJME_USB_FAILURE = 1241;
LJM_ERROR_CODE LJME_STREAM_FLUSH_TIMEOUT = 1242;

/* LJM does not support U3, U6, UE9, or U12 devices */
LJM_ERROR_CODE LJME_U3_NOT_SUPPORTED_BY_LJM = 1243;
LJM_ERROR_CODE LJME_U6_NOT_SUPPORTED_BY_LJM = 1246;
LJM_ERROR_CODE LJME_UE9_NOT_SUPPORTED_BY_LJM = 1249;

LJM_ERROR_CODE LJME_INVALID_ADDRESS = 1250;
LJM_ERROR_CODE LJME_INVALID_CONNECTION_TYPE = 1251;
LJM_ERROR_CODE LJME_INVALID_DIRECTION = 1252;
LJM_ERROR_CODE LJME_INVALID_FUNCTION = 1253;
	// Function: LJM_MBFBComm
	//     Problem: The aMBFB buffer passed as an input parameter
	//         did not have a function number corresponding to Feedback.
	//     Solution: Make sure the 8th byte of your buffer is 76 (base 10).
	//         (For example, aMBFB[7] == 76 should evaluate to true.)

LJM_ERROR_CODE LJME_INVALID_NUM_REGISTERS = 1254;
LJM_ERROR_CODE LJME_INVALID_PARAMETER = 1255;
LJM_ERROR_CODE LJME_INVALID_PROTOCOL_ID = 1256;
	// Problem: The Protocol ID was not in the proper range.

LJM_ERROR_CODE LJME_INVALID_TRANSACTION_ID = 1257;
	// Problem: The Transaction ID was not in the proper range.

LJM_ERROR_CODE LJME_NUM_WRITES_LARGER_THAN_AVAILABLE_SPACE = 1258;
	// Problem: tried to write more values than are available in write-out queue

LJM_ERROR_CODE LJME_UNKNOWN_VALUE_TYPE = 1259;

LJM_ERROR_CODE LJME_MEMORY_ALLOCATION_FAILURE = 1260;
	// Problem: A memory allocation attempt has failed, probably due to a
	//     lack of available memory.

LJM_ERROR_CODE LJME_NO_COMMAND_BYTES_SENT = 1261;
	// Problem: No bytes could be sent to the device.
	//     Possibilities:
	//         * The device was previously connected, but was suddenly
	//           disconnected.

LJM_ERROR_CODE LJME_INCORRECT_NUM_COMMAND_BYTES_SENT = 1262;
	// Problem: The expected number of bytes could not be sent to the device.
	//     Possibilities:
	//         * The device was disconnected while bytes were being sent.

LJM_ERROR_CODE LJME_NO_RESPONSE_BYTES_RECEIVED = 1263;
	// Problem: No bytes could be received from the device.
	//     Possibilities:
	//         * The device was previously connected, but was suddenly
	//           disconnected.
	//         * The timeout length was too short for the device to respond.

LJM_ERROR_CODE LJME_INCORRECT_NUM_RESPONSE_BYTES_RECEIVED = 1264;
	// Problem: The expected number of bytes could not be received from the
	//          device.
	//     Possibilities:
	//         * The device was previously connected, but was suddenly
	//           disconnected.
	//         * The device needs a firmware update.

LJM_ERROR_CODE LJME_MIXED_FORMAT_IP_ADDRESS = 1265;
	// Functions: LJM_OpenS and LJM_Open
	//     Problem: The string passed as an identifier contained an IP address
	//         that was ambiguous.
	//     Solution: Make sure the IP address is in either decimal format
	//         (i.e. "192.168.1.25") or hex format (i.e. "0xC0.A8.0.19").

LJM_ERROR_CODE LJME_UNKNOWN_IDENTIFIER = 1266;
LJM_ERROR_CODE LJME_NOT_IMPLEMENTED = 1267;
LJM_ERROR_CODE LJME_INVALID_INDEX = 1268;
	// Problem: An error internal to the LabJackM Library has occurred.
	// Solution: Please report this error to LabJack.

LJM_ERROR_CODE LJME_INVALID_LENGTH = 1269;
LJM_ERROR_CODE LJME_ERROR_BIT_SET = 1270;

LJM_ERROR_CODE LJME_INVALID_MAXBYTESPERMBFB = 1271;
	// Functions:
	//     LJM_AddressesToMBFB:
	//         Problem: This indicates the MaxBytesPerMBFB value was
	//             insufficient for any Feedback command.
	//         Solution: Pass a larger value for MaxBytesPerMBFB and make sure
	//             aMBFBCommand has memory allocated of size MaxBytesPerMBFB.
	//             The default size for MaxBytesPerMBFB is 64.

LJM_ERROR_CODE LJME_NULL_POINTER = 1272;
	// Problem: The Library has received an invalid pointer.
	// Solution: Make sure that any functions that have pointers in their
	//     parameter list are valid pointers that point to allocated memory.

LJM_ERROR_CODE LJME_NULL_OBJ = 1273;
	// Functions:
	//     LJM_OpenS and LJM_Open:
	//         Problem: The Library failed to parse the input parameters.
	//         Solution: Check the validity of your inputs and if the problem
	//             persists, please contact LabJack support.

LJM_ERROR_CODE LJME_RESERVED_NAME = 1274;
	// LJM_OpenS and LJM_Open:
	//     Problem: The string passed as Identifier was a reserved name.
	//     Solution: Use a different name for your device. You can also connect
	//         by passing the device's serial number or IP address, if
	//         applicable.

LJM_ERROR_CODE LJME_UNPARSABLE_DEVICE_TYPE = 1275;
	// LJM_OpenS:
	//     Problem: This Library could not parse the DeviceType.
	//     Solution: Check the LJM_OpenS documentation and make sure the
	//         DeviceType does not contain any unusual characters.

LJM_ERROR_CODE LJME_UNPARSABLE_CONNECTION_TYPE = 1276;
	// LJM_OpenS:
	//     Problem: This Library could not parse the ConnectionType.
	//     Solution: Check the LJM_OpenS documentation and make sure the
	//         ConnectionType does not contain any unusual characters.

LJM_ERROR_CODE LJME_UNPARSABLE_IDENTIFIER = 1277;
	// LJM_OpenS and LJM_Open:
	//     Problem: This Library could not parse the Identifier.
	//     Solution: Check the LJM_OpenS documentation and make sure the
	//         Identifier does not contain any unusual characters.

LJM_ERROR_CODE LJME_PACKET_SIZE_TOO_LARGE = 1278;
	// Problems: The packet being sent to the device contained too many bytes.
	// Note: Some LabJack devices need two bytes appended to any Modbus packets
	//       sent to a device. The packet size plus these two appended bytes
	//       could have exceeded the packet size limit.
	// Solution: Send a smaller packet, i.e. break your packet up into multiple
	//       packets.


LJM_ERROR_CODE LJME_TRANSACTION_ID_ERR = 1279;
	// Problem: LJM received an unexpected Modbus Transaction ID.

LJM_ERROR_CODE LJME_PROTOCOL_ID_ERR = 1280;
	// Problem: LJM received an unexpected Modbus Protocol ID.

LJM_ERROR_CODE LJME_LENGTH_ERR = 1281;
	// Problem: LJM received a packet with an unexpected Modbus Length.

LJM_ERROR_CODE LJME_UNIT_ID_ERR = 1282;
	// Problem: LJM received a packet with an unexpected Modbus Unit ID.

LJM_ERROR_CODE LJME_FUNCTION_ERR = 1283;
	// Problem: LJM received a packet with an unexpected Modbus Function.

LJM_ERROR_CODE LJME_STARTING_REG_ERR = 1284;
	// Problem: LJM received a packet with an unexpected Modbus address.

LJM_ERROR_CODE LJME_NUM_REGS_ERR = 1285;
	// Problem: LJM received a packet with an unexpected Modbus number of
	//     registers.

LJM_ERROR_CODE LJME_NUM_BYTES_ERR = 1286;
	// Problem: LJM received a packet with an unexpected Modbus number of bytes.

LJM_ERROR_CODE LJME_CONFIG_FILE_NOT_FOUND = 1289;
LJM_ERROR_CODE LJME_CONFIG_PARSING_ERROR = 1290;

LJM_ERROR_CODE LJME_INVALID_NUM_VALUES = 1291;
LJM_ERROR_CODE LJME_CONSTANTS_FILE_NOT_FOUND = 1292;
LJM_ERROR_CODE LJME_INVALID_CONSTANTS_FILE = 1293;
LJM_ERROR_CODE LJME_INVALID_NAME = 1294;
	// Problem: LJM received a name that was not found/matched in the constants
	//          file or was otherwise an invalid name.
	// Solution: Use LJM_ErrorToString to find the invalid name(s).

LJM_ERROR_CODE LJME_OVERSPECIFIED_PORT = 1296;
	// Functions: LJM_Open, LJM_OpenS
	//     Problem: LJM received an Identifier that specified a port/pipe, but
	//              connection type was not specified.

LJM_ERROR_CODE LJME_INTENT_NOT_READY = 1297;
	// Please contact LabJack support if the problem is not apparent.

LJM_ERROR_CODE LJME_ATTR_LOAD_COMM_FAILURE = 1298;
/**
 * Name: LJME_ATTR_LOAD_COMM_FAILURE
 * Functions: LJM_Open, LJM_OpenS
 * Desc: Indicates that a device was found and opened, but communication with
 *       that device failed, so the device was closed. The handle returned is
 *       not a valid handle. This communication failure can mean the device is
 *       in a non-responsive state or has out-of-date firmware.
 * Solutions: a) Power your device off, then back on, i.e. unplug it then plug
 *               it back in.
 *            b) Make sure your device(s) have up-to-date firmware.
**/

LJM_ERROR_CODE LJME_INVALID_CONFIG_NAME = 1299;
	// Functions: LJM_WriteLibraryConfigS, LJM_WriteLibraryConfigStringS,
	//            LJM_ReadLibraryConfigS, LJM_ReadLibraryConfigStringS
	//     Problem: An unknown string has been passed in as Parameter.
	//     Solution: Please check the documentation in this header file for the
	//         configuration parameter you are trying to read or write. Not all
	//         config parameters can be read, nor can all config parameters be
	//         written.

LJM_ERROR_CODE LJME_ERROR_RETRIEVAL_FAILURE = 1300;
	// Problem: A device has reported an error and LJM failed to to retrieve the
	//     error code from the device.
	// Solution: Please make sure the device has current firmware and that this
	//     is a current of LJM. If the problem persists, please contact LabJack
	//     support.

LJM_ERROR_CODE LJME_LJM_BUFFER_FULL = 1301;
LJM_ERROR_CODE LJME_COULD_NOT_START_STREAM = 1302;
LJM_ERROR_CODE LJME_STREAM_NOT_RUNNING = 1303;
LJM_ERROR_CODE LJME_UNABLE_TO_STOP_STREAM = 1304;
LJM_ERROR_CODE LJME_INVALID_VALUE = 1305;
LJM_ERROR_CODE LJME_SYNCHRONIZATION_TIMEOUT = 1306;
LJM_ERROR_CODE LJME_OLD_FIRMWARE = 1307;
LJM_ERROR_CODE LJME_CANNOT_READ_OUT_ONLY_STREAM = 1308;
LJM_ERROR_CODE LJME_NO_SCANS_RETURNED = 1309;
LJM_ERROR_CODE LJME_TEMPERATURE_OUT_OF_RANGE = 1310;
LJM_ERROR_CODE LJME_VOLTAGE_OUT_OF_RANGE = 1311;

LJM_ERROR_CODE LJME_FUNCTION_DOES_NOT_SUPPORT_THIS_TYPE = 1312;
	// Desc: The function does not support the given data type. For example,
	//       LJM_eReadName and LJM_eReadAddress do not support reading
	//       LJM_STRING values, which are too large.

LJM_ERROR_CODE LJME_INVALID_INFO_HANDLE = 1313;

LJM_ERROR_CODE LJME_NO_DEVICES_FOUND = 1314;
	// Desc: An Open/OpenS call was called - with any device type, any
	//       connection type, and any identifier - but no devices were found.

LJM_ERROR_CODE LJME_AUTO_IPS_FILE_NOT_FOUND = 1316;
LJM_ERROR_CODE LJME_AUTO_IPS_FILE_INVALID = 1317;

LJM_ERROR_CODE LJME_INVALID_INTERVAL_HANDLE = 1318;

LJM_ERROR_CODE LJME_NAMED_MUTEX_PERMISSION_DENIED = 1319;

LJM_ERROR_CODE LJME_DIGITAL_AUTO_RECOVERY_ERROR_DETECTED = 1320;
	// Desc: During stream, the device buffer overflowed, causing auto-recovery
	// to occur. However, the first channel of stream was not compatible with
	// auto-recovery. To avoid this error, either:
	//   1. Use one of the following channels as the first stream channel:
	//        An analog input (AIN0, AIN1, ...)
	//        FIO_STATE
	//        EIO_STATE
	//        CIO_STATE
	//        MIO_STATE
	//        EIO_CIO_STATE
	//        CIO_MIO_STATE
	// or:
	//   2. Ensure that the first stream channel cannot return 0xFFFF and set
	//      LJM_STREAM_DIGITAL_AUTO_RECOVERY_ERROR_DETECTION_DISABLED to 1.
	// See labjack.com/digital-auto-recovery-error-detection

LJM_ERROR_CODE LJME_NEGATIVE_RECEIVE_BUFFER_SIZE = 1321;
	// Problem: During stream, the receive buffer size
	// was negative. This is probably because
	// LJM_STREAM_TCP_RECEIVE_BUFFER_SIZE was set to too large a number to be represented
	// by the signed data type being used, int.
	// Solution: Use a smaller LJM_STREAM_TCP_RECEIVE_BUFFER_SIZE.


/*******************************
 * Device Management Functions *
 *******************************/

/**
 * Name: LJM_ListAll, LJM_ListAllS
 * Desc: Scans for LabJack devices, returning arrays describing the devices
 *       found, allowing LJM_dtANY and LJM_ctANY to be used
 * Para: DeviceType, filters which devices will be returned (LJM_dtT7,
 *           LJM_dtDIGIT, etc.). LJM_dtANY is allowed.
 *       ConnectionType, filters by connection type (LJM_ctUSB or LJM_ctTCP).
 *           LJM_ctANY is allowed.
 *       NumFound, a pointer that returns the number of devices found
 *       aDeviceTypes, an array that must be preallocated to size
 *           LJM_LIST_ALL_SIZE, returns the device type for each of the
 *           NumFound devices
 *       aConnectionTypes, an array that must be preallocated to size
 *           LJM_LIST_ALL_SIZE, returns the connect type for each of the
 *           NumFound devices
 *       aSerialNumbers, an array that must be preallocated to size
 *           LJM_LIST_ALL_SIZE, returns the serial number for each of the
 *           NumFound devices
 *       aIPAddresses, an array that must be preallocated to size
 *           LJM_LIST_ALL_SIZE, returns the IPAddresses for each of the
 *           NumFound devices, but only if ConnectionType is TCP-based. For
 *           each corresponding device for which aConnectionTypes[i] is not
 *           TCP-based, aIPAddresses[i] will be LJM_NO_IP_ADDRESS.
 * Note: These functions only show what devices could be opened. To actually
 *       open a device, use LJM_Open or LJM_OpenS.
 * Note: These functions will ignore NULL pointers, except for NumFound.
**/
LJM_ERROR_RETURN LJM_ListAll(int DeviceType, int ConnectionType,
	int * NumFound, int * aDeviceTypes, int * aConnectionTypes,
	int * aSerialNumbers, int * aIPAddresses);

LJM_ERROR_RETURN LJM_ListAllS(const char * DeviceType, const char * ConnectionType,
	int * NumFound, int * aDeviceTypes, int * aConnectionTypes,
	int * aSerialNumbers, int * aIPAddresses);

/**
 * Name: LJM_ListAllExtended
 * Desc: Advanced version of LJM_ListAll that performs an additional query of
 *       arbitrary registers on the device.
 * Para: DeviceType, filters which devices will be returned (LJM_dtT7,
 *           LJM_dtDIGIT, etc.). LJM_dtANY is allowed.
 *       ConnectionType, filters by connection type (LJM_ctUSB or LJM_ctTCP).
 *           LJM_ctANY is allowed.
 *       NumAddresses, the number of addresses to query. Also the size of
 *           aAddresses and aNumRegs.
 *       aAddresses, the addresses to query for each device that is found.
 *       aNumRegs, the number of registers to query for each address.
 *           Each aNumRegs[i] corresponds to aAddresses[i].
 *       MaxNumFound, the maximum number of devices to find. Also the size of
 *           aDeviceTypes, aConnectionTypes, aSerialNumbers, and aIPAddresses.
 *       NumFound, a pointer that returns the number of devices found
 *       aDeviceTypes, an array that must be preallocated to size
 *           MaxNumFound, returns the device type for each of the
 *           NumFound devices
 *       aConnectionTypes, an array that must be preallocated to size
 *           MaxNumFound, returns the connect type for each of the
 *           NumFound devices
 *       aSerialNumbers, an array that must be preallocated to size
 *           MaxNumFound, returns the serial number for each of the
 *           NumFound devices
 *       aIPAddresses, an array that must be preallocated to size
 *           MaxNumFound, returns the IPAddresses for each of the
 *           NumFound devices, but only if ConnectionType is TCP-based. For
 *           each corresponding device for which aConnectionTypes[i] is not
 *           TCP-based, aIPAddresses[i] will be LJM_NO_IP_ADDRESS.
 *       aBytes, an array that must be preallocated to size:
 *               MaxNumFound * <the sum of aNumRegs> * LJM_BYTES_PER_REGISTER,
 *           which will contain the query bytes sequentially. A device
 *           represented by index i would have an aBytes index of:
 *               (i * <the sum of aNumRegs> * LJM_BYTES_PER_REGISTER).
 * Note: These functions only show what devices could be opened. To actually
 *       open a device, use LJM_Open or LJM_OpenS.
 * Note: These functions will ignore NULL pointers, except for NumFound and
 *       aBytes.
**/
LJM_ERROR_RETURN LJM_ListAllExtended(int DeviceType, int ConnectionType,
	int NumAddresses, const int * aAddresses, const int * aNumRegs,
	int MaxNumFound, int * NumFound, int * aDeviceTypes, int * aConnectionTypes,
	int * aSerialNumbers, int * aIPAddresses, unsigned char * aBytes);

/**
 * Name: LJM_OpenS
 * Desc: Opens a LabJack device.
 * Para: DeviceType, a string containing the type of the device to be connected,
 *           optionally prepended by "LJM_dt". Possible values include "ANY",
 *           "T4", "T7", and "T8".
 *       ConnectionType, a string containing the type of the connection desired,
 *           optionally prepended by "LJM_ct". Possible values include "ANY",
 *           "USB", "TCP", "ETHERNET", and "WIFI".
 *       Identifier, a string identifying the device to be connected or
 *           "LJM_idANY"/"ANY". This can be a serial number, IP address, or
 *           device name. Device names may not contain periods.
 *       Handle, the new handle that represents a device connection upon success
 * Retr: LJME_NOERROR, if a device was successfully opened.
 *       LJME_ATTR_LOAD_COMM_FAILURE, if a device was found, but there was a
 *           communication failure.
 * Note: Input parameters are not case-sensitive.
 * Note: Empty strings passed to DeviceType, ConnectionType, or Identifier
 *           indicate the same thing as LJM_dtANY, LJM_ctANY, or LJM_idANY,
 *           respectively.
**/
LJM_ERROR_RETURN LJM_OpenS(const char * DeviceType, const char * ConnectionType,
	const char * Identifier, int * Handle);

/**
 * Name: LJM_Open
 * Desc: See the description for LJM_OpenS. The only difference between
 *       LJM_Open and LJM_OpenS is the first two parameters.
 * Para: DeviceType, a constant corresponding to the type of device to open,
 *           such as LJM_dtT7, or LJM_dtANY.
 *       ConnectionType, a constant corresponding to the type of connection to
 *           open, such as LJM_ctUSB, or LJM_ctANY.
**/
LJM_ERROR_RETURN LJM_Open(int DeviceType, int ConnectionType,
	const char * Identifier, int * Handle);

/**
 * Name: LJM_GetHandleInfo
 * Desc: Takes a device handle as input and returns details about that device.
 * Para: Handle, a valid handle to an open device.
 *       DeviceType, the output device type corresponding to a constant such as
 *           LJM_dtT7.
 *       ConnectionType, the output device type corresponding to a constant
 *           such as LJM_ctUSB.
 *       SerialNumber, the output serial number of the device.
 *       IPAddress, the output integer representation of the device's IP
 *           address when ConnectionType is TCP-based. If ConnectionType is not
 *           TCP-based, this will be LJM_NO_IP_ADDRESS. Note that this can be
 *           converted to a human-readable string with the LJM_NumberToIP
 *           function.
 *       Port, the output port if the device connection is TCP-based, or the pipe
 *           if the device connection is USB-based.
 *       MaxBytesPerMB, the maximum packet size in number of bytes that can be
 *           sent to or received from this device. Note that this can change
 *           depending on connection type and device type.
 * Note: This function returns device information loaded during an open call
 *       and therefore does not initiate communications with the device. In
 *       other words, it is fast but will not represent changes to serial
 *       number or IP address since the device was opened.
 * Warn: This function ignores null pointers
**/
LJM_ERROR_RETURN LJM_GetHandleInfo(int Handle, int * DeviceType,
	int * ConnectionType, int * SerialNumber, int * IPAddress, int * Port,
	int * MaxBytesPerMB);

/**
 * Name: LJM_Close
 * Desc: Closes the connection to the device.
 * Para: Handle, a valid handle to an open device.
**/
LJM_ERROR_RETURN LJM_Close(int Handle);

/**
 * Name: LJM_CloseAll
 * Desc: Closes all connections to all devices
**/
LJM_ERROR_RETURN LJM_CloseAll(void);

/**
 * Name: LJM_CleanInfo
 * Desc: Cleans/deallocates an InfoHandle.
 * Para: InfoHandle, The info handle to clean/deallocate.
 * Note: Calling LJM_CleanInfo on the same handle twice will return the error
 *       LJME_INVALID_INFO_HANDLE.
**/
LJM_ERROR_RETURN LJM_CleanInfo(int InfoHandle);


/******************************
 *  Easy Read/Write Functions *
 ******************************/

// Easy Functions: All type, either reading or writing, single address
/**
 * Name: LJM_eReadAddress, LJM_eReadName
 *       LJM_eWriteAddress, LJM_eWriteName
 * Desc: Creates and sends a Modbus operation, then receives and parses the
 *       response.
 * Para: Handle, a valid handle to an open device
 *       (Address), an address to read/write
 *       (Type), the type corresponding to Address
 *       (Name), a name to read/write
 *       Value, a value to write or read
 * Note: These functions may take liberties in deciding what kind of Modbus
 *       operation to create. For more control of what kind of packets may be
 *       sent/received, please see the LJM_WriteLibraryConfigS function.
**/
LJM_ERROR_RETURN LJM_eWriteAddress(int Handle, int Address, int Type, double Value);
LJM_ERROR_RETURN LJM_eReadAddress(int Handle, int Address, int Type, double * Value);

LJM_ERROR_RETURN LJM_eWriteName(int Handle, const char * Name, double Value);
LJM_ERROR_RETURN LJM_eReadName(int Handle, const char * Name, double * Value);

// Easy Functions: All type, either reading or writing, multiple addresses
/**
 * Name: LJM_eReadAddresses, LJM_eReadNames
 *       LJM_eWriteAddresses, LJM_eWriteNames
 * Desc: Creates and sends a Modbus operation, then receives and parses the
 *       response.
 * Para: Handle, a valid handle to an open device.
 *       NumFrames, the total number of reads/writes to perform.
 *       (aAddresses), an array of size NumFrames of the addresses to
 *           read/write for each frame.
 *       (aTypes), an array of size NumFrames of the data types corresponding
 *           to each address in aAddresses.
 *       (aNames), an array of size NumFrames of the names to read/write for
 *           each frame.
 *       aValues, an array of size NumFrames that represents the values to
 *           write from or read to.
 *       ErrorAddress, a pointer to an integer, which in the case of a relevant
 *           error, gets updated to contain the device-reported address that
 *           caused an error.
 * Note: Reads/writes are compressed into arrays for consecutive addresses that
 *       line up, based on type. See the LJM_ALLOWS_AUTO_CONDENSE_ADDRESSES
 *       configuration.
 * Note: These functions may take liberties in deciding what kind of Modbus
 *       operation to create. For more control of what kind of packets may be
 *       sent/received, please see the LJM_WriteLibraryConfigS function.
**/
LJM_ERROR_RETURN LJM_eReadAddresses(int Handle, int NumFrames,
	const int * aAddresses, const int * aTypes, double * aValues,
	int * ErrorAddress);
LJM_ERROR_RETURN LJM_eReadNames(int Handle, int NumFrames,
	const char ** aNames, double * aValues, int * ErrorAddress);

LJM_ERROR_RETURN LJM_eWriteAddresses(int Handle, int NumFrames,
	const int * aAddresses, const int * aTypes, const double * aValues,
	int * ErrorAddress);
LJM_ERROR_RETURN LJM_eWriteNames(int Handle, int NumFrames,
	const char ** aNames, const double * aValues, int * ErrorAddress);

// Easy Functions: All type, reading and writing, multiple values to one address
/**
 * Name: LJM_eReadAddressArray, LJM_eReadNameArray
 *       LJM_eWriteAddressArray, LJM_eWriteNameArray
 * Desc: Performs a Modbus operation to either read or write an array.
 * Para: Handle, a valid handle to an open device.
 *       (Address), the address to read an array from or write an array to.
 *       (Type), the data type of Address.
 *       (Name), the register name to read an array from or write an array to.
 *       NumValues, the size of the array to read or write.
 *       aValues, an array of size NumValues that represents the values to
 *           write from or read to.
 *       ErrorAddress, a pointer to an integer, which in the case of a relevant
 *           error, gets updated to contain the device-reported address that
 *           caused an error.
 * Note: If NumValues is large enough, these functions will automatically split
 *       writes and reads into multiple packets based on the current device's
 *       effective data packet size. Using both non-buffer and buffer registers
 *       in one function call is not supported.
**/
LJM_ERROR_RETURN LJM_eReadAddressArray(int Handle, int Address, int Type,
	int NumValues, double * aValues, int * ErrorAddress);
LJM_ERROR_RETURN LJM_eReadNameArray(int Handle, const char * Name,
	int NumValues, double * aValues, int * ErrorAddress);

LJM_ERROR_RETURN LJM_eWriteAddressArray(int Handle, int Address, int Type,
	int NumValues, const double * aValues, int * ErrorAddress);
LJM_ERROR_RETURN LJM_eWriteNameArray(int Handle, const char * Name,
	int NumValues, const double * aValues, int * ErrorAddress);

// Easy Functions: Reading and writing using bytes
/**
 * Name: LJM_eReadAddressByteArray, LJM_eReadNameByteArray
 *       LJM_eWriteAddressByteArray, LJM_eWriteNameByteArray
 * Desc: Performs a Modbus operation to either read or write a byte array.
 * Para: Handle, a valid handle to an open device.
 *       (Address), the address to read an array from or write a byte array to.
 *       (Name), the register name to read an array from or write a byte array
 *           to.
 *       NumBytes, the size of the byte array to read or write.
 *       aBytes, a byte array of size NumBytes that represents the values to
 *           write from or read to.
 *       ErrorAddress, a pointer to an integer, which in the case of a relevant
 *           error, gets updated to contain the device-reported address that
 *           caused an error.
 * Note: These functions will append a 0x00 byte to aBytes for odd-numbered
 *       NumBytes.
 * Note: If NumBytes is large enough, these functions will automatically split
 *       writes and reads into multiple packets based on the current device's
 *       effective data packet size. Using both non-buffer and buffer registers
 *       in one function call is not supported.
**/
LJM_ERROR_RETURN LJM_eReadAddressByteArray(int Handle, int Address,
	int NumBytes, char * aBytes, int * ErrorAddress);
LJM_ERROR_RETURN LJM_eReadNameByteArray(int Handle, const char * Name,
	int NumBytes, char * aBytes, int * ErrorAddress);

LJM_ERROR_RETURN LJM_eWriteAddressByteArray(int Handle, int Address,
	int NumBytes, const char * aBytes, int * ErrorAddress);
LJM_ERROR_RETURN LJM_eWriteNameByteArray(int Handle, const char * Name,
	int NumBytes, const char * aBytes, int * ErrorAddress);

// Easy Functions: All type, reading and writing, multiple addresses with
//     multiple values for each
/**
 * Name: LJM_eAddresses, LJM_eNames
 * Desc: Performs Modbus operations that write/read data.
 * Para: Handle, a valid handle to an open device.
 *       NumFrames, the total number of reads/writes frames to perform.
 *       (aAddresses), an array of size NumFrames of the addresses to
 *           read/write for each frame.
 *       (aTypes), an array of size NumFrames of the data types corresponding
 *           to each address in aAddresses.
 *       (aNames), an array of size NumFrames of the names to read/write for
 *           each frame.
 *       aWrites, an array of size NumFrames of the direction/access type
 *           (LJM_READ or LJM_WRITE) for each frame.
 *       aNumValues, an array of size NumFrames giving the number of values to
 *           read/write for each frame.
 *       aValues, an array that represents the values to write or read. The
 *           size of this array must be the sum of aNumValues.
 *       ErrorAddress, a pointer to an integer, which in the case of a relevant
 *           error, gets updated to contain the device-reported address that
 *           caused an error.
 * Note: Reads/writes are compressed into arrays for consecutive addresses that
 *       line up, based on type. See the LJM_ALLOWS_AUTO_CONDENSE_ADDRESSES
 *       configuration.
 * Note: These functions may take liberties in deciding what kind of Modbus
 *       operation to create. For more control of what kind of packets may be
 *       sent/received, please see the LJM_WriteLibraryConfigS function.
**/
LJM_ERROR_RETURN LJM_eAddresses(int Handle, int NumFrames,
	const int * aAddresses, const int * aTypes, const int * aWrites,
	const int * aNumValues, double * aValues, int * ErrorAddress);
LJM_ERROR_RETURN LJM_eNames(int Handle, int NumFrames, const char ** aNames,
	const int * aWrites, const int * aNumValues, double * aValues,
	int * ErrorAddress);

/**
 * Name: LJM_eReadNameString, LJM_eReadAddressString
 * Desc: Reads a 50-byte string from a device.
 * Para: Handle, a valid handle to an open device.
 *       (Name), the name of the registers to read, which must be of type
 *           LJM_STRING.
 *       (Address), the address of the registers to read an LJM_STRING from.
 *       String, A string that is updated to contain the result of the read.
 *           Must be allocated to size LJM_STRING_ALLOCATION_SIZE or
 *           greater prior to calling this function.
 * Note: This is a convenience function for LJM_eNames/LJM_eAddressess.
 * Note: LJM_eReadNameString checks to make sure that Name is in the constants
 *       file and describes registers that have a data type of LJM_STRING,
 *       but LJM_eReadAddressString does not perform any data type checking.
**/
LJM_ERROR_RETURN LJM_eReadNameString(int Handle, const char * Name,
	char * String);
LJM_ERROR_RETURN LJM_eReadAddressString(int Handle, int Address,
	char * String);

/**
 * Name: LJM_eWriteNameString, LJM_eWriteAddressString
 * Desc: Writes a 50-byte string to a device.
 * Para: Handle, a valid handle to an open device.
 *       (Name), the name of the registers to write to, which must be of type
 *           LJM_STRING
 *       (Address), the address of the registers to write an LJM_STRING to
 *       String, The string to write. Must null-terminate at length
 *           LJM_STRING_ALLOCATION_SIZE or less.
 * Note: This is a convenience function for LJM_eNames/LJM_eAddressess
 * Note: LJM_eWriteNameString checks to make sure that Name is in the constants
 *       file and describes registers that have a data type of LJM_STRING,
 *       but LJM_eWriteAddressString does not perform any data type checking.
**/
LJM_ERROR_RETURN LJM_eWriteNameString(int Handle, const char * Name,
	const char * String);
LJM_ERROR_RETURN LJM_eWriteAddressString(int Handle, int Address,
	const char * String);


/*********************
 *  Stream Functions *
 *********************/

/**
 * Name: LJM_eStreamStart
 * Desc: Initializes a stream object and begins streaming. This includes
 *       creating a buffer in LJM that collects data from the device.
 * Para: Handle, a valid handle to an open device.
 *       ScansPerRead, Number of scans returned by each call to the
 *           LJM_eStreamRead function. This is not tied to the maximum packet
 *           size for the device.
 *       NumAddresses, The size of aScanList. The number of addresses to scan.
 *       aScanList, Array of Modbus addresses to collect samples from, per scan.
 *       ScanRate, input/output pointer. Sets the desired number of scans per
 *           second. Upon successful return of this function, gets updated to
 *           the actual scan rate that the device will scan at.
 * Note: Address configuration such as range, resolution, and differential
 *       voltages are handled by writing to the device.
 * Note: Check your device's documentation for which addresses are valid for
 *       aScanList.
**/
LJM_ERROR_RETURN LJM_eStreamStart(int Handle, int ScansPerRead,
	int NumAddresses, const int * aScanList, double * ScanRate);

/**
 * Name: LJM_eStreamRead
 * Desc: Returns data from an initialized and running LJM stream buffer. Waits
 *       for data to become available, if necessary.
 * Para: Handle, a valid handle to an open device.
 *       aData, Output data array. Returns all addresses interleaved. Must be
 *           large enough to hold (ScansPerRead * NumAddresses) values.
 *           ScansPerRead and NumAddresses are set when stream is set up with
 *           LJM_eStreamStart. The data returned is removed from the LJM stream
 *           buffer.
 *       DeviceScanBacklog, The number of scans left in the device buffer, as
 *           measured from when data was last collected from the device. This
 *           should usually be near zero and not growing for healthy streams.
 *       LJMScanBacklog, The number of scans left in the LJM buffer, as
 *           measured from after the data returned from this function is
 *           removed from the LJM buffer. This should usually be near zero and
 *           not growing for healthy streams.
 * Note: Returns LJME_NO_SCANS_RETURNED if LJM_STREAM_SCANS_RETURN is
 *       LJM_STREAM_SCANS_RETURN_ALL_OR_NONE.
**/
LJM_ERROR_RETURN LJM_eStreamRead(int Handle, double * aData,
	int * DeviceScanBacklog, int * LJMScanBacklog);

/**
 * Name: LJM_SetStreamCallback
 * Desc: Sets a callback that is called by LJM when the stream has collected
 *       ScansPerRead scans or if an error has occurred.
 * Para: Handle, a valid handle to an open device.
 *       Callback, the callback function for LJM's stream thread to call
 *           when stream data is ready, which should call LJM_eStreamRead to
 *           acquire data.
 *       Arg, the user-defined argument that is passed to Callback when it is
 *           invoked.
 * Note: LJM_SetStreamCallback should be called after LJM_eStreamStart.
 * Note: To disable the previous callback for stream reading, pass 0 or NULL as
 *       Callback.
 * Note: LJM_SetStreamCallback may not be called from within a
 *       LJM_StreamReadCallback.
**/
typedef void (*LJM_StreamReadCallback)(void *);
LJM_ERROR_RETURN LJM_SetStreamCallback(int Handle,
	LJM_StreamReadCallback Callback, void * Arg);

/**
 * Name: LJM_eStreamStop
 * Desc: Stops LJM from streaming any more data from the device, while leaving
 *       any collected data in the LJM buffer to be read. Stops the device from
 *       streaming.
 * Para: Handle, a valid handle to an open device.
**/
LJM_ERROR_RETURN LJM_eStreamStop(int Handle);

/**
 * Name: LJM_StreamBurst
 * Desc: Initializes a stream burst and collects data. This function combines
 *       LJM_eStreamStart, LJM_eStreamRead, and LJM_eStreamStop, as well as some
 *       other device initialization.
 * Para: Handle, a valid handle to an open device.
 *       NumAddresses, The size of aScanList. The number of addresses to scan.
 *       aScanList, an array of Modbus addresses to collect samples from, per
 *           scan.
 *       ScanRate, input/output pointer. Sets the desired number of scans per
 *           second. Upon successful return of this function, gets updated to
 *           the actual scan rate that the device scanned at.
 *       NumScans, the number of scans to collect. This is how many burst scans
 *           are collected and may not be zero.
 *       aData, the output data array. Returns all addresses interleaved. Must
 *           be large enough to hold (NumScans * NumAddresses) values.
 * Note: Address configuration such as range, resolution, and differential
 *       voltages are handled by writing to the device.
 * Note: Check your device's documentation for which addresses are valid for
 *       aScanList and how many burst scans may be collected.
 * Note: This function will block for (NumScans / ScanRate) seconds or longer.
**/
LJM_ERROR_RETURN LJM_StreamBurst(int Handle, int NumAddresses,
	const int * aScanList, double * ScanRate, unsigned int NumScans,
	double * aData);

/**
 * Name: LJM_GetStreamTCPReceiveBufferStatus
 * Desc: Gets the backlog status of the TCP receive buffer.
 * Para: Handle, a valid handle to an open device which is running a
 *           TCP-based stream.
 *       ReceiveBufferBytesSize, the current maximum number of bytes that can be
 *           stored in the receive buffer before it is full.
 *       ReceiveBufferBytesBacklog, the current number of bytes stored in the
 *           receive buffer.
**/
LJM_ERROR_RETURN LJM_GetStreamTCPReceiveBufferStatus(int Handle,
    unsigned int * ReceiveBufferBytesSize,
    unsigned int * ReceiveBufferBytesBacklog);

/**
 * Name: LJM_InitializeAperiodicStreamOut
 * Desc: Initializes all device registers necessary to start a aperiodic
 *       stream-out.
 * Para: Handle, a valid handle to an open device.
 *       StreamOutIndex, the number of this stream-out.
 *           Note: T-series devices support a maximum of 4 stream-outs.
 *       TargetAddr, the register to update during stream-out
 *           stored in the receive buffer before it is full.
 *       ScanRate, the scan rate that the stream is initialized to.
**/
LJM_ERROR_RETURN LJM_InitializeAperiodicStreamOut(int Handle,
     int StreamOutIndex,
     int TargetAddr,
     double ScanRate);

/**
 * Name: LJM_WriteAperiodicStreamOut
 * Desc: Writes data to the buffer of the specified aperiodic stream-out
 * Para: Handle, a valid handle to an open device.
 *       StreamOutIndex, the number of this stream-out.
 *       NumValues, the number of values to write to the stream-out buffer
 *       aWriteData, the data array to be written to the stream-out buffer
 *           Note: the size of the array should be equal to the buffer size
 *           in bytes divided by 4 (BufferNumBytes / 4).
 *       LJMBufferStatus, the number of samples that can be written to the
 *           stream-out queue.
 * Note: The aperiodic stream-out should be initialized with
 *       LJM_InitializeAperiodicStreamOut prior to running this function.
**/
LJM_ERROR_RETURN LJM_WriteAperiodicStreamOut(int Handle,
     int StreamOutIndex,
     int NumValues,
     const double * aWriteData,
     int * LJMBufferStatus);

/**
 * Name: LJM_PeriodicStreamOut
 * Desc: Initializes all registers necessary to start streaming out a periodic
 *       waveform (looping over the values written to the function)
 * Para: Handle, a valid handle to an open device.
 *       StreamOutIndex, the number of this stream-out.
 *       TargetAddr, the register to update during stream-out
 *           stored in the receive buffer before it is full.
 *       ScanRate, the scan rate that the stream is initialized to.
 *       NumValues, the number of values to loop over
 *       aWriteData, the data array to be written (this should be one period
 *       of the waveform)
**/
LJM_ERROR_RETURN LJM_PeriodicStreamOut(int Handle,
     int StreamOutIndex,
     int TargetAddr,
     double ScanRate,
     int NumValues,
     const double * aWriteData);


/***************************************
 *  Byte-oriented Read/Write Functions *
 ***************************************/

/**
 * Name: LJM_WriteRaw
 * Desc: Sends an unaltered data packet to a device.
 * Para: Handle, a valid handle to an open device.
 *       Data, the byte array packet to send.
 *       NumBytes, the size of Data.
**/
LJM_ERROR_RETURN LJM_WriteRaw(int Handle, const unsigned char * Data,
	int NumBytes);

/**
 * Name: LJM_ReadRaw
 * Desc: Reads an unaltered data packet from a device.
 * Para: Handle, a valid handle to an open device.
 *       Data, the allocated byte array to receive the data packet.
 *       NumBytes, the number of bytes to receive.
**/
LJM_ERROR_RETURN LJM_ReadRaw(int Handle, unsigned char * Data, int NumBytes);

/**
 * Name: LJM_AddressesToMBFB
 * Desc: Takes in arrays that together represent operations to be performed on a
 *       device and outputs a byte array representing a valid Feedback command,
 *       which can be used as input to the LJM_MBFBComm function.
 * Para: MaxBytesPerMBFB, the maximum number of bytes that the Feedback command
 *           is allowed to consist of. It is highly recommended to pass the size
 *           of the aMBFBCommand buffer as MaxBytesPerMBFB to prevent buffer
 *           overflow. See LJM_DEFAULT_FEEDBACK_ALLOCATION_SIZE.
 *       aAddresses, an array of size NumFrames representing the register
 *           addresses to read from or write to for each frame.
 *       aTypes, an array of size NumFrames representing the data types to read
 *           or write. See the Data Type constants in this header file.
 *       aWrites, an array of size NumFrames of the direction/access type
 *           (LJM_READ or LJM_WRITE) for each frame.
 *       aNumValues, an array of size NumFrames giving the number of values to
 *           read/write for each frame.
 *       aValues, for every entry in aWrites[i] that is LJM_WRITE, this contains
 *           aNumValues[i] values to write and for every entry in aWrites that
 *           is LJM_READ, this contains aNumValues[i] values that can later be
 *           updated in the LJM_UpdateValues function. aValues values must be
 *           in the same order as the rest of the arrays. For example, if aWrite is
 *               {LJM_WRITE, LJM_READ, LJM_WRITE},
 *           and aNumValues is
 *               {1, 4, 2}
 *           aValues would have one value to be written, then 4 blank/garbage
 *           values to later be updated, and then 2 values to be written.
 *       NumFrames, A pointer to the number of frames being created, which is
 *           also the size of aAddresses, aTypes, aWrites, and aNumValues. Once
 *           this function returns, NumFrames will be updated to the number of
 *           frames aMBFBCommand contains.
 *       aMBFBCommand, the output parameter that contains the valid Feedback
 *           command. Transaction ID and Unit ID will be blanks that
 *           LJM_MBFBComm will fill in.
 * Warn: aMBFBCommand must be an allocated array of size large enough to hold
 *       the Feedback command (including its frames). You can use
 *       MaxBytesPerMBFB to limit the size of the Feedback command.
 * Note: If this function returns LJME_FRAMES_OMITTED_DUE_TO_PACKET_SIZE,
 *       aMBFBCommand is still valid, but does not contain all of the frames
 *       that were intended and NumFrames is updated to contain the number of
 *       frames that were included.
**/
LJM_ERROR_RETURN LJM_AddressesToMBFB(int MaxBytesPerMBFB, const int * aAddresses,
	const int * aTypes, const int * aWrites, const int * aNumValues,
	const double * aValues, int * NumFrames, unsigned char * aMBFBCommand);

/**
 * Name: LJM_MBFBComm
 * Desc: Sends a Feedback command and receives a Feedback response, parsing the
 *       response for obvious errors. This function adds its own Transaction ID
 *       to the command. The Feedback response may be parsed with the
 *       LJM_UpdateValues function.
 * Para: Handle, a valid handle to an open device.
 *       UnitID, the ID of the specific unit that the Feedback command should
 *           be sent to. Use LJM_DEFAULT_UNIT_ID unless the device
 *           documentation instructs otherwise.
 *       aMBFB, both an input parameter and an output parameter. As an input
 *           parameter, it is a valid Feedback command. As an output parameter,
 *           it is a Feedback response, which may be an error response.
 *       ErrorAddress, a pointer to an integer, which in the case of a relevant
 *                     error, gets updated to contain the device-reported
 *                     address that caused an error.
 * Note: aMBFB must be an allocated array of size large enough to hold the
 *       Feedback response.
**/
LJM_ERROR_RETURN LJM_MBFBComm(int Handle, unsigned char UnitID,
	unsigned char * aMBFB, int * ErrorAddress);

/**
 * Name: LJM_UpdateValues
 * Desc: Takes a Feedback response named aMBFBResponse from a device and the
 *       arrays corresponding to the Feedback command for which aMBFBResponse
 *       is a response and updates the aValues array based on the read data in
 *       aMBFBResponse.
 * Para: aMBFBResponse, a valid Feedback response.
 *       aTypes, an array of size NumFrames representing the data types to read
 *           or write. See the Data Type constants in this header file.
 *       aWrites, the array of constants from LabJackM.h - either LJM_WRITE or
 *           LJM_READ.
 *       aNumValues, the array representing how many values were read or written.
 *       NumFrames, the number of frames in aTypes, aWrites, and aNumValues.
 *       aValues, for every entry in aWrites[i] that is LJM_WRITE, this contains
 *           aNumValues[i] values that were written and for every entry in
 *           aWrites that is LJM_READ, this contains aNumValues[i] values
 *           that will now be updated. aValues values must be in the same
 *           order as the rest of the arrays. For example, if aWrite is
 *               {LJM_WRITE, LJM_READ, LJM_WRITE},
 *           and aNumValues is
 *               {1, 4, 2}
 *           LJM_UpdateValues would skip one value, then update 4 values with
 *           the data in aMBFBResponse, then do nothing with the last 2 values.
**/
LJM_ERROR_RETURN LJM_UpdateValues(unsigned char * aMBFBResponse,
	const int * aTypes, const int * aWrites, const int * aNumValues,
	int NumFrames, double * aValues);


/****************************
 * Constants File Functions *
 ****************************/

/**
 * Name: LJM_NamesToAddresses
 * Desc: Takes a list of Modbus register names as input and produces two lists
 *       as output - the corresponding Modbus addresses and types. These two
 *       lists can serve as input to functions that have Address/aAddresses and
 *       Type/aTypes as input parameters.
 * Para: NumFrames, the number of names in aNames and the allocated size of
 *           aAddresses and aTypes.
 *       aNames, an array of null-terminated C-string register identifiers.
 *           These register identifiers can be register names or register
 *           alternate names.
 *       aAddresses, output parameter containing the addresses described by
 *           aNames in the same order, must be allocated to the size NumFrames
 *           before calling LJM_NamesToAddresses.
 *       aTypes, output parameter containing the types described by aNames in
 *           the same order, must be allocated to the size NumFrames before
 *           calling LJM_NamesToAddresses.
 * Note: For each register identifier in aNames that is invalid, the
 *       corresponding aAddresses value will be set to LJM_INVALID_NAME_ADDRESS.
**/
LJM_ERROR_RETURN LJM_NamesToAddresses(int NumFrames, const char ** aNames,
	int * aAddresses, int * aTypes);

/**
 * Name: LJM_NameToAddress
 * Desc: Takes a Modbus register name as input and produces the corresponding
 *       Modbus address and type. These two values can serve as input to
 *       functions that have Address/aAddresses and Type/aTypes as input
 *       parameters.
 * Para: Name, a null-terminated C-string register identifier. These register
 *           identifiers can be register names or register alternate names.
 *       Address, output parameter containing the address described by Name.
 *       Type, output parameter containing the type described by Names.
 * Note: If Name is not a valid register identifier, Address will be set to
 *       LJM_INVALID_NAME_ADDRESS.
**/
LJM_ERROR_RETURN LJM_NameToAddress(const char * Name, int * Address, int * Type);

/**
 * Name: LJM_AddressesToTypes
 * Desc: Retrieves multiple data types for given Modbus register addresses.
 * Para: NumAddresses, the number of addresses to convert to data types.
 *       aAddresses, an array of size NumAddresses of Modbus register addresses.
 *       aType, a pre-allocated array of size NumAddresses which gets updated
 *           to the data type for each corresponding entry in Addresses.
 * Note: For each aAddresses[i] that is not found, the corresponding entry
 *       aTypes[i] will be set to LJM_INVALID_NAME_ADDRESS and this function
 *       will return LJME_INVALID_ADDRESS.
**/
LJM_ERROR_RETURN LJM_AddressesToTypes(int NumAddresses, int * aAddresses,
	int * aTypes);

/**
 * Name: LJM_AddressToType
 * Desc: Retrieves the data type for a given Modbus register address.
 * Para: Address, the Modbus register address to look up.
 *       Type, an integer pointer which gets updated to the data type of
 *           Address.
**/
LJM_ERROR_RETURN LJM_AddressToType(int Address, int * Type);

/**
 * Name: LJM_LookupConstantValue
 * Desc: Takes a register name or other scope and a constant name, and returns
 *       the constant value.
 * Para: Scope, the register name or other scope to search within. Must be of
 *           size LJM_MAX_NAME_SIZE or less.
 *       ConstantName, the name of the constant to search for. Must be of size
 *           LJM_MAX_NAME_SIZE or less.
 *       ConstantValue, the returned value of ConstantName within the scope
 *           of Scope, if found.
**/
LJM_ERROR_RETURN LJM_LookupConstantValue(const char * Scope,
	const char * ConstantName, double * ConstantValue);

/**
 * Name: LJM_LookupConstantName
 * Desc: Takes a register name or other scope and a value, and returns the
 *       name of that value, if found within the given scope.
 * Para: Scope, the register name or other scope to search within. Must be of
 *           size LJM_MAX_NAME_SIZE or less.
 *       ConstantValue, the constant value to search for.
 *       ConstantName, a pointer to a char array allocated to size
 *           LJM_MAX_NAME_SIZE, used to return the null-terminated constant
 *           name.
**/
LJM_ERROR_RETURN LJM_LookupConstantName(const char * Scope,
	double ConstantValue, char * ConstantName);

/**
 * Name: LJM_ErrorToString
 * Desc: Gets the name of an error code.
 * Para: ErrorCode, the error code to look up.
 *       ErrorString, a pointer to a char array allocated to size
 *           LJM_MAX_NAME_SIZE, used to return the null-terminated error name.
 * Note: If the constants file that has been loaded does not contain
 *       ErrorCode, this returns a null-terminated message saying so.
 *       If the constants file could not be opened, this returns a
 *       null-terminated string saying so and where that constants file was
 *       expected to be.
**/
LJM_VOID_RETURN LJM_ErrorToString(int ErrorCode, char * ErrorString);

/**
 * Name: LJM_LoadConstants
 * Desc: Manually loads or reloads the constants files associated with
 *       the LJM_ErrorToString and LJM_NamesToAddresses functions.
 * Note: This step is handled automatically. This function does not need to be
 *       called before either LJM_ErrorToString or LJM_NamesToAddresses.
**/
LJM_VOID_RETURN LJM_LoadConstants(void);

/**
 * Name: LJM_LoadConstantsFromFile
 * Desc: Alias for executing:
 *       LJM_WriteLibraryConfigStringS(LJM_CONSTANTS_FILE, FileName)
 * Para: FileName, the absolute or relative file path string to pass to
 *            LJM_WriteLibraryConfigStringS as the String parameter. Must
 *            null-terminate.
**/
LJM_ERROR_RETURN LJM_LoadConstantsFromFile(const char * FileName);

/**
 * Name: LJM_LoadConstantsFromString
 * Desc: Parses JsonString as the constants file and loads it.
 * Para: JsonString, A JSON string containing a "registers" array and/or an "errors"
 *           array.
 * Note: If the JSON string does not contain a "registers" array, the Modbus-related
 *       constants are not affected. Similarly, if the JSON string does not contain
 *       an "errors" array, the errorcode-related constants are not affected.
**/
LJM_ERROR_RETURN LJM_LoadConstantsFromString(const char * JsonString);


/******************************
 *  Type Conversion Functions *
 ******************************/

// Thermocouple conversion

// Thermocouple Type constants
static const long LJM_ttB = 6001;
static const long LJM_ttE = 6002;
static const long LJM_ttJ = 6003;
static const long LJM_ttK = 6004;
static const long LJM_ttN = 6005;
static const long LJM_ttR = 6006;
static const long LJM_ttS = 6007;
static const long LJM_ttT = 6008;
static const long LJM_ttC = 6009;

/**
 * Desc: Converts thermocouple voltage to temperature.
 * Para: TCType, the thermocouple type. See "Thermocouple Type constants".
 *       TCVolts, the voltage reported by the thermocouple.
 *       CJTempK, the cold junction temperature in degrees Kelvin.
 *       pTCTempK, outputs the calculated temperature in degrees Kelvin.
 * Note: B-type measurements below ~373 degrees Kelvin or ~0.04 millivolts (at a
 *       cold junction junction temperature of 273.15 degrees Kelvin) may be
 *       inaccurate.
**/
LJM_ERROR_RETURN LJM_TCVoltsToTemp(int TCType, double TCVolts, double CJTempK,
	double * pTCTempK);


/**
 * Name: LJM_TYPE#BitsToByteArray
 * Desc: Converts big-endian Modbus data from C-types array to a byte array.
 * Para: aTYPE#Bits (such as aFLOAT32), the array of values to be converted.
 *       RegisterOffset, the register offset to put the converted values in aBytes.
 *           The actual offset depends on how many bits the type is.
 *       NumTYPE#Bits (such as NumFLOAT32), the number of values to convert.
 *       aBytes, the converted values in byte form.
 * Note: On little-endian platforms, automatic endian conversions will be
 *       performed.
**/

/**
 * Name: LJM_ByteArrayToTYPE#Bits
 * Desc: Converts big-endian Modbus data from byte array to C-type array.
 * Para: aBytes, the bytes to be converted.
 *       RegisterOffset, the register offset to get the values from in aBytes.
 *           The actual offset depends on how many bits the type is.
 *       NumTYPE#Bits (such as NumFLOAT32), the number of values to convert.
 *       aTYPE#Bits (such as aFLOAT32), the converted values in C-type form.
 * Note: On little-endian platforms, automatic endian conversions will be
 *       performed.
**/

// Single precision float, 32 bits
//   (the C type "float")
LJM_VOID_RETURN LJM_FLOAT32ToByteArray(const float * aFLOAT32, int RegisterOffset, int NumFLOAT32, unsigned char * aBytes);
LJM_VOID_RETURN LJM_ByteArrayToFLOAT32(const unsigned char * aBytes, int RegisterOffset, int NumFLOAT32, float * aFLOAT32);

// Unsigned 16 bit integer
//   (the C type "unsigned short" or similar)
LJM_VOID_RETURN LJM_UINT16ToByteArray(const unsigned short * aUINT16, int RegisterOffset, int NumUINT16, unsigned char * aBytes);
LJM_VOID_RETURN LJM_ByteArrayToUINT16(const unsigned char * aBytes, int RegisterOffset, int NumUINT16, unsigned short * aUINT16);

// Unsigned 32 bit integer
//   (the C type "unsigned int" or similar)
LJM_VOID_RETURN LJM_UINT32ToByteArray(const unsigned int * aUINT32, int RegisterOffset, int NumUINT32, unsigned char * aBytes);
LJM_VOID_RETURN LJM_ByteArrayToUINT32(const unsigned char * aBytes, int RegisterOffset, int NumUINT32, unsigned int * aUINT32);

// Signed 32 bit integer
//   (the C type "int" or similar)
LJM_VOID_RETURN LJM_INT32ToByteArray(const int * aINT32, int RegisterOffset, int NumINT32, unsigned char * aBytes);
LJM_VOID_RETURN LJM_ByteArrayToINT32(const unsigned char * aBytes, int RegisterOffset, int NumINT32, int * aINT32);

/**
 * Name: LJM_NumberToIP
 * Desc: Takes an integer representing an IPv4 address and outputs the corresponding
 *       decimal-dot IPv4 address as a null-terminated string.
 * Para: Number, the numerical representation of an IP address to be converted to
 *           a string representation.
 *       IPv4String, a char array that must be allocated to size LJM_IPv4_STRING_SIZE
 *           which will be set to contain the null-terminated string representation
 *           of the IP address after the completion of this function. Unused
 *           bytes will be set to NULL.
 * Retr: LJME_NOERROR for no detected errors,
 *       LJME_NULL_POINTER if IPv4String is NULL.
**/
LJM_ERROR_RETURN LJM_NumberToIP(unsigned int Number, char * IPv4String);

/**
 * Name: LJM_IPToNumber
 * Desc: Takes a decimal-dot IPv4 string representing an IPv4 address and outputs the
 *       corresponding integer version of the address.
 * Para: IPv4String, the string representation of the IP address to be converted to
 *           a numerical representation.
 *       Number, the output parameter that will be updated to contain the numerical
 *           representation of IPv4String after the completion of this function.
 * Retr: LJME_NOERROR for no detected errors,
 *       LJME_NULL_POINTER if IPv4String or Number is NULL,
 *       LJME_INVALID_PARAMETER if IPv4String could not be parsed as a IPv4 address.
**/
LJM_ERROR_RETURN LJM_IPToNumber(const char * IPv4String, unsigned int * Number);

/**
 * Name: LJM_NumberToMAC
 * Desc: Takes an integer representing a MAC address and outputs the corresponding
 *       hex-colon MAC address as a string.
 * Para: Number, the numerical representation of a MAC address to be converted to
 *           a string representation.
 *       MACString, a char array that must be allocated to size LJM_MAC_STRING_SIZE
 *           which will be set to contain the string representation of the MAC
 *           address after the completion of this function.
 * Retr: LJME_NOERROR for no detected errors,
 *       LJME_NULL_POINTER if MACString is NULL.
**/
LJM_ERROR_RETURN LJM_NumberToMAC(unsigned long long Number, char * MACString);

/**
 * Name: LJM_MACToNumber
 * Desc: Takes a hex-colon string representing a MAC address and outputs the
 *       corresponding integer version of the address.
 * Para: MACString, the string representation of the MAC address to be converted to
 *           a numerical representation.
 *       Number, the output parameter that will be updated to contain the numerical
 *           representation of MACString after the completion of this function.
 * Retr: LJME_NOERROR for no detected errors,
 *       LJME_NULL_POINTER if MACString or Number is NULL,
 *       LJME_INVALID_PARAMETER if MACString could not be parsed as a MAC address.
**/
LJM_ERROR_RETURN LJM_MACToNumber(const char * MACString, unsigned long long * Number);


/*****************************
 *  Timing Utility Functions *
 *****************************/

/**
 * Name: LJM_GetHostTick
 * Desc: Queries the host system's steady (monotonic) clock, preferentially with
 *       high precision.
 * Retr: The current clock tick in microseconds.
**/
LJM_LONG_LONG_RETURN LJM_GetHostTick(void);

/**
 * Name: LJM_GetHostTick32Bit
 * Desc: The same as LJM_GetHostTick, but with two 32-bit integers as parameters
 * Para: TickUpper, the upper (most significant) 32 bits of the clock tick
 *       TickLower, the lower (least significant) 32 bits of the clock tick
**/
LJM_VOID_RETURN LJM_GetHostTick32Bit(unsigned int * TickUpper, unsigned int * TickLower);

/**
 * Name: LJM_StartInterval
 * Desc: Allocates memory for the given IntervalHandle and begins a reoccurring
 *       interval timer. This function does not wait.
 * Para: IntervalHandle, the user-generated interval identifier.
 *       Microseconds, the number of microseconds in the interval.
**/
LJM_ERROR_RETURN LJM_StartInterval(int IntervalHandle, int Microseconds);

/**
 * Name: LJM_WaitForNextInterval
 * Desc: Waits (blocks/sleeps) until the next interval occurs. If intervals are
 *       skipped, this function still waits until the next complete interval.
 * Para: IntervalHandle, the user-generated interval identifier.
 *       SkippedIntervals, the number of skipped intervals that occurred since
 *           the last time this function was called.
 * Retr: LJME_INVALID_INTERVAL_HANDLE, if IntervalHandle was not set up using
 *           LJM_StartInterval
**/
LJM_ERROR_RETURN LJM_WaitForNextInterval(int IntervalHandle, int * SkippedIntervals);

/**
 * Name: LJM_CleanInterval
 * Desc: Cleans/deallocates memory for the given IntervalHandle.
 * Para: IntervalHandle, the user-generated interval identifier.
 * Retr: LJME_INVALID_INTERVAL_HANDLE, if IntervalHandle was not set up using
 *           LJM_StartInterval
**/
LJM_ERROR_RETURN LJM_CleanInterval(int IntervalHandle);


/**********************
 *  LJM Configuration *
 **********************/

// Config Parameters

/**
 * Desc: The maximum number of milliseconds that LJM will wait for a packet to
 *       be sent and also for a packet to be received before timing out. In
 *       other words, LJM can wait this long for a command to be sent, then
 *       wait this long again for the response to be received.
**/
static const char * const LJM_USB_SEND_RECEIVE_TIMEOUT_MS = "LJM_USB_SEND_RECEIVE_TIMEOUT_MS";
static const char * const LJM_ETHERNET_SEND_RECEIVE_TIMEOUT_MS = "LJM_ETHERNET_SEND_RECEIVE_TIMEOUT_MS";
static const char * const LJM_WIFI_SEND_RECEIVE_TIMEOUT_MS = "LJM_WIFI_SEND_RECEIVE_TIMEOUT_MS";

/**
 * Desc: Sets LJM_USB_SEND_RECEIVE_TIMEOUT_MS, LJM_ETHERNET_SEND_RECEIVE_TIMEOUT_MS,
 *       and LJM_WIFI_SEND_RECEIVE_TIMEOUT_MS.
 * Note: Write-only. May not be read.
**/
static const char * const LJM_SEND_RECEIVE_TIMEOUT_MS = "LJM_SEND_RECEIVE_TIMEOUT_MS";

/**
 * Name: LJM_ETHERNET_OPEN_TIMEOUT_MS
 * Desc: The maximum number of milliseconds that LJM will wait for a device
 *       being opened via TCP to respond before timing out.
**/
static const char * const LJM_ETHERNET_OPEN_TIMEOUT_MS = "LJM_ETHERNET_OPEN_TIMEOUT_MS";

/**
 * Name: LJM_WIFI_OPEN_TIMEOUT_MS
 * Desc: The maximum number of milliseconds that LJM will wait for a device
 *       being opened via TCP to respond before timing out.
**/
static const char * const LJM_WIFI_OPEN_TIMEOUT_MS = "LJM_WIFI_OPEN_TIMEOUT_MS";

/**
 * Name: LJM_OPEN_TCP_DEVICE_TIMEOUT_MS
 * Desc: Sets both LJM_ETHERNET_OPEN_TIMEOUT_MS and LJM_WIFI_OPEN_TIMEOUT_MS.
 * Note: Write-only. May not be read.
**/
static const char * const LJM_OPEN_TCP_DEVICE_TIMEOUT_MS = "LJM_OPEN_TCP_DEVICE_TIMEOUT_MS";

/**
 * Name: LJM_DEBUG_LOG_MODE
 * Desc: Any of the following modes:
 * Vals: 1 (default) - Never logs anything, regardless of LJM_DEBUG_LOG_LEVEL.
 *       2 - Log continuously to the log file according to LJM_DEBUG_LOG_LEVEL (see LJM_DEBUG_LOG_FILE).
 *       3 - Continuously stores a finite number of log messages, writes them to file upon error.
**/
static const char * const LJM_DEBUG_LOG_MODE = "LJM_DEBUG_LOG_MODE";
enum {
	LJM_DEBUG_LOG_MODE_NEVER = 1,
	LJM_DEBUG_LOG_MODE_CONTINUOUS = 2,
	LJM_DEBUG_LOG_MODE_ON_ERROR = 3
};

/**
 * Name: LJM_DEBUG_LOG_LEVEL
 * Desc: The level of priority that LJM will log. Levels that are lower than
 *       the current LJM_DEBUG_LOG_LEVEL are not logged. For example, if log
 *       priority is set to LJM_WARNING, messages with priority level
 *       LJM_WARNING and greater are logged to the debug file.
 * Vals: See below.
 * Note: LJM_PACKET is the default value.
**/
static const char * const LJM_DEBUG_LOG_LEVEL = "LJM_DEBUG_LOG_LEVEL";
enum {
	LJM_STREAM_PACKET = 1,
	LJM_TRACE = 2,
	LJM_DEBUG = 4,
	LJM_INFO = 6,
	LJM_PACKET = 7,
	LJM_WARNING = 8,
	LJM_USER = 9,
	LJM_ERROR = 10,
	LJM_FATAL = 12
};

/**
 * Name: LJM_DEBUG_LOG_BUFFER_MAX_SIZE
 * Desc: The number of log messages LJM's logger buffer can hold.
**/
static const char * const LJM_DEBUG_LOG_BUFFER_MAX_SIZE = "LJM_DEBUG_LOG_BUFFER_MAX_SIZE";

/**
 * Name: LJM_DEBUG_LOG_SLEEP_TIME_MS
 * Desc: The number of milliseconds the logger thread will sleep for between
 *       flushing the messages in the logger buffer to the log file.
 * Note: See also LJM_DEBUG_LOG_BUFFER_MAX_SIZE
**/
static const char * const LJM_DEBUG_LOG_SLEEP_TIME_MS = "LJM_DEBUG_LOG_SLEEP_TIME_MS";

/**
 * Name: LJM_LIBRARY_VERSION
 * Desc: Returns the current version of LJM. This will match LJM_VERSION (at
 *       the top of this header file) if you are using the executable LJM that
 *       corresponds to this header file.
**/
static const char * const LJM_LIBRARY_VERSION = "LJM_LIBRARY_VERSION";

/**
 * Name: LJM_ALLOWS_AUTO_MULTIPLE_FEEDBACKS
 * Desc: A mode that sets whether or not LJM will automatically send/receive
 *       multiple Feedback commands when the desired operations would exceed
 *       the maximum packet length. This mode is relevant to Easy functions
 *       such as LJM_eReadNames.
 * Vals: 0             - Disable
 *       Anything else - Enable (default)
**/
static const char * const LJM_ALLOWS_AUTO_MULTIPLE_FEEDBACKS = "LJM_ALLOWS_AUTO_MULTIPLE_FEEDBACKS";

/**
 * Name: LJM_ALLOWS_AUTO_CONDENSE_ADDRESSES
 * Desc: A mode that sets whether or not LJM will automatically condense
 *       single address reads/writes into array reads/writes, which minimizes
 *       packet size. This mode is relevant to Easy functions such as
 *       LJM_eReadNames.
 * Vals: 0             - Disable
 *       Anything else - Enable (default)
**/
static const char * const LJM_ALLOWS_AUTO_CONDENSE_ADDRESSES = "LJM_ALLOWS_AUTO_CONDENSE_ADDRESSES";

/**
 * Name: LJM_AUTO_IPS_FILE
 * Desc: The file LJM uses for the auto IPs feature. See LJM_AUTO_IPS.
**/
static const char * const LJM_AUTO_IPS_FILE = "LJM_AUTO_IPS_FILE";

/**
 * Name: LJM_AUTO_IPS
 * Desc: Sets whether or not LJM attempts to use the auto IPs feature. The auto
 *       IP feature reads and writes to the LJM_AUTO_IPS_FILE to help open
 *       network connections to LabJack devices.
 * Vals: 0 - Disable
 *       1 - Enable (default)
**/
static const char * const LJM_AUTO_IPS = "LJM_AUTO_IPS";

/**
 * Name: LJM_AUTO_RECONNECT_STICKY_CONNECTION
 * Desc: Sets whether or not LJM attempts to reconnect disrupted / disconnected
 *       connections according to same connection type as the original handle.
 * Vals: 0 - Disable
 *       1 - Enable (default)
**/
static const char * const LJM_AUTO_RECONNECT_STICKY_CONNECTION = "LJM_AUTO_RECONNECT_STICKY_CONNECTION";

/**
 * Name: LJM_AUTO_RECONNECT_STICKY_SERIAL
 * Desc: Sets whether or not LJM attempts to reconnect disrupted / disconnected
 *       connections according to same serial number as the original handle.
 * Vals: 0 - Disable
 *       1 - Enable (default)
**/
static const char * const LJM_AUTO_RECONNECT_STICKY_SERIAL = "LJM_AUTO_RECONNECT_STICKY_SERIAL";

/**
 * Name: LJM_AUTO_RECONNECT_WAIT_MS
 * Desc: Determines how long in milliseconds LJM waits between attempts to
 *       reconnect when a device has been found to be disconnected.
 * Note: Default is 500
**/
static const char * const LJM_AUTO_RECONNECT_WAIT_MS = "LJM_AUTO_RECONNECT_WAIT_MS";

/**
 * Name: LJM_INTERVAL_CLOCK_TYPE
 * Desc: Sets which type of clock LJM_StartInterval initializes.
 *       LJM_WaitForNextInterval will then use that given clock type for the
 *       given IntervalHandle.
 * Note: Default is LJM_INTERVAL_CLOCK_TYPE_STEADY
**/
static const char * const LJM_INTERVAL_CLOCK_TYPE = "LJM_INTERVAL_CLOCK_TYPE";
enum {
    LJM_INTERVAL_CLOCK_TYPE_STEADY = 1,
    LJM_INTERVAL_CLOCK_TYPE_SYSTEM = 2
};

/**
 * Name: LJM_MODBUS_MAP_CONSTANTS_FILE
 * Desc: Specifies absolute or relative path of the constants file to use for
 *       functions that use the LJM Name functionality, such as
 *       LJM_NamesToAddresses and LJM_eReadName.
**/
static const char * const LJM_MODBUS_MAP_CONSTANTS_FILE = "LJM_MODBUS_MAP_CONSTANTS_FILE";

/**
 * Name: LJM_ERROR_CONSTANTS_FILE
 * Desc: Specifies absolute or relative path of the constants file to use for
 *       LJM_ErrorToString.
**/
static const char * const LJM_ERROR_CONSTANTS_FILE = "LJM_ERROR_CONSTANTS_FILE";

/**
 * Name: LJM_DEBUG_LOG_FILE
 * Desc: Describes the absolute or relative path of the file to output log
 *       messages to.
 * Note: See LJM_DEBUG_LOG_MODE and LJM_DEBUG_LOG_LEVEL.
**/
static const char * const LJM_DEBUG_LOG_FILE = "LJM_DEBUG_LOG_FILE";

/**
 * Name: LJM_CONSTANTS_FILE
 * Desc: Sets LJM_MODBUS_MAP_CONSTANTS_FILE and LJM_ERROR_CONSTANTS_FILE at the same
 *       time, as an absolute or relative file path.
 * Note: Cannot be read, since LJM_MODBUS_MAP_CONSTANTS_FILE and
 *       LJM_ERROR_CONSTANTS_FILE can be different files.
**/
static const char * const LJM_CONSTANTS_FILE = "LJM_CONSTANTS_FILE";

/**
 * Name: LJM_DEBUG_LOG_FILE_MAX_SIZE
 * Desc: The maximum size of the log file in number of characters.
 * Note: This is an approximate limit.
**/
static const char * const LJM_DEBUG_LOG_FILE_MAX_SIZE = "LJM_DEBUG_LOG_FILE_MAX_SIZE";

/**
 * Name: LJM_DEEP_SEARCH_FILE
 * Desc: The file that specifies Deep Search IP ranges, which are IP address
 *       ranges that are directly checked for possible LabJack device UDP or TCP
 *       connections.
 * Note: For more details, see:
 *   https://labjack.com/support/software/api/ljm/constants/DeepSearchConfigs
**/
static const char * const LJM_DEEP_SEARCH_FILE = "LJM_DEEP_SEARCH_FILE";

/**
 * Name: LJM_SPECIFIC_IPS_FILE
 * Desc: The file that specifies Specific IPs, which are IP addresses that
 *       are specifically checked for possible LabJack device TCP connections.
 * Note: For more details, see:
 *   https://labjack.com/support/software/api/ljm/constants/SpecificIPsConfigs
**/
static const char * const LJM_SPECIFIC_IPS_FILE = "LJM_SPECIFIC_IPS_FILE";

/**
 * Name: LJM_STREAM_AIN_BINARY
 * Desc: Sets whether data returned from LJM_eStreamRead will be
 *       calibrated or uncalibrated.
 * Vals: 0 - Calibrated floating point AIN data (default)
 *       1 - Uncalibrated binary AIN data
**/
static const char * const LJM_STREAM_AIN_BINARY = "LJM_STREAM_AIN_BINARY";

/**
 * Name: LJM_STREAM_DIGITAL_AUTO_RECOVERY_ERROR_DETECTION_DISABLED
 * Desc: Sets LJM's behavior if the two following conditions are true:
 *         1. The first channel of stream is digital (excluding FIO_STATE,
 *            EIO_STATE, CIO_STATE, MIO_STATE, EIO_CIO_STATE, and CIO_MIO_STATE)
 *         2. Auto-recovery occurs
 *       If both of those conditions are true, stream will be stopped and
 *       LJM_eStreamRead return the error
 *       LJME_DIGITAL_AUTO_RECOVERY_ERROR_DETECTED, unless
 *       LJM_STREAM_DIGITAL_AUTO_RECOVERY_ERROR_DETECTION_DISABLED is set to 1,
 *       in which case LJM will perform auto-recovery as normal, while treating
 *       the first channel as if it will never return 0xFFFF. For more details,
 *       see: labjack.com/digital-auto-recovery-error-detection
 * Vals: 0 (default) - If auto-recovery happens and the first channel is digital
 *           (excluding FIO_STATE, EIO_STATE, CIO_STATE, MIO_STATE,
 *           EIO_CIO_STATE, and CIO_MIO_STATE), LJM_eStreamRead returns the
 *           error LJME_DIGITAL_AUTO_RECOVERY_ERROR_DETECTED.
 *       1 - If auto-recovery happens and the first channel is digital
 *           (excluding FIO_STATE, EIO_STATE, CIO_STATE, MIO_STATE,
 *           EIO_CIO_STATE, and CIO_MIO_STATE), LJM will treat the first channel
 *           as if it cannot return 0xFFFF and will insert the missing scans as
 *           LJM_DUMMY_VALUE values.
**/
static const char * const LJM_STREAM_DIGITAL_AUTO_RECOVERY_ERROR_DETECTION_DISABLED =
	"LJM_STREAM_DIGITAL_AUTO_RECOVERY_ERROR_DETECTION_DISABLED";

/**
 * Name: LJM_STREAM_SCANS_RETURN
 * Desc: Sets how LJM_eStreamRead will return data.
 * Note: Does not affect currently running or already initialized streams.
**/
static const char * const LJM_STREAM_SCANS_RETURN = "LJM_STREAM_SCANS_RETURN";
enum {
	/**
	 * Name: LJM_STREAM_SCANS_RETURN_ALL
	 * Desc: A mode that will cause LJM_eStreamRead to sleep until the full
	 *       ScansPerRead scans are collected by LJM.
	 * Note: ScansPerRead is a parameter of LJM_eStreamStart.
	 * Note: This mode may not be appropriate for stream types that are not
	 *       consistently timed, such as gate stream mode or external clock stream
	 *       mode.
	**/
	LJM_STREAM_SCANS_RETURN_ALL = 1,

	/**
	 * Name: LJM_STREAM_SCANS_RETURN_ALL_OR_NONE
	 * Desc: A mode that will cause LJM_eStreamRead to never sleep, and instead
	 *       either:
	 *           consume ScansPerRead scans and return LJME_NOERROR, or
	 *           consume no scans and return LJME_NO_SCANS_RETURNED.
	 *       LJM_eStreamRead will consume ScansPerRead if the LJM handle has
	 *       received ScansPerRead or more scans, otherwise it will consume none.
	 * Note: ScansPerRead is a parameter of LJM_eStreamStart.
	**/
	LJM_STREAM_SCANS_RETURN_ALL_OR_NONE = 2

	/**
	 * Name: LJM_STREAM_SCANS_RETURN_AVAILABLE
	 * Desc: A mode that will cause LJM_eStreamRead to never sleep, and always
	 *       consume the number of scans that the LJM handle has received, up to
	 *       a maximum of ScansPerRead. Fills the excess scan places in aData
	 *       not read, if any, with LJM_SCAN_NOT_READ.
	 * Note: ScansPerRead is a parameter of LJM_eStreamStart.
	 * TODO: LJM_STREAM_SCANS_RETURN_AVAILABLE is not currently implemented.
	**/
	// LJM_STREAM_SCANS_RETURN_AVAILABLE = 3
};

/**
 * Name: LJM_STREAM_RECEIVE_TIMEOUT_MODE
 * Desc: Sets how stream should time out.
 * Note: Does not affect currently running or already initialized streams.
**/
static const char * const LJM_STREAM_RECEIVE_TIMEOUT_MODE = "LJM_STREAM_RECEIVE_TIMEOUT_MODE";
enum {
	/**
	 * Name: LJM_STREAM_RECEIVE_TIMEOUT_MODE_CALCULATED
	 * Desc: Calculates how long the stream timeout should be, according to the
	 *       scan rate reported by the device.
	 * Note: This is the default LJM_STREAM_RECEIVE_TIMEOUT_MODE.
	**/
	LJM_STREAM_RECEIVE_TIMEOUT_MODE_CALCULATED = 1,

	/**
	 * Name: LJM_STREAM_RECEIVE_TIMEOUT_MODE_MANUAL
	 * Desc: Manually sets how long the stream timeout should be.
	 * Note: The actual stream timeout value is set via
	 *       LJM_STREAM_RECEIVE_TIMEOUT_MS.
	**/
	LJM_STREAM_RECEIVE_TIMEOUT_MODE_MANUAL = 2
};

/**
 * Name: LJM_STREAM_TCP_RECEIVE_BUFFER_SIZE
 * Desc: Sets the size of the OS TCP receive buffer in bytes for stream
 * Note: 0 is default (as set by the operating system)
 * Note: Does not affect currently running or already initialized streams.
 * Note: Setting this configuration to a non-zero value will probably disable
 *       operating system auto-tuning of the receive buffer size.
**/
static const char * const LJM_STREAM_TCP_RECEIVE_BUFFER_SIZE = "LJM_STREAM_TCP_RECEIVE_BUFFER_SIZE";

/**
 * Name: LJM_STREAM_THREADS_PRIORITY_MODE
 * Desc: Determines how LJM sets processor priority for LJM's internal stream
 *       threads.
**/
static const char * const LJM_STREAM_THREADS_PRIORITY_MODE = "LJM_STREAM_THREADS_PRIORITY_MODE";
enum {
	/**
	 * Name: LJM_STREAM_THREADS_PRIORITY_AUTO_ELEVATED
	 * Desc: When created, LJM sets elevated priority on internal stream
	 *       threads:
	 *        - Device data collection thread:
	 *            - Windows: THREAD_PRIORITY_TIME_CRITICAL
	 *            - Linux/macOS: Real-time SCHED_FIFO, priority 90
	 *        - Other LJM stream threads, including any callback threads (see
	 *          LJM_SetStreamCallback):
	 *            - Windows: THREAD_PRIORITY_HIGHEST
	 *            - Linux/macOS: Real-time SCHED_RR, priority 50
	 * Note: This is the default LJM_STREAM_THREADS_PRIORITY_MODE.
	 * Note: If thread priority cannot be set, a debug log message is
	 *       generated.
	**/
	LJM_STREAM_THREADS_PRIORITY_AUTO_ELEVATED = 1,

	/**
	 * Name: LJM_STREAM_THREADS_PRIORITY_UNALTERED
	 * Desc: LJM's internal stream threads inherit priority.
	**/
	LJM_STREAM_THREADS_PRIORITY_UNALTERED = 2
};

/**
 * Name: LJM_STREAM_PROCESS_PRIORITY_MODE
 * Desc: Sets whether or not LJM elevates the process priority temporarily
 *       during stream.
**/
static const char * const LJM_STREAM_PROCESS_PRIORITY_MODE = "LJM_STREAM_PROCESS_PRIORITY_MODE";
enum {
	/**
	 * Name; LJM_STREAM_PROCESS_PRIORITY_ELEVATED
	 * Desc: Elevates the process priority when LJM_eStreamStart or
	 *       LJM_StreamBurst is successfully called. When LJM_eStreamStop is
	 *       called or when LJM_StreamBurst ends, process priority is reset back
	 *       to what it was essentially when the first LJM call was made (or
	 *       NORMAL_PRIORITY_CLASS if the initial call to GetPriorityClass()
	 *       failed).
	 * Note: Windows: Elevates process priority to REALTIME_PRIORITY_CLASS.
	 * Note: Linux/macOS: Not implemented.
	 * Note: If the process priority cannot be set, a debug log message is
	 *       generated.
	 * Note: If multiple devices are streaming at once, the process priority is
	 *       only reset once LJM_eStreamStop has been called (or once
	 *       LJM_StreamBurst has ended) for each device that was streaming.
	 * Note: This is the default LJM_STREAM_PROCESS_PRIORITY_MODE.
	**/
	LJM_STREAM_PROCESS_PRIORITY_ELEVATED = 1,

	/**
	 * Name: LJM_STREAM_PROCESS_PRIORITY_UNALTERED
	 * Desc: The priority of the process is not altered.
	**/
	LJM_STREAM_PROCESS_PRIORITY_UNALTERED = 2
};

/**
 * Name: LJM_STREAM_RECEIVE_TIMEOUT_MS
 * Desc: Manually sets the stream receive timeout in milliseconds. Writing to
 *       this configuration sets LJM_STREAM_RECEIVE_TIMEOUT_MODE to be
 *       LJM_STREAM_RECEIVE_TIMEOUT_MODE_MANUAL.
 * Note: 0 is never timeout.
 * Note: Only affects currently running or already initialized streams if those
 *       streams were initialized with a LJM_STREAM_RECEIVE_TIMEOUT_MODE of
 *       LJM_STREAM_RECEIVE_TIMEOUT_MODE_MANUAL.
**/
static const char * const LJM_STREAM_RECEIVE_TIMEOUT_MS = "LJM_STREAM_RECEIVE_TIMEOUT_MS";

/**
 * Name: LJM_STREAM_TRANSFERS_PER_SECOND
 * Desc: Sets/gets the number of times per second stream threads attempt to
 *       read from the stream.
 * Note: Does not affect currently running or already initialized streams.
**/
static const char * const LJM_STREAM_TRANSFERS_PER_SECOND = "LJM_STREAM_TRANSFERS_PER_SECOND";

/**
 * Name: LJM_RETRY_ON_TRANSACTION_ID_MISMATCH
 * Desc: Sets/gets whether or not LJM will automatically retry an operation if
 *       an LJME_TRANSACTION_ID_ERR occurs.
 * Vals: 0 - Disable
 *       1 - Enable (default)
**/
static const char * const LJM_RETRY_ON_TRANSACTION_ID_MISMATCH = "LJM_RETRY_ON_TRANSACTION_ID_MISMATCH";

/**
 * Name: LJM_OLD_FIRMWARE_CHECK
 * Desc: Sets/gets whether or not LJM will check the constants file (see
 *       LJM_CONSTANTS_FILE) to make sure the firmware of the current device is
 *       compatible with the Modbus register(s) being read from or written to,
 *       when applicable. When device firmware is lower than fwmin for the
 *       register(s) being read/written, LJM will return LJME_OLD_FIRMWARE and
 *       not perform the Modbus operation(s).
 * Vals: 0 - Disable
 *       1 - Enable (default)
 * Note: When enabled, LJM will perform a check that is linear in size
 *       proportional to the number of register entries in the constants file
 *       for each address/name being read/written.
**/
static const char * const LJM_OLD_FIRMWARE_CHECK = "LJM_OLD_FIRMWARE_CHECK";

/**
 * Name: LJM_USE_TCP_INIT_FOR_T7_WIFI_TCP
 * Desc: Sets/gets whether LJM will use UDP or TCP for T7 WiFi connection
 *       initialization when ConnectionType is TCP.
 * Vals: 0 - Disable (use UDP)
 *       1 - Enable (default; use TCP)
**/
static const char * const LJM_USE_TCP_INIT_FOR_T7_WIFI_TCP = "LJM_USE_TCP_INIT_FOR_T7_WIFI_TCP";

/**
 * Name: LJM_ZERO_LENGTH_ARRAY_MODE
 * Desc: Determines the behavior of array read/write functions when the array
 *       size is 0.
**/
static const char * const LJM_ZERO_LENGTH_ARRAY_MODE = "LJM_ZERO_LENGTH_ARRAY_MODE";
enum {
	/**
	 * Name: LJM_ZERO_LENGTH_ARRAY_ERROR
	 * Desc: Sets LJM to return an error when an array of size 0 is detected.
	 * Note: This is the default LJM_ZERO_LENGTH_ARRAY_MODE.
	**/
	LJM_ZERO_LENGTH_ARRAY_ERROR = 1,

	/**
	 * Name: LJM_ZERO_LENGTH_ARRAY_IGNORE_OPERATION
	 * Desc: Sets LJM to ignore the operation when all arrays in the
	 *       operation are of size 0.
	**/
	LJM_ZERO_LENGTH_ARRAY_IGNORE_OPERATION = 2
};

// Config functions

/**
 * Name: LJM_WriteLibraryConfigS
 * Desc: Writes/sets a library configuration/setting.
 * Para: Parameter, the name of the configuration setting. Not
 *           case-sensitive. Must null-terminate.
 *       Value, the config value to apply to Parameter.
 * Retr: LJM_NOERROR for success,
 *       LJME_INVALID_CONFIG_NAME for a Parameter value that is unknown.
 * Note: See "Config Parameters" for valid Parameters and "Config Values" for
 *       valid Values.
**/
LJM_ERROR_RETURN LJM_WriteLibraryConfigS(const char * Parameter, double Value);

/**
 * Name: LJM_WriteLibraryConfigStringS
 * Desc: Writes/sets a library configuration/setting.
 * Para: Parameter, the name of the configuration setting. Not
 *           case-sensitive. Must null-terminate.
 *       String, the config value string to apply to Parameter. Must
 *           null-terminate. Must not be of size greater than
 *           LJM_MAX_NAME_SIZE, including null-terminator.
 * Retr: LJM_NOERROR for success,
 *       LJME_INVALID_CONFIG_NAME for a Parameter value that is unknown
 * Note: See "Config Parameters" for valid Parameters and "Config Values" for
 *       valid Values.
**/
LJM_ERROR_RETURN LJM_WriteLibraryConfigStringS(const char * Parameter,
	const char * String);

/**
 * Name: LJM_ReadLibraryConfigS
 * Desc: Reads a configuration/setting from the library.
 * Para: Parameter, the name of the configuration setting. Not
 *           case-sensitive. Must null-terminate.
 *       Value, return value representing the config value.
 * Retr: LJM_NOERROR for success,
 *       LJME_INVALID_CONFIG_NAME for a Parameter value that is unknown.
 * Note: See "Config Parameters" for valid Parameters and "Config Values" for
 *       valid Values.
**/
LJM_ERROR_RETURN LJM_ReadLibraryConfigS(const char * Parameter, double * Value);

/**
 * Name: LJM_ReadLibraryConfigStringS
 * Desc: Reads a configuration/setting from the library.
 * Para: Parameter, the name of the configuration setting. Not
 *           case-sensitive. Must null-terminate.
 *       string, return value representing the config string. Must be
 *           pre-allocated to size LJM_MAX_NAME_SIZE.
 * Retr: LJM_NOERROR for success,
 *       LJME_INVALID_CONFIG_NAME for a Parameter value that is unknown.
 * Note: See "Config Parameters" for valid Parameters and "Config Values" for
 *       valid Values.
**/
LJM_ERROR_RETURN LJM_ReadLibraryConfigStringS(const char * Parameter, char * String);

/**
 * Desc: Load all the configuration values in a specified file.
 * Para: FileName, a relative or absolute file location. "default" maps to the
 *           default configuration file ljm_startup_config.json in the
 *           constants file location. Must null-terminate.
**/
LJM_ERROR_RETURN LJM_LoadConfigurationFile(const char * FileName);

/**
 * Desc: Get information about whether the specific IPs file was parsed
 *       successfully. (See LJM_SPECIFIC_IPS_FILE)
 * Para: InfoHandle, a handle to Info that should be passed to LJM_CleanInfo
 *           after Info has been read.
 *       Info, a pointer to a JSON char * (allocated by LJM) describing the
 *           state of the specific IPs. Semantics:
 *           {
 *               "errorCode": Integer LJME_ error code. 0 indicates no error
 *               "IPs": Array of strings - the presentation-format IPs
 *               "message": Human-readable string description of success/failure
 *               "filePath": String absolute or relative file path
 *               "invalidIPs": Array of strings - the unparsable lines
 *           }
 * Retr: An error code indicating whether or not the Specific IP file was parsed
 *       successfully. This may be LJME_CONFIG_PARSING_ERROR even if some
 *       addresses in the Specific IP file were parsed without error.
**/
LJM_ERROR_RETURN LJM_GetSpecificIPsInfo(int * InfoHandle, const char ** Info);

/**
 * Desc: Get information about whether the Deep Search file was parsed
 *       successfully. (See LJM_DEEP_SEARCH_FILE)
 * Para: InfoHandle, a handle to Info that should be passed to LJM_CleanInfo
 *           after Info has been read.
 *       Info, a pointer to a JSON char * (allocated by LJM) describing the
 *           state of the Deep Search IPs. Semantics:
 *           {
 *               "errorCode": Integer LJME_ error code. 0 indicates no error
 *               "IPs": Array of strings - the presentation-format IPs
 *               "message": Human-readable string description of success/failure
 *               "filePath": String absolute or relative file path
 *               "invalidIPs": Array of strings - the unparsable lines
 *           }
 * Retr: An error code indicating whether or not the Deep Search file was parsed
 *       successfully. This may be LJME_CONFIG_PARSING_ERROR even if some
 *       addresses in the Deep Search file were parsed without error.
**/
LJM_ERROR_RETURN LJM_GetDeepSearchInfo(int * InfoHandle, const char ** Info);


/******************
 *  Log Functions *
 ******************/

/**
 * Name: LJM_Log
 * Desc: Sends a message of the specified level to the LJM debug logger.
 * Para: Level, the level to output the message at. See LJM_DEBUG_LOG_LEVEL.
 *       String, the debug message to be written to the log file.
 * Note: By default, LJM_DEBUG_LOG_MODE is to never log, so LJM does not output
 *       any log messages, even from this function.
 * Note: For more information on the LJM debug logger, see LJM_DEBUG_LOG_MODE,
 *       LJM_DEBUG_LOG_LEVEL, LJM_DEBUG_LOG_BUFFER_MAX_SIZE,
 *       LJM_DEBUG_LOG_SLEEP_TIME_MS, LJM_DEBUG_LOG_FILE,
 *       LJM_DEBUG_LOG_FILE_MAX_SIZE
**/
LJM_ERROR_RETURN LJM_Log(int Level, const char * String);

/**
 * Name: LJM_ResetLog
 * Desc: Clears all characters from the debug log file.
 * Note: See the LJM configuration properties for Log-related properties.
**/
LJM_ERROR_RETURN LJM_ResetLog(void);


/******************
 *  Reconnection  *
 ******************/

typedef void (*LJM_DeviceReconnectCallback)(int);

/**
 * Desc: Sets a callback that is called by LJM after the device is found to be
 *       disconnected (resulting in a read/write error) and the device is then
 *       reconnected.
 * Para: Handle, a valid handle to an open device.
 *       Callback, the callback function which will receive the device Handle
 *           as a parameter.
 * Note: To disable the previous callback for reconnect, pass 0 or NULL as
 *       Callback.
 * Note: LJM_RegisterDeviceReconnectCallback may not be called from within a
 *       LJM_DeviceReconnectCallback.
**/
LJM_ERROR_RETURN LJM_RegisterDeviceReconnectCallback(int Handle,
	LJM_DeviceReconnectCallback Callback);



/*****************************
 *  Deprecated declarations  *
 *****************************/

// These declarations will remain here for backwards compatibility, but they
// have been superseded by other declarations.

// Deprecated - use LJM_TCP_PORT instead
static const int LJM_DEFAULT_PORT = 502;

// Deprecated - use LJM_ETHERNET_UDP_PORT or LJM_WIFI_UDP_PORT instead
static const int LJM_UDP_PORT = 52362;

// Deprecated. Maximum packet size should instead be read from LJM_GetHandleInfo
static const int LJM_MAX_TCP_PACKET_NUM_BYTES_T7 = 1040;
static const int LJM_MAX_USB_PACKET_NUM_BYTES = 64;
static const int LJM_MAX_ETHERNET_PACKET_NUM_BYTES_T7 = 1040;
static const int LJM_MAX_WIFI_PACKET_NUM_BYTES_T7 = 500;

// Deprecated - use LJME_DEVICE_CURRENTLY_CLAIMED_BY_ANOTHER_PROCESS instead
LJM_ERROR_CODE LJME_COULD_NOT_CLAIM_DEVICE = 1230;

// Deprecated - use LJME_U3_NOT_SUPPORTED_BY_LJM instead
LJM_ERROR_CODE LJME_U3_CANNOT_BE_OPENED_BY_LJM = 1243;

// Deprecated - use LJME_U6_NOT_SUPPORTED_BY_LJM instead
LJM_ERROR_CODE LJME_U6_CANNOT_BE_OPENED_BY_LJM = 1246;

// Deprecated - use LJME_UE9_NOT_SUPPORTED_BY_LJM instead
LJM_ERROR_CODE LJME_UE9_CANNOT_BE_OPENED_BY_LJM = 1249;

// Deprecated - use LJME_UNKNOWN_VALUE_TYPE instead
LJM_ERROR_CODE LJME_INVALID_VALUE_TYPE = 1259;

// Deprecated - use LJM_SPECIFIC_IPS_FILE instead
static const char * const LJM_SPECIAL_ADDRESSES_FILE = "LJM_SPECIAL_ADDRESSES_FILE";

// Deprecated - use LJM_GetSpecificIPsInfo() instead
static const char * const LJM_SPECIAL_ADDRESSES_STATUS = "LJM_SPECIAL_ADDRESSES_STATUS";

// Deprecated
static const char * const LJM_OPEN_MODE = "LJM_OPEN_MODE";
enum { LJM_KEEP_OPEN = 1, LJM_OPEN_CLOSE = 2 };

#ifdef __cplusplus
}
#endif

#endif // #define LAB_JACK_M_HEADER
