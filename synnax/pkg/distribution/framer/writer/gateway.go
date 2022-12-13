package writer

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
)

// newGateway opens a new StreamWriter that writes to the store on the gateway node.
func (s *Service) newGateway(cfg Config) (StreamWriter, error) {
	w, err := s.TS.NewStreamWriter(cfg.toStorage())
	if err != nil {
		return nil, err
	}
	pipe := plumber.New()
	plumber.SetSegment[cesium.WriteRequest, cesium.WriteResponse](pipe, "toStorage", w)
	reqT := &confluence.LinearTransform[Request, cesium.WriteRequest]{}
	reqT.Transform = newRequestTranslator()
	resT := &confluence.LinearTransform[cesium.WriteResponse, Response]{}
	resT.Transform = newResponseTranslator(s.HostResolver.HostID())
	plumber.SetSegment[Request, cesium.WriteRequest](pipe, "requests", reqT)
	plumber.SetSegment[cesium.WriteResponse, Response](pipe, "responses", resT)
	plumber.MustConnect[cesium.WriteRequest](pipe, "requests", "toStorage", 1)
	plumber.MustConnect[cesium.WriteResponse](pipe, "toStorage", "responses", 1)
	seg := &plumber.Segment[Request, Response]{Pipeline: pipe}
	lo.Must0(seg.RouteInletTo("requests"))
	lo.Must0(seg.RouteOutletFrom("responses"))
	return seg, nil
}
