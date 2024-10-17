// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <stdio.h>
#include "LabJackM.h"
#include "LJM_Utilities.h"

int readNames()
{
    int err, frameI, arrayI, valueI, handle;
    int errorAddress = INITIAL_ERR_ADDRESS;

#define NUM_FRAMES 6

    const char * aNames[NUM_FRAMES] = {"DAC0", "TEST_UINT16", "TEST_UINT16", "SERIAL_NUMBER",
                                       "PRODUCT_ID", "FIRMWARE_VERSION"};
    int aWrites[NUM_FRAMES] = {LJM_WRITE, LJM_WRITE, LJM_READ, LJM_READ,
                               LJM_READ, LJM_READ};
    int aNumValues[NUM_FRAMES] = {1, 1, 1, 1, 1, 1};
    double aValues[6] = {2.5, 12345, 0.0, 0.0, 0.0};

    // Open first found LabJack
    err = LJM_Open(LJM_dtANY, LJM_ctANY, "LJM_idANY", &handle);
    ErrorCheck(err, "LJM_Open");

    PrintDeviceInfoFromHandle(handle);

    err = LJM_eNames(handle, NUM_FRAMES, aNames, aWrites, aNumValues,
                     aValues, &errorAddress);
    ErrorCheckWithAddress(err, errorAddress, "LJM_eNames");

    printf("\nLJM_eNames results:\n");
    valueI = 0;
    for (frameI=0; frameI<NUM_FRAMES; frameI++) {
        printf("\t");
        if (aWrites[frameI] == LJM_WRITE) {
            printf("Wrote");
        }
        else {
            printf("Read ");
        }
        printf(" - %s: [", aNames[frameI]);

        for (arrayI=0; arrayI<aNumValues[frameI]; arrayI++) {
            printf(" %f", aValues[valueI++]);
        }
        printf(" ]\n");
    }

    err = LJM_Close(handle);
    ErrorCheck(err, "LJM_Close");

    WaitForUserIfWindows();

    return LJME_NOERROR;
}

int scan(){
    int err;
    int i;
    int DeviceType = LJM_dtANY;
    int ConnectionType = LJM_ctANY;

    int aDeviceTypes[LJM_LIST_ALL_SIZE];
    int aConnectionTypes[LJM_LIST_ALL_SIZE];
    int aSerialNumbers[LJM_LIST_ALL_SIZE];
    int aIPAddresses[LJM_LIST_ALL_SIZE];
    int NumFound = 0;

    char IPv4String[LJM_IPv4_STRING_SIZE];

    printf("Calling LJM_ListAll with device type: %s, connection type: %s\n",
           NumberToDeviceType(DeviceType), NumberToConnectionType(ConnectionType));
    err = LJM_ListAll(DeviceType, ConnectionType, &NumFound, aDeviceTypes, aConnectionTypes,
                      aSerialNumbers, aIPAddresses);
    ErrorCheck(err, "LJM_ListAll with device type: %s, connection type: %s",
               NumberToDeviceType(DeviceType), NumberToConnectionType(ConnectionType));

    printf("Found %d device connections\n", NumFound);
    for (i=0; i<NumFound; i++) {
        err = LJM_NumberToIP(aIPAddresses[i], IPv4String);
        ErrorCheck(err, "LJM_NumberToIP");
        printf("[%3d]\naDeviceTypes: %s \naConnectionTypes: %s\n",
               i, NumberToDeviceType(aDeviceTypes[i]), NumberToConnectionType(aConnectionTypes[i]));
        printf("aSerialNumbers: %d,\naIPAddresses: %s (%u)\n\n",
               aSerialNumbers[i], IPv4String, aIPAddresses[i]);
    }

    WaitForUserIfWindows();

    return LJME_NOERROR;
}


int read_ain(){
    int err;
    int handle;

    // Set up for reading AIN value
    double value = 0;
    const char * NAME = "AIN0";

    // Open first found LabJack
    handle = OpenOrDie(LJM_dtANY, LJM_ctANY, "LJM_idANY");
    // handle = OpenSOrDie("LJM_dtANY", "LJM_ctANY", "LJM_idANY");

    PrintDeviceInfoFromHandle(handle);
    printf("\n");

    // Read AIN from the LabJack
    err = LJM_eReadName(handle, NAME, &value);
    ErrorCheck(err, "LJM_eReadName");

    // Print results
    printf("%s: %f V\n", NAME, value);

    CloseOrDie(handle);

    WaitForUserIfWindows();

    return LJME_NOERROR;
}


int read_di(){
    int err;
    int handle;

    // Set up for reading DIO state
    double value = 0;
    const char * name; // Changed from char * to const char * else get compile error

    // Open first found LabJack
    handle = OpenOrDie(LJM_dtANY, LJM_ctANY, "LJM_idANY");
    // handle = OpenSOrDie("LJM_dtANY", "LJM_ctANY", "LJM_idANY");

    PrintDeviceInfoFromHandle(handle);

    if (GetDeviceType(handle) == LJM_dtT4) {
        // Reading from FIO4 on the LabJack T4. FIO0-FIO3 are reserved for
        // AIN0-AIN3. Note: Reading a single digital I/O will change the line
        // from analog to digital input.
        name = "FIO4";
    }
    else {
        // Reading from FIO0 on the LabJack T7 and T8
        name = "FIO0";
    }

    // Read DIO state from the LabJack
    err = LJM_eReadName(handle, name, &value);
    ErrorCheck(err, "LJM_eReadName");

    printf("\n%s state : %f\n", name, value);

    CloseOrDie(handle);

    WaitForUserIfWindows();

    return LJME_NOERROR;
}

int write_di(){
    int err;
    int handle;

    // Set up for setting DIO state
    double value = 0; // Output state = low (0 = low, 1 = high)
    const char * name;

    // Open first found LabJack
    handle = OpenOrDie(LJM_dtANY, LJM_ctANY, "LJM_idANY");
    // handle = OpenSOrDie("LJM_dtANY", "LJM_ctANY", "LJM_idANY");

    PrintDeviceInfoFromHandle(handle);

    if (GetDeviceType(handle) == LJM_dtT4) {
        // Setting FIO4 on the LabJack T4. FIO0-FIO3 are reserved for AIN0-AIN3.
        name = "FIO4";

        // If the FIO/EIO line is an analog input, it needs to first be changed
        // to a digital I/O by reading from the line or setting it to digital
        // I/O with the DIO_ANALOG_ENABLE register.
        // For example:
        // 	double temp;
        // 	LJM_eReadName(handle, name, &temp);
    }
    else {
        // Setting FIO0 on the LabJack T7 and T8
        name = "FIO0";
    }

    // Set DIO state on the LabJack
    err = LJM_eWriteName(handle, name, value);
    ErrorCheck(err, "LJM_eWriteName");

    printf("\nSet %s state : %f\n", name, value);

    CloseOrDie(handle);

    WaitForUserIfWindows();

    return LJME_NOERROR;
}

int multi_ain(){
    int err, errorAddress;
    int handle;
    int i;
    int SkippedIntervals;
    int deviceType, ConnectionType, SerialNumber, IPAddress, Port,
            MaxBytesPerMB;
    const int INTERVAL_HANDLE = 1;

    // Set up for reading AIN values
    enum { NUM_FRAMES_AIN = 3 };
    double aValuesAIN[NUM_FRAMES_AIN] = {0};
    const char * aNamesAIN[NUM_FRAMES_AIN] = {"AIN0", "AIN1", "FIO4"};

    int msDelay = 1000; // sets sample rate?

    // Open first found LabJack
    handle = OpenOrDie(LJM_dtANY, LJM_ctANY, "LJM_idANY");
    // handle = OpenSOrDie("LJM_dtANY", "LJM_ctANY", "LJM_idANY");

    // Get device info
    err = LJM_GetHandleInfo(handle, &deviceType, &ConnectionType,
                            &SerialNumber, &IPAddress, &Port, &MaxBytesPerMB);
    ErrorCheck(err,
               "PrintDeviceInfoFromHandle (LJM_GetHandleInfo)");

    PrintDeviceInfo(deviceType, ConnectionType, SerialNumber, IPAddress, Port,
                    MaxBytesPerMB);

    // Setup and call eWriteNames to configure AIN resolution on the LabJack.
    WriteNameOrDie(handle, "AIN0_RESOLUTION_INDEX", 0);
    WriteNameOrDie(handle, "AIN1_RESOLUTION_INDEX", 0);

    // Range/gain configs only apply to the T7/T8
    if (deviceType != LJM_dtT4) {
        // Range = 10; This corresponds to ±10V (T7), or ±11V (T8)
        WriteNameOrDie(handle, "AIN0_RANGE", 10);
        WriteNameOrDie(handle, "AIN1_RANGE", 10);
    }
    // Negative channel = single ended (199). Only applies to the T7
    if (deviceType == LJM_dtT7) {
        WriteNameOrDie(handle, "AIN0_NEGATIVE_CH", 199);
        WriteNameOrDie(handle, "AIN1_NEGATIVE_CH", 199);
    }

    printf("\nStarting read loop.  Press Ctrl+c to stop.\n");

    err = LJM_StartInterval(INTERVAL_HANDLE, msDelay * 1000);
    ErrorCheck(err, "LJM_StartInterval");

    // Note: The LabJackM (LJM) library will catch the Ctrl+c signal, close
    //       all open devices, then exit the program.
    while (1) {
        // Read AIN from the LabJack
        err = LJM_eReadNames(handle, NUM_FRAMES_AIN, aNamesAIN, aValuesAIN,
                             &errorAddress);
        ErrorCheckWithAddress(err, errorAddress, "LJM_eReadNames");

        printf("%s : %f V, %s : %f V, %s : %f\n", aNamesAIN[0], aValuesAIN[0],
               aNamesAIN[1], aValuesAIN[1], aNamesAIN[2], aValuesAIN[2]);

        err = LJM_WaitForNextInterval(INTERVAL_HANDLE, &SkippedIntervals);
        ErrorCheck(err, "LJM_WaitForNextInterval");
        if (SkippedIntervals > 0) {
            printf("SkippedIntervals: %d\n", SkippedIntervals);
        }
    }

    err = LJM_CleanInterval(INTERVAL_HANDLE);
    PrintErrorIfError(err, "LJM_CleanInterval");

    CloseOrDie(handle);

    WaitForUserIfWindows();

    return LJME_NOERROR;
}


int main() {
    return multi_ain();
}
