package gosocket

type ArgsRequest struct {
	Id   string      `json:"id"`
	Args interface{} `json:"args"`
}

type ArgsResponse struct {
	Id   string   `json:"id"`
	Args Response `json:"args"`
}

type Response struct {
	Result  bool        `json:"result"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type response struct {
	identity string
	client   ClientFace
	event    string
	Response
}

func NewResponse(client ClientFace, event string) (r *response) {
	r = new(response)
	r.client = client
	r.event = event
	return
}

func (r *response) SetIdentity(identity string) {
	r.identity = identity
	return
}

func (r *response) Set(result bool, message string, data interface{}) {
	r.Result = result
	r.Message = message
	r.Data = data
	return
}

func (r *response) Success(data interface{}) {
	r.Set(true, "ok", data)
	r.Emit()
}

func (r *response) Fail(message string) {
	r.Set(false, message, nil)
	r.Emit()
}

func (r *response) Emit() {
	if r.identity == "" {
		r.client.Emit(r.event, r.Response)
	} else {
		var args ArgsResponse
		args.Id = r.identity
		args.Args = r.Response
		r.client.Emit(r.event, args)
	}
}
