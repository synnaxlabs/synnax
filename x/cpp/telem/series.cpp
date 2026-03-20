// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "x/cpp/telem/proto.gen.h"
#include "x/cpp/telem/series.h"

namespace x::telem {

pb::Series Series::to_proto() const {
    pb::Series pb;
    auto [tr, err] = this->time_range.to_proto();
    *pb.mutable_time_range() = tr;
    pb.set_data_type(this->data_type_.name());
    pb.set_data(this->data_.get(), byte_size());
    pb.set_alignment(this->alignment.uint64());
    return pb;
}

std::pair<Series, x::errors::Error> Series::from_proto(const pb::Series &pb) {
    auto [tr, err] = TimeRange::from_proto(pb.time_range());
    if (err) return {Series(UNKNOWN_T, 0), err};
    return {Series(pb, tr), x::errors::NIL};
}

}
