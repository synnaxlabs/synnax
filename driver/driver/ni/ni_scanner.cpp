// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//
// Created by Synnax on 3/24/2024.
//

#include "ni_scanner.h"
#include "nisyscfg.h"
#include "nlohmann/json.hpp"

ni::NiScanner::NiScanner() {}

ni::NiScanner::~NiScanner() {}

json ni::NiScanner::getDevices() {
    json j;
    char productName[1024] = "";
    char serialNumber[1024] = "";
    char isSimulated[1024] = "";
    char isDevice[1024] = "";
    char isChassis[1024] = "";

    NISysCfgStatus status = NISysCfg_OK;
    NISysCfgEnumResourceHandle resourcesHandle = NULL;
    NISysCfgResourceHandle resource = NULL;
    NISysCfgFilterHandle filter = NULL;
    NISysCfgSessionHandle session = NULL;

    // initialized cfg session
    status = NISysCfgInitializeSession( //TODO: look into this
            "localhost",            // target (ip, mac or dns name)
            NULL,                   // username (NULL for local system)
            NULL,                   // password (NULL for local system)
            NISysCfgLocaleDefault,  // language
            NISysCfgBoolTrue,       //force pproperties to be queried everytime rather than cached
            10000,                  // timeout (ms)
            NULL,                   // expert handle
            &session                //session handle
    );

    // create a filter to find only valid NI devices
    NISysCfgCreateFilter(session, &filter);
    NISysCfgSetFilterProperty(filter, NISysCfgFilterPropertyIsDevice, NISysCfgBoolTrue);

    // Attempt to find hardware
    auto err =  NISysCfgFindHardware(session, NISysCfgFilterModeAll, filter, NULL, &resourcesHandle);
    if(err != NISysCfg_OK){
        return ""; // TODO: handle error more meaningfully
    }
    j["devices"] = json::array();
    // Iterate through all hardware found and grab the relevant information
    while(NISysCfgNextResource(session, resourcesHandle, &resource)  ==  NISysCfg_OK) { // instead  do while (!= NISysCfgWarningNoMoreItems) ?
        json device;
        NISysCfgGetResourceProperty(resource, NISysCfgResourcePropertyProductName, productName);
        NISysCfgGetResourceProperty(resource, NISysCfgResourcePropertySerialNumber, serialNumber);
        NISysCfgGetResourceProperty(resource, NISysCfgResourcePropertyProductName, productName);
        NISysCfgGetResourceProperty(resource, NISysCfgResourcePropertyIsSimulated, isSimulated);
        NISysCfgGetResourceProperty(resource, NISysCfgResourcePropertyIsChassis, isChassis);
        device["productName"] = productName;
        device["serialNumber"] = serialNumber;
        device["isSimulated"] = (isSimulated) ? 1 : 0;
        device["isChassis"] = (isChassis) ? 1 : 0;
        j["devices"].push_back(device);
    }
    NISysCfgCloseHandle(filter);
    NISysCfgCloseHandle(resourcesHandle);
    NISysCfgCloseHandle(session);
    return j;
}
