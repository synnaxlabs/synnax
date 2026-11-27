#include <set>
#include <vector>

#include "benchmark/benchmark.h"

#include "client/cpp/framer/framer.h"
#include "driver/common/read_task.h"
#include "driver/common/status.h"

namespace {
constexpr size_t N_CH = 10;

x::telem::Frame make_frame(const size_t n) {
    x::telem::Frame fr(N_CH + 1);
    for (size_t c = 0; c < N_CH; c++) {
        x::telem::Series s(x::telem::FLOAT64_T, n);
        for (size_t i = 0; i < n; i++)
            s.write(static_cast<double>(i));
        fr.emplace(c + 1, std::move(s));
    }
    x::telem::Series idx(x::telem::TIMESTAMP_T, n);
    for (size_t i = 0; i < n; i++)
        idx.write(static_cast<int64_t>(i));
    fr.emplace(100, std::move(idx));
    return fr;
}

synnax::framer::Codec make_codec() {
    std::vector<synnax::channel::Key> keys;
    std::vector<x::telem::DataType> dts;
    for (size_t c = 0; c < N_CH; c++) {
        keys.push_back(c + 1);
        dts.push_back(x::telem::FLOAT64_T);
    }
    keys.push_back(100);
    dts.push_back(x::telem::TIMESTAMP_T);
    return synnax::framer::Codec(keys, dts);
}

struct Ch {
    synnax::channel::Key synnax_key;
    synnax::channel::Channel ch;
};
}

static void BM_SeriesAtTyped(benchmark::State &st) {
    const size_t n = st.range(0);
    x::telem::Series s(x::telem::FLOAT64_T, n);
    for (size_t i = 0; i < n; i++)
        s.write(1.0 * i);
    for (auto _: st) {
        double sum = 0;
        for (size_t i = 0; i < n; i++)
            sum += s.at<double>(static_cast<int>(i));
        benchmark::DoNotOptimize(sum);
    }
    st.SetItemsProcessed(st.iterations() * n);
}
BENCHMARK(BM_SeriesAtTyped)->Arg(400);

static void BM_SeriesRawRead(benchmark::State &st) {
    const size_t n = st.range(0);
    x::telem::Series s(x::telem::FLOAT64_T, n);
    for (size_t i = 0; i < n; i++)
        s.write(1.0 * i);
    for (auto _: st) {
        double sum = 0;
        const auto *p = reinterpret_cast<const double *>(s.data());
        for (size_t i = 0; i < n; i++)
            sum += p[i];
        benchmark::DoNotOptimize(sum);
    }
    st.SetItemsProcessed(st.iterations() * n);
}
BENCHMARK(BM_SeriesRawRead)->Arg(400);

static void BM_SeriesAtVariant(benchmark::State &st) {
    const size_t n = st.range(0);
    x::telem::Series s(x::telem::FLOAT64_T, n);
    for (size_t i = 0; i < n; i++)
        s.write(1.0 * i);
    for (auto _: st) {
        double sum = 0;
        for (size_t i = 0; i < n; i++)
            sum += std::get<double>(s.at(static_cast<int>(i)));
        benchmark::DoNotOptimize(sum);
    }
    st.SetItemsProcessed(st.iterations() * n);
}
BENCHMARK(BM_SeriesAtVariant)->Arg(400);

static void BM_SeriesWriteTyped(benchmark::State &st) {
    const size_t n = st.range(0);
    x::telem::Series s(x::telem::FLOAT64_T, n);
    for (auto _: st) {
        s.clear();
        for (size_t i = 0; i < n; i++)
            s.write(1.0 * i);
        benchmark::DoNotOptimize(s.data());
    }
    st.SetItemsProcessed(st.iterations() * n);
}
BENCHMARK(BM_SeriesWriteTyped)->Arg(400);

static void BM_DataTypeCopy(benchmark::State &st) {
    x::telem::Series s(x::telem::TIMESTAMP_T, 1);
    for (auto _: st)
        benchmark::DoNotOptimize(s.data_type().density());
}
BENCHMARK(BM_DataTypeCopy);

static void BM_SourceFillFrame(benchmark::State &st) {
    const size_t n = st.range(0);
    std::vector<std::unique_ptr<Ch>> chans;
    for (size_t c = 0; c < N_CH; c++) {
        auto ch = std::make_unique<Ch>();
        ch->synnax_key = c + 1;
        ch->ch.data_type = x::telem::FLOAT64_T;
        chans.push_back(std::move(ch));
    }
    const std::set<synnax::channel::Key> idx{100};
    std::vector<double> buf(N_CH * n, 1.5);
    x::telem::Frame fr(0);
    for (auto _: st) {
        fr.clear();
        driver::common::initialize_frame(fr, chans, idx, n);
        driver::common::transfer_buf(buf, fr, N_CH, n);
        auto &ix = fr.series->at(N_CH);
        ix.clear();
        for (size_t i = 0; i < n; i++)
            ix.write(static_cast<int64_t>(i));
        benchmark::DoNotOptimize(fr.series->data());
    }
}
BENCHMARK(BM_SourceFillFrame)->Arg(1)->Arg(40)->Arg(400);

static void BM_WriterEncode(benchmark::State &st) {
    const size_t n = st.range(0);
    auto codec = make_codec();
    auto fr = make_frame(n);
    std::vector<uint8_t> out;
    grpc::framer::WriterRequest req;
    std::string wire;
    for (auto _: st) {
        benchmark::DoNotOptimize(codec.encode(fr, out));
        req.set_command(1);
        req.set_buffer(out.data(), out.size());
        wire.clear();
        benchmark::DoNotOptimize(req.SerializeToString(&wire));
        benchmark::DoNotOptimize(wire.data());
    }
}
BENCHMARK(BM_WriterEncode)->Arg(1)->Arg(40)->Arg(400);

static void BM_CodecEncodeOnly(benchmark::State &st) {
    const size_t n = st.range(0);
    auto codec = make_codec();
    auto fr = make_frame(n);
    std::vector<uint8_t> out;
    for (auto _: st)
        benchmark::DoNotOptimize(codec.encode(fr, out));
}
BENCHMARK(BM_CodecEncodeOnly)->Arg(1)->Arg(400);

static void BM_StreamerDecode(benchmark::State &st) {
    const size_t n = st.range(0);
    auto codec = make_codec();
    auto fr = make_frame(n);
    std::vector<uint8_t> out;
    benchmark::DoNotOptimize(codec.encode(fr, out));
    for (auto _: st) {
        auto [f, err] = codec.decode(out.data(), out.size());
        benchmark::DoNotOptimize(f.series->data());
    }
}
BENCHMARK(BM_StreamerDecode)->Arg(1)->Arg(400);

static void BM_ReceiveCopy(benchmark::State &st) {
    grpc::framer::StreamerResponse src;
    src.set_buffer(std::string(st.range(0), 'x'));
    for (auto _: st) {
        grpc::framer::StreamerResponse res = src;
        std::pair<grpc::framer::StreamerResponse, x::errors::Error> p{
            res,
            x::errors::NIL
        };
        benchmark::DoNotOptimize(p.first.buffer().data());
    }
}
BENCHMARK(BM_ReceiveCopy)->Arg(64)->Arg(36000);

static void BM_ReceiveMove(benchmark::State &st) {
    grpc::framer::StreamerResponse src;
    src.set_buffer(std::string(st.range(0), 'x'));
    for (auto _: st) {
        grpc::framer::StreamerResponse res = src;
        std::pair<grpc::framer::StreamerResponse, x::errors::Error> p{
            std::move(res),
            x::errors::NIL
        };
        benchmark::DoNotOptimize(p.first.buffer().data());
    }
}
BENCHMARK(BM_ReceiveMove)->Arg(64)->Arg(36000);

static void BM_StatusClearNoWarning(benchmark::State &st) {
    auto ctx = std::make_shared<driver::task::MockContext>(nullptr);
    synnax::task::Task task;
    task.name = "probe";
    driver::common::StatusHandler h(ctx, task);
    for (auto _: st)
        h.clear_warning();
}
BENCHMARK(BM_StatusClearNoWarning);

static void BM_StatusPersistentWarning(benchmark::State &st) {
    auto ctx = std::make_shared<driver::task::MockContext>(nullptr);
    synnax::task::Task task;
    task.name = "probe";
    driver::common::StatusHandler h(ctx, task);
    const auto &w = driver::common::SKEW_WARNING;
    driver::common::Warning logged;
    for (auto _: st) {
        benchmark::DoNotOptimize(w != logged);
        logged = w;
        h.send_warning(w.message, w.description);
    }
}
BENCHMARK(BM_StatusPersistentWarning);

static void BM_TareNoop(benchmark::State &st) {
    std::vector<synnax::channel::Channel> chans(N_CH);
    for (size_t c = 0; c < N_CH; c++)
        chans[c].key = c + 1;
    driver::transform::Tare tare(chans);
    auto fr = make_frame(40);
    for (auto _: st)
        benchmark::DoNotOptimize(tare.transform(fr));
}
BENCHMARK(BM_TareNoop);
