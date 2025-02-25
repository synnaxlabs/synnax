#include "driver/ni/read_task.h"

ni::ReadTaskConfig::ReadTaskConfig(ReadTaskConfig &&other):
    data_saving(other.data_saving),
    device_key(std::move(other.device_key)),
    sample_rate(other.sample_rate),
    stream_rate(other.stream_rate),
    timing_source(std::move(other.timing_source)),
    samples_per_channel(other.samples_per_channel),
    software_timed(other.software_timed),
    buffer_size(other.buffer_size),
    indexes(std::move(other.indexes)),
    channels(std::move(other.channels)) {
}

ni::ReadTaskConfig::ReadTaskConfig(
    std::shared_ptr<synnax::Synnax> &client,
    xjson::Parser &cfg, std::string task_type):
    data_saving(cfg.optional<bool>("data_saving", false)),
    device_key(cfg.optional<std::string>("device", "cross-device")),
    sample_rate(telem::Rate(cfg.required<float>("sample_rate"))),
    stream_rate(telem::Rate(cfg.required<float>("stream_rate"))),
    timing_source(cfg.optional<std::string>("timing_source", "none")),
    samples_per_channel(std::floor(sample_rate.value / stream_rate.value)),
    software_timed(this->timing_source == "none" && task_type == "ni_digital_read"),
    channels(cfg.map<std::unique_ptr<InputChan> >(
        "channels",
        [&](xjson::Parser &ch_cfg) -> std::pair<std::unique_ptr<InputChan>, bool> {
            auto ch = parse_input_chan(ch_cfg, {});
            return {std::move(ch), ch->enabled};
        }
    )) {
    if (this->channels.empty()) {
        cfg.field_err("channels", "task must have at least one channel");
        return;
    }
    std::vector<synnax::ChannelKey> channel_keys;
    for (const auto &ch: this->channels) channel_keys.push_back(ch->synnax_key);
    auto [channel_vec, err] = client->channels.retrieve(channel_keys);
    if (err) {
        cfg.field_err("", "failed to retrieve channels for task");
        return;
    }
    auto remote_channels = channel_keys_map(channel_vec);
    if (this->device_key != "cross-device") {
        auto [device, err] = client->hardware.retrieve_device(this->device_key);
        if (err) {
            cfg.field_err("", "failed to retrieve device for task");
            return;
        }
    }
    std::vector<std::string> dev_keys;
    for (const auto &ch: this->channels) dev_keys.push_back(ch->dev_key);
    auto [devices_vec, dev_err] = client->hardware.retrieve_devices(dev_keys);
    if (dev_err) {
        cfg.field_err("", "failed to retrieve devices for task");
        return;
    }
    auto devices = device_keys_map(devices_vec);
    for (auto &ch: this->channels) {
        auto remote_ch = remote_channels.at(ch->synnax_key);
        auto dev = devices[ch->dev_key];
        ch->bind_remote_info(remote_ch, dev.location);
        this->buffer_size += this->samples_per_channel * remote_ch.data_type.
                density();
        if (ch->ch.index != 0) this->indexes.insert(ch->ch.index);
    }
}

std::pair<ni::ReadTaskConfig, xerrors::Error> ni::ReadTaskConfig::parse(
    std::shared_ptr<synnax::Synnax> &client, const synnax::Task &task) {
    auto parser = xjson::Parser(task.config);
    return {ReadTaskConfig(client, parser, task.type), parser.error()};
}

xerrors::Error ni::ReadTaskConfig::apply(const std::shared_ptr<SugaredDAQmx> &dmx,
                                         TaskHandle handle) const {
    if (!this->software_timed)
        dmx->CfgSampClkTiming(
            handle,
            this->timing_source == "none" ? nullptr : this->timing_source.c_str(),
            this->sample_rate.value,
            DAQmx_Val_Rising,
            DAQmx_Val_ContSamps,
            this->sample_rate.value
        );
    for (const auto &ch: this->channels)
        if (auto err = ch->apply(dmx, handle)) return err;
    return xerrors::NIL;
}

synnax::WriterConfig ni::ReadTaskConfig::writer_config() const {
    std::vector<synnax::ChannelKey> keys;
    keys.reserve(this->channels.size() + this->indexes.size());
    for (const auto &ch: this->channels) keys.push_back(ch->ch.key);
    for (const auto &idx: this->indexes) keys.push_back(idx);
    return synnax::WriterConfig{
        .channels = keys,
        .mode = synnax::data_saving_writer_mode(this->data_saving)
    };
}

template<typename T>
ni::DAQmxHardwareInterface<T>::DAQmxHardwareInterface(
    TaskHandle task_handle, std::shared_ptr<SugaredDAQmx> dmx):
    task_handle(task_handle), dmx(std::move(dmx)) {
}

template<typename T>
xerrors::Error ni::DAQmxHardwareInterface<T>::start() const {
    return this->dmx->StartTask(this->task_handle);
}

template<typename T>
xerrors::Error ni::DAQmxHardwareInterface<T>::stop() const {
    return this->dmx->StopTask(this->task_handle);
}

xerrors::Error ni::AnalogHardwareInterface::read(
    const size_t samples_per_channel,
    std::vector<double> &data
) {
    return this->dmx->ReadAnalogF64(
        this->task_handle,
        samples_per_channel,
        -1,
        DAQmx_Val_GroupByChannel,
        data.data(),
        data.size(),
        nullptr,
        nullptr
    );
}

xerrors::Error ni::DigitalHardwareInterface::read(
    const size_t samples_per_channel,
    std::vector<uint8_t> &digital_data
) {
    return this->dmx->ReadDigitalLines(
        this->task_handle,
        samples_per_channel,
        -1,
        DAQmx_Val_GroupByChannel,
        digital_data.data(),
        digital_data.size(),
        nullptr,
        nullptr,
        nullptr
    );
}

template<typename T>
ni::ReadTask<T>::ReadTask(
    synnax::Task task,
    const std::shared_ptr<task::Context> &ctx,
    ReadTaskConfig cfg,
    const breaker::Config &breaker_cfg,
    std::unique_ptr<HardwareInterface<T> > hw_api
):
    task(std::move(task)),
    cfg(std::move(cfg)),
    ctx(ctx),
    tare_mw(std::make_shared<pipeline::TareMiddleware>(
        this->cfg.writer_config().channels)),
    source(std::make_shared<Source>(*this, std::move(hw_api))),
    pipe(
        this->ctx->client,
        this->cfg.writer_config(),
        this->source,
        breaker_cfg
    ) {
    this->pipe.add_middleware(this->tare_mw);
}

template<typename T>
void ni::ReadTask<T>::exec(task::Command &cmd) {
    if (cmd.type == "start") this->start(cmd.key);
    else if (cmd.type == "stop") this->stop(cmd.key);
    else if (cmd.type == "tare") this->tare_mw->tare(cmd.args);
}

template<typename T>
void ni::ReadTask<T>::stop() { this->source->breaker.stop(); }

template<typename T>
void ni::ReadTask<T>::stop(const std::string &cmd_key) {
    this->state.key = cmd_key;
    this->source->breaker.stop();
    this->source->sample_thread.join();
    this->pipe.stop();
    this->ctx->set_state(this->state);
}

template<typename T>
void ni::ReadTask<T>::start(const std::string &cmd_key) {
    this->state.key = cmd_key;
    this->pipe.start();
}