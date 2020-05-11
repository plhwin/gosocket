package gosocket

type Request struct {
	Id   string      `json:"id"`
	Args interface{} `json:"args"`
}

type Response struct {
	identity string
	client   ClientFace
	event    string
	response
}

type response struct {
	Result  bool        `json:"result"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func NewResponse(client ClientFace, event, identity string) (r *Response) {
	r = new(Response)
	r.client = client
	r.event = event
	r.identity = identity
	return
}

func (r *Response) Set(message string, result bool, data interface{}) {
	r.Result = result
	r.Message = message
	r.Data = data
	return
}

func (r *Response) Success(data interface{}) {
	r.Set("ok", true, data)
	r.emit()
}

func (r *Response) Fail(message string) {
	r.Set(message, false, nil)
	r.emit()
}

func (r *Response) emit() {
	var req Request
	req.Id = r.identity
	req.Args = r.response
	r.client.Emit(r.event, req)
}
