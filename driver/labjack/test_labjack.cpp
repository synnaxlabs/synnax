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
#include "LJM_StreamUtilities.h"

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

void HardcodedConfigureStream(int handle)
{
    const int STREAM_TRIGGER_INDEX = 0;
    const int STREAM_CLOCK_SOURCE = 0;
    const int STREAM_RESOLUTION_INDEX = 0;
    const double STREAM_SETTLING_US = 0;
    const double AIN_ALL_RANGE = 0;
    const int AIN_ALL_NEGATIVE_CH = LJM_GND;

    printf("Writing configurations:\n");

    if (STREAM_TRIGGER_INDEX == 0) {
        printf("    Ensuring triggered stream is disabled:");
    }
    printf("    Setting STREAM_TRIGGER_INDEX to %d\n", STREAM_TRIGGER_INDEX);
    WriteNameOrDie(handle, "STREAM_TRIGGER_INDEX", STREAM_TRIGGER_INDEX);

    if (STREAM_CLOCK_SOURCE == 0) {
        printf("    Enabling internally-clocked stream:");
    }
    printf("    Setting STREAM_CLOCK_SOURCE to %d\n", STREAM_CLOCK_SOURCE);
    WriteNameOrDie(handle, "STREAM_CLOCK_SOURCE", STREAM_CLOCK_SOURCE);

    // Configure the analog inputs' negative channel, range, settling time and
    // resolution.
    // Note: when streaming, negative channels and ranges can be configured for
    // individual analog inputs, but the stream has only one settling time and
    // resolution.
    printf("    Setting STREAM_RESOLUTION_INDEX to %d\n",
           STREAM_RESOLUTION_INDEX);
    WriteNameOrDie(handle, "STREAM_RESOLUTION_INDEX", STREAM_RESOLUTION_INDEX);

    printf("    Setting STREAM_SETTLING_US to %f\n", STREAM_SETTLING_US);
    WriteNameOrDie(handle, "STREAM_SETTLING_US", STREAM_SETTLING_US);

    printf("    Setting AIN_ALL_RANGE to %f\n", AIN_ALL_RANGE);
    WriteNameOrDie(handle, "AIN_ALL_RANGE", AIN_ALL_RANGE);

    printf("    Setting AIN_ALL_NEGATIVE_CH to ");
    if (AIN_ALL_NEGATIVE_CH == LJM_GND) {
        printf("LJM_GND");
    }
    else {
        printf("%d", AIN_ALL_NEGATIVE_CH);
    }
    printf("\n");
    WriteNameOrDie(handle, "AIN_ALL_NEGATIVE_CH", AIN_ALL_NEGATIVE_CH);
}

void Stream(int handle, int numChannels, const char ** channelNames,
            double scanRate, int scansPerRead, int numReads)
{
    int err, iteration, channel;
    int numSkippedScans = 0;
    int totalSkippedScans = 0;
    int deviceScanBacklog = 0;
    int LJMScanBacklog = 0;
    unsigned int receiveBufferBytesSize = 0;
    unsigned int receiveBufferBytesBacklog = 0;
    int connectionType;

    int * aScanList = (int*)malloc(sizeof(int) * numChannels);

    unsigned int aDataSize = numChannels * scansPerRead;
    double * aData = (double*)malloc(sizeof(double) * aDataSize);


    err = LJM_GetHandleInfo(handle, NULL, &connectionType, NULL, NULL, NULL,
                            NULL);
    ErrorCheck(err, "LJM_GetHandleInfo");

    // Clear aData. This is not strictly necessary, but can help debugging.
    memset(aData, 0, sizeof(double) * aDataSize);

    err = LJM_NamesToAddresses(numChannels, channelNames, aScanList, NULL);
    ErrorCheck(err, "Getting positive channel addresses");

//    HardcodedConfigureStream(handle);

    printf("\n");
    printf("Starting stream...\n");
    err = LJM_eStreamStart(handle, scansPerRead, numChannels, aScanList,
                           &scanRate);
    ErrorCheck(err, "LJM_eStreamStart");
    printf("Stream started. Actual scan rate: %.02f Hz (%.02f sample rate)\n",
           scanRate, scanRate * numChannels);
    printf("\n");

    // Read the scans
    printf("Now performing %d reads\n", numReads);
    printf("\n");
    for (iteration = 0; iteration < numReads; iteration++) {
        err = LJM_eStreamRead(handle, aData, &deviceScanBacklog,
                              &LJMScanBacklog);
        ErrorCheck(err, "LJM_eStreamRead");

        printf("iteration: %d - deviceScanBacklog: %d, LJMScanBacklog: %d",
               iteration, deviceScanBacklog, LJMScanBacklog);
        if (connectionType != LJM_ctUSB) {
            err = LJM_GetStreamTCPReceiveBufferStatus(handle,
                                                      &receiveBufferBytesSize, &receiveBufferBytesBacklog);
            ErrorCheck(err, "LJM_GetStreamTCPReceiveBufferStatus");
            printf(", receive backlog: %f%%",
                   ((double)receiveBufferBytesBacklog) / receiveBufferBytesSize * 100);
        }
        printf("\n");
        printf("  1st scan out of %d:\n", scansPerRead);
        for (channel = 0; channel < numChannels; channel++) {
            printf("    %s = %0.5f\n", channelNames[channel], aData[channel]);
        }

//        for (channel = 0; channel < numChannels; channel++) {
//            for(int sample = 0; sample < 1000; sample++) {
//                printf("    %s = %0.5f\n", channelNames[channel], aData[channel * sample]);
//                printf("%0.5f\n", aData[channel * sample]);
//
//            }
//        }


        numSkippedScans = CountAndOutputNumSkippedSamples(numChannels, scansPerRead, aData);

        if (numSkippedScans) {
            printf("  %d skipped scans in this LJM_eStreamRead\n",
                   numSkippedScans);
            totalSkippedScans += numSkippedScans;
        }
        printf("\n");
    }
    if (totalSkippedScans) {
        printf("\n****** Total number of skipped scans: %d ******\n\n",
               totalSkippedScans);
    }

    printf("Stopping stream\n");
    err = LJM_eStreamStop(handle);
    ErrorCheck(err, "Stopping stream");

    free(aData);
    free(aScanList);
}

int basic_stream(){
    int handle;

    // How fast to stream in Hz
    double INIT_SCAN_RATE = 2000;

    // How many scans to get per call to LJM_eStreamRead. INIT_SCAN_RATE/2 is
    // recommended
    int SCANS_PER_READ = (int)INIT_SCAN_RATE / 1000;

    // How many times to call LJM_eStreamRead before calling LJM_eStreamStop
    const int NUM_READS = 1000;

    // Channels/Addresses to stream. NUM_CHANNELS can be less than or equal to
    // the size of CHANNEL_NAMES
    enum { NUM_CHANNELS = 3 };
    const char * CHANNEL_NAMES[] = {"AIN0", "AIN1", "FIO4"};

    // Open first found LabJack
    handle = OpenOrDie(LJM_dtANY, LJM_ctANY, "LJM_idANY");
    // handle = OpenSOrDie("LJM_dtANY", "LJM_ctANY", "LJM_idANY");

    PrintDeviceInfoFromHandle(handle);
    printf("\n");

    Stream(handle, NUM_CHANNELS, CHANNEL_NAMES, INIT_SCAN_RATE, SCANS_PER_READ,
           NUM_READS);

    CloseOrDie(handle);

    WaitForUserIfWindows();

    return LJME_NOERROR;
}

int digital_out()
{
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

    for(int i = 0 ; i < 100000; i++){
        value = 1;
        err = LJM_eWriteName(handle, name, value);
        ErrorCheck(err, "LJM_eWriteName");
        printf("\nSet %s state : %f\n", name, value);
        Sleep(100);
        value = 0;
        err = LJM_eWriteName(handle, name, value);
        ErrorCheck(err, "LJM_eWriteName");
        printf("\nSet %s state : %f\n", name, value);
        Sleep(100);
    }


    printf("\nSet %s state : %f\n", name, value);

    CloseOrDie(handle);

    WaitForUserIfWindows();

    return LJME_NOERROR;
}

void PrintErrorToString(int err)
{
    char errName[LJM_MAX_NAME_SIZE];
    LJM_ErrorToString(err, errName);
    printf("LJM_ErrorToString(%d) returned %s\n", err, errName);
}

int PrintErrors(){
    printf("Manual values:\n");
    PrintErrorToString(0);
    PrintErrorToString(LJME_CONSTANTS_FILE_NOT_FOUND);
    PrintErrorToString(LJME_INVALID_CONSTANTS_FILE);
    PrintErrorToString(LJME_TRANSACTION_ID_ERR);
    PrintErrorToString(LJME_WARNINGS_BEGIN);
    PrintErrorToString(LJME_U3_NOT_SUPPORTED_BY_LJM);
    PrintErrorToString(199); // non-existent error
    PrintErrorToString(2330); // LabJack device error

    WaitForUserIfWindows();

    return LJME_NOERROR;
}

int main() {
//    return PrintErrors();
//    return scan();
    return multi_ain();
//    return basic_stream();
//    return digital_out();
}
