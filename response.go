package gosocket

type ArgsRequest struct {
	Id   string      `json:"id"`
	Args interface{} `json:"args"`
}

type ArgsResponse struct {
	Id   string   `json:"id"`
	Args response `json:"args"`
}

type response struct {
	Result  bool        `json:"result"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type Response struct {
	identity string
	client   ClientFace
	event    string
	response
}

func NewResponse(client ClientFace, event, identity string) (r *Response) {
	r = new(Response)
	r.client = client
	r.event = event
	r.identity = identity
	return
}

func (r *Response) SetIdentity(identity string) {
	r.identity = identity
	return
}

func (r *Response) Set(result bool, message string, data interface{}) {
	r.Result = result
	r.Message = message
	r.Data = data
	return
}

func (r *Response) Success(data interface{}) {
	r.Set(true, "ok", data)
	r.Emit()
}

func (r *Response) Fail(message string) {
	r.Set(false, message, nil)
	r.Emit()
}

func (r *Response) Emit() {
	if r.identity == "" {
		r.client.Emit(r.event, r.response)
	} else {
		var args ArgsResponse
		args.Id = r.identity
		args.Args = r.response
		r.client.Emit(r.event, args)
	}
}
