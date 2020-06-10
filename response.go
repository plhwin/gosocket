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
	clientId string // client connection id
	client   ClientFace
	event    string
	id       string // client transparent id
	response
}

func NewResponse(client ClientFace, event, clientId, id string) (r *Response) {
	r = new(Response)
	r.client = client
	r.event = event
	r.clientId = clientId
	r.id = id
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
	if r.clientId == "" {
		r.client.Emit(r.event, r.response, r.id)
	} else {
		var args ArgsResponse
		args.Id = r.clientId
		args.Args = r.response
		r.client.Emit(r.event, args, r.id)
	}
}
