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


ni::Scanner::Scanner(const std::shared_ptr<task::Context> &ctx,
                     const synnax::Task &task) : ctx(ctx), task(task)
{
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


    // create a filter to only identify NI devices rather than chassis and devices which are connected (which includes simulated devices)
    this->filter = NULL;
    ni::NiSysCfgInterface::CreateFilter(this->session, &this->filter);
    ni::NiSysCfgInterface::SetFilterProperty(this->filter, NISysCfgFilterPropertyIsDevice, NISysCfgBoolTrue);
    NISysCfgSetFilterProperty(filter, NISysCfgFilterPropertyIsPresent, NISysCfgIsPresentTypePresent);
    LOG(INFO) << "[ni.scanner] successfully configured scanner for task " << this->task.name;
}

ni::Scanner::~Scanner()
{
    // TODO: Error Handling
    ni::NiSysCfgInterface::CloseHandle(this->filter);
    ni::NiSysCfgInterface::CloseHandle(this->resourcesHandle);
    ni::NiSysCfgInterface::CloseHandle(this->session);
    LOG(INFO) << "[ni.scanner] successfully closed scanner for task " << this->task.name;
}

void ni::Scanner::scan()
{
    LOG(INFO) << "[ni.scanner] scanning devices for task " << this->task.name;
    NISysCfgResourceHandle resource = NULL;

    // first find hardware
    auto err = ni::NiSysCfgInterface::FindHardware(this->session, NISysCfgFilterModeAll, this->filter, NULL,
                                                   &this->resourcesHandle);
    if (err != NISysCfg_OK)
    {
        this->ok_state = false;
        return; // TODO: handle error more meaningfully
    }

    // Now iterate through found devices and get requested properties
    devices["devices"] = json::array();

    while (ni::NiSysCfgInterface::NextResource(this->session, this->resourcesHandle, &resource) == NISysCfg_OK)
    {   
        auto device = getDeviceProperties(resource);
        devices["devices"].push_back(device);
    }
    LOG(INFO) << "[ni.scanner] successfully scanned devices from task " << this->task.name;
}


json ni::Scanner::getDeviceProperties(NISysCfgResourceHandle resource){
    json device;

    char propertyValue[1024] = "";
    
    ni::NiSysCfgInterface::GetResourceProperty(resource, NISysCfgResourcePropertySerialNumber, propertyValue);
    device["serial_number"] = propertyValue;

    ni::NiSysCfgInterface::GetResourceProperty(resource, NISysCfgResourcePropertyProductName, propertyValue);
    std::string model = propertyValue;
    model = model.substr(3, model.size());
    device["model"] = model;

    ni::NiSysCfgInterface::GetResourceIndexedProperty(resource, NISysCfgIndexedPropertyExpertUserAlias, 0, propertyValue);
    device["location"] = propertyValue;

    ni::NiSysCfgInterface::GetResourceIndexedProperty(resource, NISysCfgIndexedPropertyExpertResourceName, 0, propertyValue);
    std::string rsrc_name = propertyValue;
    rsrc_name = rsrc_name.substr(1, rsrc_name.size() - 2);
    device["resource_name"] = rsrc_name;

    //get temp
    double temp;
    ni::NiSysCfgInterface::GetResourceProperty(resource, NISysCfgResourcePropertyCurrentTemp, &temp);
    device["temperature"] = temp;

    // check if its simulated
    NISysCfgBool isSimulated;
    ni::NiSysCfgInterface::GetResourceProperty(resource, NISysCfgResourcePropertyIsSimulated, &isSimulated);
    if(isSimulated){
        device["is_simulated"] = true;
        device["key"] = device["resource_name"];
    } else {
        device["is_simulated"] = false;
        device["key"] = device["serial_number"];
    }

    return device;
}

void ni::Scanner::createDevices(){
    for(auto &device : devices["devices"]){

        // first  try to rereive the device and if found, do not create a new device, simply continue
        auto [retrieved_device, err] = this->ctx->client->hardware.retrieveDevice(device["key"]);
        if(err == freighter::NIL){
            LOG(INFO) << "[ni.scanner] device " << device["model"] << " and key "  << device["key"] << "at location: " << device["location"] << " found for task " << this->task.name;
            continue;
        }
        auto new_device = synnax::Device(
            device["key"].get<std::string>(),
            device["resource_name"].get<std::string>(),
            synnax::taskKeyRack(this->task.key),                    // rack key
            device["location"].get<std::string>(),                  
            device["serial_number"].get<std::string>(),             
            "NI",                                                   // make                               
            device["model"].get<std::string>(),
            device.dump()                                           // device properties
        );
        
        if(this->ctx->client->hardware.createDevice(new_device) != freighter::NIL){
            LOG(ERROR) << "[ni.scanner] failed to create device " << device["model"] << " with key " << device["key"] << " for task " << this->task.name;
        }        
        LOG(INFO) << "[ni.scanner] successfully created device " << device["model"] <<  " with key " << device["key"] << " for task " << this->task.name;
    }

    
    // auto new_device = synnax::Device()


    // // no iterate through the set and retrieve device and print out name
    // for (auto &serial : device_serials)
    // {
    //     auto [device, err] = this->ctx->client->hardware.retrieveDevice(serial);
    //     if (err)
    //     {
    //         LOG(ERROR) << "[ni.scanner] failed to retrieve device with serial number " << serial;
    //     }
    //     else
    //     {
    //         LOG(INFO) << "[ni.scanner] retrieved device with serial number " << serial << " and name " << device.name;
    //     }
    // }

    // // scanned devices, now create them if they arent in the set.
    // for (auto &device : devices["devices"])
    // {
    //     if (device_serials.find(device["SerialNumber"]) == device_serials.end())
    //     {
    //         // add serial to set
    //         device_serials.insert(device["key"]);
    //         // create device
    //         auto new_device = synnax::Device({device["key"].get<std::string>(),
    //                                           device["DeviceName"].get<std::string>(),
    //                                           synnax::taskKeyRack(this->task.key),
    //                                           device["Location"].get<std::string>(),
    //                                           device["SerialNumber"].get<std::string>(),
    //                                           "NI",
    //                                           device["DeviceName"].get<std::string>().substr(3),
    //                                           device.dump()});
    //         this->ctx->client->hardware.createDevice(new_device);
    //     }
    // }

    // no iterate through the set and retrieve device and print out name
}




void ni::Scanner::testConnection(){
    // TODO: Implement this
    return;
}

bool ni::Scanner::ok(){
    return ok_state; // TODO: remove? only internal state
}

json ni::Scanner::getDevices(){
    return devices;
}

