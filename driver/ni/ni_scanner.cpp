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

#include "driver/ni/ni_reader.h"
#include "nisyscfg.h"
#include "nlohmann/json.hpp"



ni::Scanner::Scanner() {

    //initialize syscfg session for the scanner (TODO: Error Handling for status)
    NISysCfgStatus status = NISysCfg_OK;
    status = NISysCfgInitializeSession( //TODO: look into this
            "localhost",            // target (ip, mac or dns name)
            NULL,                   // username (NULL for local system)
            NULL,                   // password (NULL for local system)
            NISysCfgLocaleDefault,  // language
            NISysCfgBoolTrue,       //force pproperties to be queried everytime rather than cached
            10000,                  // timeout (ms)
            NULL,                   // expert handle
            &this->session                //session handle
    );
    

    this->filter = NULL;
    NISysCfgCreateFilter(this->session, &this->filter);
    NISysCfgSetFilterProperty(this->filter, NISysCfgFilterPropertyIsDevice, NISysCfgBoolTrue);
}

ni::Scanner::~Scanner() {
    // TODO: Error Handling
    NISysCfgCloseHandle(this->filter);
    NISysCfgCloseHandle(this->resourcesHandle);
    NISysCfgCloseHandle(this->session);
}

void ni::Scanner::scan(json properties) {

    NISysCfgResourceHandle resource = NULL;
    // TODO: use parser to verify there is a properties key
    auto property_arr = properties["properties"];

    // first find hardware
    auto err = NISysCfgFindHardware(this->session, NISysCfgFilterModeAll, this->filter, NULL, &this->resourcesHandle);
    if(err != NISysCfg_OK){
        return ""; // TODO: handle error more meaningfully
    }

    // Now iterate through found devices and get requested properties
    devices["devices"] = json::array();  

    while(NIsysCfgNextResource(this->session, this->resourcesHandle, &resource) == NISysCfg_OK){
        json device;
        for(auto &property: property_arr){
            char propertyValue[1024] = "";
            NISysCfgGetResourceProperty(resource, property, propertyValue);
            device[property] = propertyValue;
        }
        devices["devices"].push_back(device);
    }
}

void ni::Scanner::testConnection() {
    //TODO: Implement this
    return;
}

bool ni::Scanner::ok() {
    return ok_state;
}


// ni::NiScanner::NiScanner() {}

// ni::NiScanner::~NiScanner() {}

// json ni::NiScanner::getDevices() {
//     json j;
//     char productName[1024] = "";
//     char serialNumber[1024] = "";
//     char isSimulated[1024] = "";
//     char isDevice[1024] = "";
//     char isChassis[1024] = "";

//     NISysCfgStatus status = NISysCfg_OK;
//     NISysCfgEnumResourceHandle resourcesHandle = NULL;
//     NISysCfgResourceHandle resource = NULL;
//     NISysCfgFilterHandle filter = NULL;
//     NISysCfgSessionHandle session = NULL;

//     // initialized cfg session
//     status = NISysCfgInitializeSession( //TODO: look into this
//             "localhost",            // target (ip, mac or dns name)
//             NULL,                   // username (NULL for local system)
//             NULL,                   // password (NULL for local system)
//             NISysCfgLocaleDefault,  // language
//             NISysCfgBoolTrue,       //force pproperties to be queried everytime rather than cached
//             10000,                  // timeout (ms)
//             NULL,                   // expert handle
//             &session                //session handle
//     );

//     // create a filter to find only valid NI devices
//     NISysCfgCreateFilter(session, &filter);
//     NISysCfgSetFilterProperty(filter, NISysCfgFilterPropertyIsDevice, NISysCfgBoolTrue);

//     // Attempt to find hardware
//     auto err =  NISysCfgFindHardware(session, NISysCfgFilterModeAll, filter, NULL, &resourcesHandle);
//     if(err != NISysCfg_OK){
//         return ""; // TODO: handle error more meaningfully
//     }
//     j["devices"] = json::array();
//     // Iterate through all hardware found and grab the relevant information
//     while(NISysCfgNextResource(session, resourcesHandle, &resource)  ==  NISysCfg_OK) { // instead  do while (!= NISysCfgWarningNoMoreItems) ?
//         json device;
//         NISysCfgGetResourceProperty(resource, NISysCfgResourcePropertyProductName, productName);
//         NISysCfgGetResourceProperty(resource, NISysCfgResourcePropertySerialNumber, serialNumber);
//         NISysCfgGetResourceProperty(resource, NISysCfgResourcePropertyProductName, productName);
//         NISysCfgGetResourceProperty(resource, NISysCfgResourcePropertyIsSimulated, isSimulated);
//         NISysCfgGetResourceProperty(resource, NISysCfgResourcePropertyIsChassis, isChassis);
//         device["productName"] = productName;
//         device["serialNumber"] = serialNumber;
//         device["isSimulated"] = (isSimulated) ? 1 : 0;
//         device["isChassis"] = (isChassis) ? 1 : 0;
//         j["devices"].push_back(device);
//     }
//     NISysCfgCloseHandle(filter);
//     NISysCfgCloseHandle(resourcesHandle);
//     NISysCfgCloseHandle(session);
//     return j;
// }
