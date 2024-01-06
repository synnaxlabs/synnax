// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.


#include "synnax/device/device.h"

using namespace synnax;

Rack::Rack(RackKey key, std::string name):
        key(key),
        name(name) {
}

Rack::Rack(std::string name): name(name) {
}

Rack::Rack(const api::v1::Rack &a) :
        key(a.key()),
        name(a.name()) {
}

void Rack::to_proto(api::v1::Rack *rack) const {
    rack->set_key(key.value);
    rack->set_name(name);
}


const std::string RETRIEVE_RACK_ENDPOINT = "/device/rack/retrieve";
const std::string CREATE_RACK_ENDPOINT = "/device/rack/create";

std::pair<Rack, freighter::Error> DeviceClient::retrieveRack(std::uint64_t key) const {
    auto req = api::v1::DeviceRetrieveRackRequest();
    req.add_keys(key);
    auto [res, err] = rack_retrieve_client->send(RETRIEVE_RACK_ENDPOINT, req);
    if (err) return {Rack(), err};
    auto rack = Rack(res.racks(0));
    rack.modules = ModuleClient(rack.key, module_create_client, module_retrieve_client, module_delete_client);
    return {rack, err};
}

freighter::Error DeviceClient::createRack(Rack &rack) const {
    auto req = api::v1::DeviceCreateRackRequest();
    rack.to_proto(req.add_racks());
    auto [res, err] = rack_create_client->send(CREATE_RACK_ENDPOINT, req);
    if (err) return err;
    rack.key = res.racks().at(0).key();
    rack.modules = ModuleClient(rack.key, module_create_client, module_retrieve_client, module_delete_client);
    return err;
}

freighter::Error DeviceClient::deleteRack(std::uint64_t key) const {
    auto req = api::v1::DeviceDeleteRackRequest();
    req.add_keys(key);
    auto [res, err] = rack_delete_client->send(CREATE_RACK_ENDPOINT, req);
    return err;
}

Module::Module(ModuleKey key, std::string name, std::string type, std::string config):
        key(key),
        name(name),
        type(type),
        config(config) {
}

Module::Module(RackKey rack, std::string name, std::string type, std::string config):
        key(ModuleKey(rack, 0)),
        name(name),
        type(type),
        config(config) {
}

Module::Module(const api::v1::Module &a) :
        key(a.key()),
        name(a.name()),
        type(a.type()),
        config(a.config()) {
}

void Module::to_proto(api::v1::Module *module) const {
    module->set_key(key);
    module->set_name(name);
    module->set_type(type);
    module->set_config(config);
}

const std::string RETRIEVE_MODULE_ENDPOINT = "/device/module/retrieve";
const std::string CREATE_MODULE_ENDPOINT = "/device/module/create";
const std::string DELETE_MODULE_ENDPOINT = "/device/module/delete";

std::pair<Module, freighter::Error> ModuleClient::retrieve(std::uint64_t key) const {
    auto req = api::v1::DeviceRetrieveModuleRequest();
    req.add_keys(key);
    auto [res, err] = module_retrieve_client->send(RETRIEVE_MODULE_ENDPOINT, req);
    if (err) return {Module(), err};
    return {Module(res.modules(0)), err};
}

freighter::Error ModuleClient::create(Module &module) const {
    auto req = api::v1::DeviceCreateModuleRequest();
    module.to_proto(req.add_modules());
    auto [res, err] = module_create_client->send(CREATE_MODULE_ENDPOINT, req);
    if (err) return err;
    module.key = res.modules().at(0).key();
    return err;
}

freighter::Error ModuleClient::del(std::uint64_t key) const {
    auto req = api::v1::DeviceDeleteModuleRequest();
    req.add_keys(key);
    auto [res, err] = module_delete_client->send(DELETE_MODULE_ENDPOINT, req);
    return err;
}

std::pair<std::vector<Module>, freighter::Error> ModuleClient::list() const {
    auto req = api::v1::DeviceRetrieveModuleRequest();
    req.set_rack(rack.value);
    auto [res, err] = module_retrieve_client->send(RETRIEVE_MODULE_ENDPOINT, req);
    if (err) return {std::vector<Module>(), err};
    std::vector<Module> modules = {res.modules().begin(), res.modules().end()};
    return {modules, err};
}


