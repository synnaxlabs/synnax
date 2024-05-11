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

#include "driver/ni/ni.h"
#include "nisyscfg_api.h"
#include "nlohmann/json.hpp"
#include <string>
#include <algorithm>

const std::vector <std::string> ni::Scanner::required_properties = {"DeviceName"};
const std::vector <std::string> ni::Scanner::optional_properties = {"SerialNumber"};


ni::Scanner::Scanner(const std::shared_ptr <task::Context> &ctx,
                     const synnax::Task &task) {

    this->ctx = ctx;
    this->task = task;

    // create parser 
    auto config_parser = config::Parser(task.config);
    this->parseConfig(config_parser);

    if (!this->ok()) {
        LOG(ERROR) << "[ni.scanner] failed to parse configuration for task " << this->task.name;
        this->ctx->setState({.task = task.key,
                                    .variant = "error",
                                    .details = config_parser.error_json()});
        return;
    }
    LOG(INFO) << "[ni.scanner] successfully parsed configuration for task " << this->task.name;

    // initialize syscfg session for the scanner (TODO: Error Handling for status)
    NISysCfgStatus status = NISysCfg_OK;
    status = ni::NiSysCfgInterface::InitializeSession( // TODO: look into this
            "localhost",                                   // target (ip, mac or dns name)
            NULL,                                          // username (NULL for local system)
            NULL,                                          // password (NULL for local system)
            NISysCfgLocaleDefault,                         // language
            NISysCfgBoolTrue,                              // force pproperties to be queried everytime rather than cached
            10000,                                         // timeout (ms)
            NULL,                                          // expert handle
            &this->session                                 // session handle
    );

    this->filter = NULL;
    ni::NiSysCfgInterface::CreateFilter(this->session, &this->filter);
    ni::NiSysCfgInterface::SetFilterProperty(this->filter, NISysCfgFilterPropertyIsDevice, NISysCfgBoolTrue);
    LOG(INFO) << "[ni.scanner] successfully configured scanner for task " << this->task.name;
}


void ni::Scanner::parseConfig(config::Parser &parser) {
    // this->requestedProperties = parser.required<json::array_t>("properties");
    json config = json::parse(this->task.config);
    if (!config.contains("properties")) {
        // print contains output
        json err;
        err["errors"] = "Missing properties key";
        this->ok_state = false;
        LOG(ERROR) << "[ni.scanner] failed to find properties key in configuration for task " << this->task.name;
        this->ctx->setState({.task = this->task.key,
                                    .variant = "error",
                                    .details = err});
        return;
    }
    this->requestedProperties = config["properties"];

    if (!this->requestedProperties.is_array() || this->requestedProperties.empty()) {
        json err;
        err["errors"] = "Invalid properties list";
        this->ok_state = false;
        LOG(ERROR) << "[ni.scanner] properties list is empty or not an array " << this->task.name;
        this->ctx->setState({.task = task.key,
                                    .variant = "error",
                                    .details = err});
        return;
    }
    // properties array not empty, check if all required properties are present
    for (auto property: required_properties) {
        // as long as there is one required property, this also makes sure any other elt in the array is also a string;
        auto it = std::find(this->requestedProperties.begin(), this->requestedProperties.end(), property);
        if (it == this->requestedProperties.end()) {
            json err;
            err["errors"] = "Missing required property: " + property;
            this->ok_state = false;
            LOG(ERROR) << "[ni.scanner] failed to find required property \"" << property
                       << "\"  in configuration for task " << this->task.name;
            this->ctx->setState({.task = task.key,
                                        .variant = "error",
                                        .details = err});
            return;
        }
    }
    // now check that every property in the list is either requried or
    std::vector <std::string> all_properties = ni::Scanner::required_properties;
    all_properties.insert(all_properties.end(), ni::Scanner::optional_properties.begin(),
                          ni::Scanner::optional_properties.end());
    for (auto &property: this->requestedProperties) {
        bool found = false;
        // just iterate through properties and continue if match
        for (auto &prop: all_properties) {
            if (!found && (property.get<std::string>() == prop)) {
                found = true;
            }
        }
        if (!found) {
            json err;
            err["errors"] = "Invalid property: " + property.get<std::string>();
            this->ok_state = false;
            LOG(ERROR) << "[ni.scanner] failed to find invalid property \"" << property.get<std::string>()
                       << "\" in configuration for task " << this->task.name;
            this->ctx->setState({.task = task.key,
                                        .variant = "error",
                                        .details = err});
        }
    }
}


ni::Scanner::~Scanner() {
    // TODO: Error Handling
    ni::NiSysCfgInterface::CloseHandle(this->filter);
    ni::NiSysCfgInterface::CloseHandle(this->resourcesHandle);
    ni::NiSysCfgInterface::CloseHandle(this->session);
    LOG(INFO) << "[ni.scanner] successfully closed scanner for task " << this->task.name;
}

void ni::Scanner::scan() {
    LOG(INFO) << "[ni.scanner] scanning devices for task " << this->task.name;
    NISysCfgResourceHandle resource = NULL;
    // TODO: use parser to verify there is a properties key
    auto property_arr = this->requestedProperties;

    // first find hardware
    auto err = ni::NiSysCfgInterface::FindHardware(this->session, NISysCfgFilterModeAll, this->filter, NULL,
                                                   &this->resourcesHandle);
    if (err != NISysCfg_OK) {
        this->ok_state = false;
        return; // TODO: handle error more meaningfully
    }

    // Now iterate through found devices and get requested properties
    devices["devices"] = json::array();

    while (ni::NiSysCfgInterface::NextResource(this->session, this->resourcesHandle, &resource) == NISysCfg_OK) {
        json device;
        for (auto &property_str: property_arr) {
            char propertyValue[1024] = "";
            auto property = getPropertyId(property_str);
            ni::NiSysCfgInterface::GetResourceProperty(resource, property, propertyValue);
            device[property_str] = propertyValue;
        }
        devices["devices"].push_back(device);
    }
    LOG(INFO) << "[ni.scanner] successfully scanned devices from task " << this->task.name;
}

void ni::Scanner::testConnection() {
    // TODO: Implement this
    return;
}

bool ni::Scanner::ok() {
    return ok_state;
}

json ni::Scanner::getDevices() {
    return devices;
}

NISysCfgResourceProperty ni::Scanner::getPropertyId(std::string property) {
    if (property == "SerialNumber") {
        return NISysCfgResourcePropertySerialNumber; // char *
    } else if (property == "DeviceName") {
        return NISysCfgResourcePropertyProductName; // char *
    } else {
        return NISysCfgResourcePropertyProductName; // default to product name
    }
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
