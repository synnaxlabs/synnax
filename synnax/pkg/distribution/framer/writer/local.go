package writer

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
)

func (s *Service) newGatewayWriter(cfg Config) (confluence.Segment[Request, Response], error) {
	w, err := s.TS.NewStreamWriter(storage.WriterConfig{Start: cfg.Start, Channels: cfg.Keys.Strings()})
	if err != nil {
		return nil, err
	}

	pipe := plumber.New()
	plumber.SetSegment[cesium.WriteRequest, cesium.WriteResponse](pipe, "writerClient", w)
	reqT := newRequestTranslator(s.HostResolver.HostID(), s.Logger)
	resT := newResponseTranslator()
	plumber.SetSegment[Request, cesium.WriteRequest](pipe, "requestTranslator", reqT)
	plumber.SetSegment[cesium.WriteResponse, Response](pipe, "responseTranslator", resT)
	plumber.MustConnect(pipe, "requestTranslator", "writerClient", 1)
	plumber.MustConnect(pipe, "writerClient", "responseTranslator", 1)
	seg := &plumber.Segment[Request, Response]{Pipeline: pipe}
	lo.Must0(seg.RouteInletTo("requestTranslator"))
	lo.Must0(seg.RouteOutletFrom("responseTranslator"))
	return seg, nil
}
