/**
 * Name: eNames.c
 * Desc: Shows how to use the LJM_eNames function
 *
 * Relevant Documentation:
 *
 * LJM Library:
 *	LJM Library Installer:
 *		https://labjack.com/support/software/installers/ljm
 *	LJM Users Guide:
 *		https://labjack.com/support/software/api/ljm
 *	Opening and Closing:
 *		https://labjack.com/support/software/api/ljm/function-reference/opening-and-closing
 *	eNames:
 *		https://labjack.com/support/software/api/ljm/function-reference/ljmenames
 *
 * T-Series and I/O:
 *	Modbus Map:
 *		https://labjack.com/support/software/api/modbus/modbus-map
 *	Hardware Overview(Device Information Registers):
 *		https://labjack.com/support/datasheets/t-series/hardware-overview
**/

// For printf
#include <stdio.h>

// For the LabJackM library
#include "LabJackM.h"

// For LabJackM helper functions, such as OpenOrDie, PrintDeviceInfoFromHandle,
// ErrorCheck, etc.
#include "LJM_Utilities.h"

int main()
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
