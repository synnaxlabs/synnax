#include <vector>

#include "v1/framer.pb.h"
#include "synnax/framer/framer.h"

using namespace Synnax::Framer;

Frame::Frame(std::vector<ChannelKey> *channels, std::vector<Telem::Series> *series) {
    this->columns = channels;
    this->series = series;
}

Frame::Frame(const api::v1::Frame &f) {
    auto key = f.keys();
    columns = new std::vector<ChannelKey>(key.begin(), key.end());
    series = new std::vector<Telem::Series>();
    series->reserve(f.series_size());
    for (auto &ser: f.series()) series->push_back(Telem::Series(ser));
}

void Frame::to_proto(api::v1::Frame *f) const {
    for (auto &key: *columns) f->add_keys(key);
    f->mutable_series()->Reserve(series->size());
    for (auto &ser: *series) ser.to_proto(f->add_series());
}
