// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// std
#include <string>
#include <regex>

/// internal
#include "driver/ni/scan_task.h"

ni::ScanTask::ScanTask(
    const std::shared_ptr<::syscfg::SugaredAPI> &syscfg,
    const std::shared_ptr<task::Context> &ctx,
    synnax::Task task,
    ScanTaskConfig cfg
) : task(std::move(task)),
    cfg(std::move(cfg)),
    timer(this->cfg.rate),
    ctx(ctx),
    syscfg(syscfg) {
    state.key = task.key;
}

const auto SKIP_DEVICE_ERR = xerrors::Error("ni.skip_device", "");

xerrors::Error ni::ScanTask::find_devices() {
    NISysCfgEnumResourceHandle resources = nullptr;
    NISysCfgResourceHandle curr_resource = nullptr;
    auto err = this->syscfg->FindHardware(
        this->session,
        NISysCfgFilterModeAll,
        this->filter,
        nullptr,
        &resources
    );
    if (err) return err;
    while (true) {
        if (const auto next_err = this->syscfg->NextResource(
            this->session,
            resources,
            &curr_resource
        )) {
            break;
        }
        auto [dev, parse_err] = this->parse_device(curr_resource);
        err = parse_err;
        if (err) continue;
        this->devices[dev.key] = dev;
        this->syscfg->CloseHandle(curr_resource);
    }
    auto close_err = this->syscfg->CloseHandle(resources);
    if (err.skip(SKIP_DEVICE_ERR)) return err;
    return close_err;
}


std::pair<ni::Device, xerrors::Error> ni::ScanTask::parse_device(
    NISysCfgResourceHandle resource
) const {
    char property_value_buf[1024];
    Device dev;
    dev.make = MAKE;
    dev.rack = synnax::task_key_rack(this->task.key);

    NISysCfgBool is_simulated;
    if (const auto err = this->syscfg->GetResourceProperty(
        resource,
        NISysCfgResourcePropertyIsSimulated,
        &is_simulated
    ))
        return {dev, err};
    dev.is_simulated = is_simulated;

    if (!is_simulated) {
        if (const auto err = this->syscfg->GetResourceProperty(
            resource,
            NISysCfgResourcePropertySerialNumber,
            property_value_buf
        ))
            return {Device(), err};
        dev.key = property_value_buf;
    }

    if (const auto err = this->syscfg->GetResourceProperty(
        resource,
        NISysCfgResourcePropertyProductName,
        property_value_buf
    ))
        return {Device(), err};
    dev.model = property_value_buf;
    if (dev.model.size() > 3) dev.model = dev.model.substr(3);
    dev.name = MAKE + " " + dev.model;

    if (const auto err = this->syscfg->GetResourceIndexedProperty(
        resource,
        NISysCfgIndexedPropertyExpertUserAlias,
        0,
        property_value_buf
    ))
        return {Device(), err};
    dev.location = property_value_buf;

    if (const auto err = this->syscfg->GetResourceIndexedProperty(
        resource,
        NISysCfgIndexedPropertyExpertResourceName,
        0,
        property_value_buf
    ))
        return {dev, err};
    dev.resource_name = property_value_buf;
    if (dev.resource_name.size() > 2)
        dev.resource_name = dev.resource_name.substr(
            1, dev.resource_name.size() - 2);
    if (is_simulated) dev.key = dev.resource_name;

    for (const auto &pattern: this->cfg.ignored_models)
        if (std::regex_match(dev.model, pattern))
            return {dev, xerrors::NIL};

    return {dev, xerrors::NIL};
}

xerrors::Error ni::ScanTask::update_remote() {
    const auto client = this->ctx->client;
    for (auto &[key, device]: devices) {
        auto [retrieved_device, err] = client->hardware.retrieve_device(key);
        if (err.matches(xerrors::NOT_FOUND)) {
            auto sy_dev = device.to_synnax();
            if (const auto c_err = client->hardware.create_device(sy_dev)) return c_err;
        } else if (err) return err;
    }
    return xerrors::NIL;
}

xerrors::Error ni::ScanTask::scan() {
    if (const auto err = this->find_devices()) return err;
    return this->update_remote();
}

std::pair<std::unique_ptr<task::Task>, xerrors::Error> ni::ScanTask::configure(
    const std::shared_ptr<syscfg::SugaredAPI> &syscfg,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    auto parser = xjson::Parser(task.config);
    auto cfg = ScanTaskConfig(parser);
    if (parser.error()) return {nullptr, parser.error()};
    auto tsk = std::make_unique<ni::ScanTask>(syscfg, ctx, task, cfg);
    tsk->start();
    return {std::move(tsk), xerrors::NIL};
}

void ni::ScanTask::start() {
    if (!this->breaker.start()) return;
    this->thread = std::thread(&ni::ScanTask::run, this);
}

void ni::ScanTask::stop(bool will_reconfigure) {
    if (!this->breaker.stop()) return;
    this->thread.join();
}

void ni::ScanTask::exec(task::Command &cmd) {
    this->state.key = cmd.key;
    if (cmd.type == "stop") return this->stop(false);
    xerrors::Error err = xerrors::NIL;
    if (cmd.type == "start") this->start();
    else if (cmd.type == "scan") err = this->scan();
    if (!err) return;
    this->state.variant = "error";
    this->state.details["message"] = err.message();
    this->ctx->set_state(this->state);
}

xerrors::Error ni::ScanTask::initialize_syscfg_session() {
    if (const auto err = this->syscfg->InitializeSession(
        nullptr, // target (ip, mac or dns name)
        nullptr, // username (NULL for local system)
        nullptr, // password (NULL for local system)
        NISysCfgLocaleDefault, // language
        NISysCfgBoolTrue,
        // force properties to be queried everytime rather than cached
        (this->cfg.rate.period() - telem::SECOND).milliseconds(),
        nullptr, // expert handle
        &this->session // session handle
    ))
        return err;

    if (const auto err = this->syscfg->CreateFilter(this->session, &this->filter))
        return err;
    if (const auto err = this->syscfg->SetFilterProperty(
        this->filter,
        NISysCfgFilterPropertyIsDevice,
        NISysCfgBoolTrue
    ))
        return err;
    if (const auto err = this->syscfg->SetFilterProperty(
        this->filter,
        NISysCfgFilterPropertyIsPresent,
        NISysCfgIsPresentTypePresent
    ))
        return err;
    if (const auto err = this->syscfg->SetFilterProperty(
        this->filter,
        NISysCfgFilterPropertyIsChassis,
        NISysCfgBoolFalse
    ))
        return err;
    return this->syscfg->SetFilterProperty(
        this->filter,
        NISysCfgFilterPropertyIsNIProduct,
        NISysCfgBoolTrue
    );
}

void ni::ScanTask::run() {
    if (const auto err = this->initialize_syscfg_session()) {
        this->state.variant = "error";
        this->state.details["message"] = err.message();
        this->ctx->set_state(this->state);
        return;
    }
    this->state.variant = "success";
    this->state.details["message"] = "scan task started";
    this->ctx->set_state(this->state);
    while (this->breaker.running()) {
        if (const auto err = this->scan()) {
            this->state.variant = "warning";
            this->state.details["message"] = err.message();
            this->ctx->set_state(this->state);
            LOG(WARNING) << "[ni.scan_task] failed to scan for devices: " << err;
        }
        this->timer.wait(breaker);
    }
    this->state.variant = "success";
    this->state.details["message"] = "scan task stopped";
    this->ctx->set_state(this->state);
    const auto f_err = this->syscfg->CloseHandle(this->filter);
    const auto s_err = this->syscfg->CloseHandle(this->session);
}
