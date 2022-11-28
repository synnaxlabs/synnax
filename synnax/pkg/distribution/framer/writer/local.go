package writer

import (
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
)

func newLocalWriter(keys channel.Keys, cfg Config) (confluence.Segment[Request, Response], error) {
	w, err := cfg.TS.NewStreamWriter(storage.WriterConfig{
		Start:    cfg.Start,
		Channels: keys.Strings(),
	})
	if err != nil {
		return nil, err
	}
	pipe := plumber.New()
	plumber.SetSegment[cesium.WriteRequest, cesium.WriteResponse](pipe, "writerClient", w)
	reqT := newRequestTranslator(cfg.HostResolver.HostID(), cfg.Logger)
	resT := newResponseTranslator()
	plumber.SetSegment[Request, cesium.WriteRequest](pipe, "requestTranslator", reqT)
	plumber.SetSegment[cesium.WriteResponse, Response](pipe, "responseTranslator", resT)
	plumber.UnaryRouter[cesium.WriteRequest]{
		SourceTarget: "requestTranslator",
		SinkTarget:   "writerClient",
	}.MustRoute(pipe)
	plumber.UnaryRouter[cesium.WriteResponse]{
		SourceTarget: "writerClient",
		SinkTarget:   "responseTranslator",
	}.MustRoute(pipe)
	seg := &plumber.Segment[Request, Response]{Pipeline: pipe}
	if err := seg.RouteInletTo("requestTranslator"); err != nil {
		panic(err)
	}
	if err := seg.RouteOutletFrom("responseTranslator"); err != nil {
		panic(err)
	}
	return seg, nil
}
