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
        printf("aSerialNumbers: %d,\naIPAddresses: %s (%u)\n\m",
               aSerialNumbers[i], IPv4String, aIPAddresses[i]);
    }

    WaitForUserIfWindows();

    return LJME_NOERROR;
}




int main() {
    scan();
    return 0;
}