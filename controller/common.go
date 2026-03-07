package controller

import "GopherAI/common/code"

type Response struct {
	Code code.Code      `json:"code"`
	Msg  string         `json:"msg"`
	Data []interface{} `json:"data,omitempty"`
}

func (r *Response) CodeOf(code code.Code) Response {
	if nil == r {
		r = new(Response)
	}
	r.Code = code
	r.Msg = code.Msg()
	return *r
}

func (r *Response) Success() {
	r.CodeOf(code.CodeSuccess)
}