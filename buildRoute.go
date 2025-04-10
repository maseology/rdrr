package rdrr

func (ev *Evaluator) buildRoute() []*route {
	r := make([]*route, len(ev.Dsws))
	for k, d := range ev.Dsws {
		r[k] = &route{
			conv: *ev.Schannel[k],
			sds:  d,
		}
	}
	return r
}
