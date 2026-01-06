// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * Name: LJM_Utilities.h
 * Desc: Provides some basic helper functions
**/

#pragma once

#ifdef _WIN32
	#ifndef WIN32_LEAN_AND_MEAN
	#define WIN32_LEAN_AND_MEAN
	#endif
	#include <winsock2.h>
    #include <windows.h>
	#include <ws2tcpip.h>
#else
	#include <unistd.h> // For sleep() (with Mac OS or Linux).
	#include <arpa/inet.h>  // For inet_ntoa()
	#include <sys/time.h>
#endif

#include <stdlib.h>
#include <stdio.h>
#include <stdarg.h>

#include "LabJackM.h"

#define COMMAND 2
#define RESPONSE 0

#define INITIAL_ERR_ADDRESS -2
// This is just something negative so normal addresses are not confused with it


// Prompts the user to press enter
void WaitForUser();

// Calls WaitForUser if this is compiled on Windows. This call is intended to be
// placed at the end of a program to prevent output from being lost in the case
// that it is being called from Visual Studio.
void WaitForUserIfWindows();

// Returns a new handle or exits on error with a description of the error.
int OpenOrDie(int deviceType, int connectionType, const char * identifier);
int OpenSOrDie(const char *  deviceType, const char *  connectionType,
	const char * identifier);

// Closes the handle or exits on error with a description of the error.
void CloseOrDie(int handle);

// Print device info
void PrintDeviceInfoFromHandle(int handle);
void PrintDeviceInfo(int deviceType, int connectionType, int serialNumber,
	int ipAddressInt, int portOrPipe, int MaxBytesPerMB);

// Queries the device for value
double Get(int handle, const char * valueName);

// Queries the device for value and prints the results
double GetAndPrint(int handle, const char * valueName);

// Queries the device for value and prints the results as an IP string
void GetAndPrintIPAddress(int handle, const char * valueName);

// Queries the device for value and prints the results as an MAC string.
// valueName is just a description, while valueAddress the address that
// determines what value is actually read from the device. valueAddress must be
// of type UINT64
void GetAndPrintMACAddressFromValueAddress(int handle, const char * valueName,
	int valueAddress);

// Queries the device for an address and prints the results
void GetAndPrintAddressAndType(int handle, const char * valueDescription,
	int address, int type);

// Queries LJM for the config value via LJM_ReadLibraryConfigS.
// Passes configParameter as Parameter and prints Value.
void GetAndPrintConfigValue(const char * configParameter);

// Queries LJM for the config value via LJM_ReadLibraryConfigStringS.
// Passes configParameter as Parameter and prints Value.
void GetAndPrintConfigString(const char * configParameter);

// Prints the current LJM logging configurations
void DisplayDebugLoggingConfigurations();

// Sets the LJM config value via LJM_WriteLibraryConfigS.
// Outputs error, if any
void SetConfigValue(const char * configParameter, double value);

// Sets the LJM config value via LJM_WriteLibraryConfigStringS.
// Outputs error, if any
void SetConfigString(const char * configParameter, const char * string);

// Performs an LJM_eWriteName call.
// Prints error, if any. The "OrDie" version exits the program upon error
void WriteNameOrDie(int handle, const char * name, double value);
int WriteName(int handle, const char * name, double value);

void WriteNameAltTypeOrDie(int handle, const char * name, int type,
	double value);

// Performs an LJM_eWriteNames call.
// Prints error, if any. The "OrDie" version exits the program upon error
void WriteNamesOrDie(int handle, int NumFrames, const char ** aNames,
	const double * aValues);
int WriteNames(int handle,  int NumFrames, const char ** aNames,
	const double * aValues, int * errorAddress);

// Performs a LJM_eWriteNameArray/LJM_eWriteNameByteArray call.
// On error, prints message and exits the program.
void WriteNameArrayOrDie(int handle, const char * name, int numValues,
	const double * aValues);
void WriteNameByteArrayOrDie(int handle, const char * name, int numBytes,
	const char * aBytes);

// Performs a LJM_eReadNameArray/LJM_eReadNameByteArray call.
// On error, prints message and exits the program.
void ReadNameArrayOrDie(int handle, const char * name, int numValues,
	double * aValues);
void ReadNameByteArrayOrDie(int handle, const char * name, int numBytes,
	char * aBytes);

/**
 * Name: ErrorCheck
 * Desc: If err is not LJME_NOERROR, displays the error and exits the program.
 * Para: err, the error code being checked
 *       formattedDescription, a string representing what sort of operation
 *                             returned err. This can have formatting arguments
 *                             in it, just like a string passed to printf
**/
void ErrorCheck(int err, const char * formattedDescription, ...);

/**
 * Desc: Prints the error if there is an error
**/
void PrintErrorIfError(int err, const char * formattedDescription, ...);
void PrintErrorWithAddressIfError(int err, int errAddress,
	const char * formattedDescription, ...);

// Displays the error, error name, and value name
void CouldNotRead(int err, const char * valueName);

/**
 * Name: ErrorCheckWithAddress
 * Desc: If err is not LJME_NOERROR, displays the error and exits the program.
 *       If errAddress is not negative, outputs the error address corresponding
 *       to err
 * Para: err, the error code being checked
 *       formattedDescription, a string representing what sort of operation
 *                             returned err. This can have formatting arguments
 *                             in it, just like a string passed to printf
 *       errAddress, the address corresponding to err. If this is negative, it
 *                   will not be output. Note that LJM functions such as
 *                   LJM_eReadAddresses will not change the ErrorAddress unless
 *                   relevant
**/
void ErrorCheckWithAddress(int err, int errAddress,
	const char * formattedDescription, ...);

// Sleeps for the given number of milliseconds
void MillisecondSleep(unsigned int milliseconds);

// Turns on the specified level of logging
void EnableLoggingLevel(double logLevel);

/**
 * Desc: Gets the device type of an open device handle.
**/
int GetDeviceType(int handle);

/**
 * Name: NumberToDebugLogMode
 * Desc: Takes an integer representing a LJM_DEBUG_LOG_MODE and returns the mode
 *       name as a string
 * Retr: the mode name, or "Unknown LJM_DEBUG_LOG_MODE" if mode is not
 *       recognized
 * Note: See LJM_DEBUG_LOG_MODE in LabJackM.h
**/
const char * NumberToDebugLogMode(int mode);

/**
 * Desc: Returns the current CPU time in milliseconds
**/
unsigned int GetCurrentTimeMS();

/**
 * Desc: Returns IPv4String in integer form, handling error by calling
 *       ErrorAddress
**/
unsigned int IPToNumber(const char * IPv4String);

/**
 * Desc: Returns 0 (false) if v1 and v2 are not within delta of each other,
 *       returns 1 (true) if they are within delta each other
**/
int EqualFloats(double v1, double v2, double delta);

/**
 * Desc: Returns 1 (true) if connectionType is TCP-based,
 *       returns 0 (false) if not.
**/
int IsTCP(int connectionType);

/**
 * Desc: Returns the address of named register.
**/
int GetAddressFromNameOrDie(const char * name);

/**
 * Desc: Returns true if connectionType is a network type,
 *       returns false if connectionType is USB
**/
int IsNetwork(int connectionType);

/**
 * Desc: Determines whether or not a device has WiFi capability. E
 * Retr: Returns 1 if the device referred to by handle has WiFi installed,
 *       returns 0 otherwise
**/
int DoesDeviceHaveWiFi(int handle);

inline const char * NumberToConnectionType(int connectionType)
{
	switch (connectionType) {
	case LJM_ctANY:          return "LJM_ctANY";
	case LJM_ctUSB:          return "LJM_ctUSB";
	case LJM_ctTCP:          return "LJM_ctTCP";
	case LJM_ctETHERNET:     return "LJM_ctETHERNET";
	case LJM_ctWIFI:         return "LJM_ctWIFI";
	case LJM_ctNETWORK_UDP:  return "LJM_ctNETWORK_UDP";
	case LJM_ctETHERNET_UDP: return "LJM_ctETHERNET_UDP";
	case LJM_ctWIFI_UDP:     return "LJM_ctWIFI_UDP";
	case LJM_ctNETWORK_ANY:  return "LJM_ctNETWORK_ANY";
	case LJM_ctETHERNET_ANY: return "LJM_ctETHERNET_ANY";
	case LJM_ctWIFI_ANY:     return "LJM_ctWIFI_ANY";
	default:                 return "Unknown connection type";
	}
}

inline const char * NumberToDeviceType(int deviceType)
{
	switch (deviceType) {
	case LJM_dtANY:     return "LJM_dtANY";
	case 4:             return "LJM_dtT4";
	case LJM_dtT7:      return "LJM_dtT7";
	case 84:            return "LJM_dtTSERIES";
	case LJM_dtDIGIT:   return "LJM_dtDIGIT";
	case -4:            return "Demo fake usb";
	default:
		printf(
			"%s:%d NumberToDeviceType: Unknown device type: %d\n",
			__FILE__,
			__LINE__,
			deviceType
		);
		return "Unknown device type";
	}
}